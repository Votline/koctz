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

	var orderSKU string
	if err := s.odb.ProcessPayment(ctx, req.EventID, req.OrderID, func(ord *db.Order) error {
		orderSKU = ord.SKU
		if ord.Status != "created" {
			return nil
		}

		if req.Status == "failed" {
			ord.Status = "payment_failed"
			return nil
		}

		ord.Status = "paid"
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

	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer bgCancel()

		requestID := fmt.Sprintf("req-%s", req.EventID)

		code, err := s.splc.IssueKey(bgCtx, orderSKU, requestID)

		var finalStatus string
		var finalCode string

		if err != nil {
			if strings.Contains(err.Error(), "out of stock") {
				finalStatus = "out_of_stock"
			} else {
				finalStatus = "delivery_failed"
			}
			s.log.Error("failed to issue key from suppliers",
				zap.Error(err),
				zap.String("order_id", req.OrderID))
		} else {
			finalStatus = "delivered"
			finalCode = code
		}

		err = s.odb.ProcessPayment(bgCtx, fmt.Sprintf("delivery-%s", req.EventID), req.OrderID, func(ord *db.Order) error {
			ord.Status = finalStatus
			ord.Code = finalCode
			return nil
		})
		if err != nil {
			s.log.Error("failed to update order final status",
				zap.Error(err),
				zap.String("order_id", req.OrderID))
		}
	}()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
