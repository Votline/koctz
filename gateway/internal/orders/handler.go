// Package orders handler.go implements logic
// for orders-service endpoints
package orders

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type newOrderReq struct {
	Sku string `json:"sku"`
}

type orderResp struct {
	OrderID string `json:"order_id"`
	Sku     string `json:"sku"`
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
}

func (s *ordersservice) NewOrder(w http.ResponseWriter, r *http.Request) {
	const op = "orders.NewOrder"

	var req newOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("%s: decode request: %s", op, err.Error()), http.StatusBadRequest)
		return
	}

	if req.Sku == "" {
		http.Error(w, fmt.Sprintf("%s: parse request: nil sku", op), http.StatusBadRequest)
		return
	}

	resp := orderResp{
		OrderID: "ord_123",
		Sku:     req.Sku,
		Status:  "created",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("%s: encode response: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}
}

func (s *ordersservice) GetOrder(w http.ResponseWriter, r *http.Request) {
	const op = "orders.GetOrder"

	id := r.PathValue("id")

	resp := orderResp{
		OrderID: id,
		Sku:     "STEAM-TOPUP-500",
		Status:  "delivered",
		Code:    "LFXC-TNCS-BPCD",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("%s: encode response: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}
}
