// Package orders handler.go implements logic
// for orders-service endpoints
package orders

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"koctz/internal/db"
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

type reconcileResp struct {
	TotalPending int        `json:"total_pending"`
	Orders       []db.Order `json:"orders"`
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

	orderID := generateOrderID()
	order := &db.Order{
		ID:        orderID,
		SKU:       req.Sku,
		Status:    "created",
		CreatedAt: time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	if err := s.db.CreateOrder(ctx, order); err != nil {
		http.Error(w, fmt.Sprintf("%s: create order: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := orderResp{
		OrderID: order.ID,
		Sku:     order.SKU,
		Status:  order.Status,
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

	orderID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	order, err := s.db.GetOrderByID(ctx, orderID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("%s: get order: %s", op, err.Error()), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("%s: get order: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := orderResp{
		OrderID: order.ID,
		Sku:     order.SKU,
		Status:  order.Status,
		Code:    order.Code,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("%s: encode response: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}
}

func (s *ordersservice) Reconcile(w http.ResponseWriter, r *http.Request) {
	const op = "orders.Reconcile"

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	pendingOrders, err := s.db.GetPendingOrders(ctx, 100)
	if err != nil {
		http.Error(w, fmt.Sprintf("%s: get pending orders: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := reconcileResp{
		TotalPending: len(pendingOrders),
		Orders:       pendingOrders,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("%s: encode response: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}
}

func generateOrderID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "ord_" + hex.EncodeToString(b)
}
