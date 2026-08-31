// Package webhooks handler.go implements logic
// for webhooks-service endpoints
package webhooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

	resp := webhookResp{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("%s: encode response: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}
}
