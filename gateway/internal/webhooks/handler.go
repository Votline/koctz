// Package webhooks handler.go implements logic
// for webhooks-service endpoints
package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"koctz/internal/db"

	"go.uber.org/zap"
)

type webhookReq struct {
	EventID   string    `json:"event_id"`
	OrderID   string    `json:"order_id"`
	Status    string    `json:"status"`
	Amount    int       `json:"amount"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type webhookResp struct {
	Status string `json:"status"`
}

func (s *webhooksservice) Payment(w http.ResponseWriter, r *http.Request) {
	const op = "payment.Payment"

	var req webhookReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("%s: decode request: %s", op, err.Error()), http.StatusBadRequest)
		return
	}

	if req.EventID == "" || req.OrderID == "" {
		http.Error(w, fmt.Sprintf("%s: parse request: empty event or order id", op), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	var paymentFailed bool
	if err := s.odb.ProcessPayment(ctx, req.EventID, req.OrderID, func(ord *db.Order) error {
		if ord.Status != "created" && ord.Status != db.OrderStatusPending {
			return nil
		}

		if req.Status == "failed" {
			ord.Status = db.OrderStatusFailed
			paymentFailed = true
			for i := range ord.Items {
				ord.Items[i].Status = db.OrderStatusFailed
			}
			return nil
		} else {
			if err := s.hdb.SaveEvent(ctx, &db.OrderHistoryEvent{
				OrderID:   req.OrderID,
				Status:    "paid",
				EventType: "paid",
				Amount:    float64(req.Amount),
			}); err != nil {
				s.log.Error("failed to save history", zap.String("op", op), zap.Error(err))
			}
		}

		ord.Status = "paid"
		for i := range ord.Items {
			ord.Items[i].Status = "paid"
		}
		return nil
	}); err != nil {
		if strings.Contains(err.Error(), "already processed") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"already processed"}`))
			return
		}

		http.Error(w, fmt.Sprintf("%s: process payment: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	if paymentFailed {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"payment failed"}`))
		return
	}

	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer bgCancel()

		ord, err := s.odb.GetOrderByID(bgCtx, req.OrderID)
		if err != nil {
			s.log.Error("failed to get order for delivery", zap.Error(err))
			return
		}

		var totalPaid, totalDelivered float64

		for i := range ord.Items {
			item := &ord.Items[i]
			totalPaid += item.Price

			if item.Status == "delivered" {
				totalDelivered += item.Price
				continue
			}

			requestID := fmt.Sprintf("req-%s-%s", req.EventID, item.ID)
			code, err := s.splc.IssueKey(bgCtx, item.SKU, requestID)

			if err != nil {
				item.Status = db.ItemStatusRefund
				s.log.Error("failed to issue key", zap.Error(err), zap.String("item_id", item.ID))
			} else {
				item.Status = "delivered"
				item.Code = code
				totalDelivered += item.Price
			}
		}

		if totalDelivered == totalPaid {
			ord.Status = db.OrderStatusCompleted
		} else if totalDelivered > 0 {
			ord.Status = db.OrderStatusPartiallyRefund
		} else {
			ord.Status = db.OrderStatusFailed
		}

		if ord.Status == db.OrderStatusPartiallyRefund || ord.Status == db.OrderStatusCompleted {
			if err := s.hdb.SaveEvent(bgCtx, &db.OrderHistoryEvent{
				OrderID:   ord.ID,
				Status:    ord.Status,
				EventType: ord.Status,
				Amount:    totalDelivered,
			}); err != nil {
				s.log.Error("failed to save history", zap.String("op", op), zap.Error(err))
			}
		}

		if err := s.odb.UpdateOrderDelivery(bgCtx, ord); err != nil {
			s.log.Error("failed to update order delivery status", zap.Error(err))
			return
		}

		s.log.Info("order delivery finalized", zap.String("order_id", ord.ID), zap.String("status", ord.Status))
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
