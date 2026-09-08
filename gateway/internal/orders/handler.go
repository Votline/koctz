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

type newOrderItemReq struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

type newOrderReq struct {
	Items []newOrderItemReq `json:"items"`
}

type orderItemResp struct {
	ItemID string  `json:"item_id"`
	SKU    string  `json:"sku"`
	Status string  `json:"status"`
	Price  float64 `json:"price"`
	Code   string  `json:"code,omitempty"`
}

type orderResp struct {
	OrderID   string          `json:"order_id"`
	Status    string          `json:"status"`
	Price     float64         `json:"price"`
	Items     []orderItemResp `json:"items"`
	CreatedAt time.Time       `json:"created_at"`
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

	if len(req.Items) == 0 {
		http.Error(w, fmt.Sprintf("%s: parse request: nil sku", op), http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	orderID := generateOrderID()

	var totalPrice float64
	orderItems := make([]db.OrderItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		if itemReq.SKU == "" {
			http.Error(w, fmt.Sprintf("%s: parse request: empty sku in items", op), http.StatusBadRequest)
			return
		}

		totalPrice += itemReq.Price
		orderItems = append(orderItems, db.OrderItem{
			ID:        generateItemID(),
			OrderID:   orderID,
			SKU:       itemReq.SKU,
			Price:     itemReq.Price,
			Status:    "pending",
			CreatedAt: now,
		})
	}

	order := &db.Order{
		ID:        orderID,
		Status:    "created",
		Price:     totalPrice,
		Items:     orderItems,
		CreatedAt: now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.ctxTimeout)
	defer cancel()

	if err := s.db.CreateOrder(ctx, order); err != nil {
		http.Error(w, fmt.Sprintf("%s: create order: %s", op, err.Error()), http.StatusInternalServerError)
		return
	}

	respItems := make([]orderItemResp, 0, len(order.Items))
	for _, item := range order.Items {
		respItems = append(respItems, orderItemResp{
			ItemID: item.ID,
			SKU:    item.SKU,
			Status: item.Status,
			Price:  item.Price,
			Code:   item.Code,
		})
	}

	resp := orderResp{
		OrderID:   order.ID,
		Status:    order.Status,
		Price:     order.Price,
		Items:     respItems,
		CreatedAt: order.CreatedAt,
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

	respItems := make([]orderItemResp, 0, len(order.Items))
	for _, item := range order.Items {
		respItems = append(respItems, orderItemResp{
			ItemID: item.ID,
			SKU:    item.SKU,
			Status: item.Status,
			Price:  item.Price,
			Code:   item.Code,
		})
	}

	resp := orderResp{
		OrderID:   order.ID,
		Status:    order.Status,
		Price:     order.Price,
		Items:     respItems,
		CreatedAt: order.CreatedAt,
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

func generateItemID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "item_" + hex.EncodeToString(b)
}
