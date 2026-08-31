// Package webhooks handler.go implements logic
// for webhooks-service endpoints
package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"koctz/internal/db"
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

	if err := s.odb.ProcessPayment(ctx, req.EventID, req.OrderID, func(ord *db.Order) error {
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
		http.Error(w, fmt.Sprintf("%s: process payment: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
