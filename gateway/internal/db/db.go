// Package db provides a database connection and
// interfaces for services
package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	OrderStatusPending         = "pending"
	OrderStatusProcessing      = "processing"
	OrderStatusCompleted       = "completed"
	OrderStatusPartiallyRefund = "partially_refund"
	OrderStatusFailed          = "failed"

	ItemStatusPending = "pending"
	ItemStatusSuccess = "success"
	ItemStatusFailed  = "failed"
	ItemStatusRefund  = "refunded"
)

type OrdersRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrderByID(ctx context.Context, id string) (*Order, error)
	GetPendingOrders(ctx context.Context, limit int) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, itemID, status, code string) error
	UpdateOrderDelivery(ctx context.Context, ord *Order) error
	ProcessPayment(ctx context.Context, eventID, orderID string, updateFn func(ord *Order) error) error
}

type HistoryRepository interface {
	SaveEvent(ctx context.Context, event *OrderHistoryEvent) error
	GetStateAt(ctx context.Context, orderID string, at time.Time) (*OrderHistoryEvent, error)
	GetReconciliation(ctx context.Context, from, to time.Time) (paid, delivered, refunded float64, isBalanced bool, err error)
}

type PaymentRepository interface {
	RegisterEvent(ctx context.Context, eventID, orderID, status string) error
}

type KeysRepository interface {
	GetByRequestID(ctx context.Context, requestID string) (*Key, error)
	ReserveAndIssueKey(ctx context.Context, sku, requestID string) (*Key, error)
}

type Order struct {
	ID        string      `json:"order_id" db:"id"`
	Items     []OrderItem `json:"items,omitempty" db:"-"`
	Status    string      `json:"status" db:"status"`
	Price     float64     `json:"price" db:"price"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
}

type OrderItem struct {
	ID        string    `json:"id" db:"id"`
	OrderID   string    `json:"order_id" db:"order_id"`
	SKU       string    `json:"sku" db:"sku"`
	Price     float64   `json:"price" db:"price"`
	Status    string    `json:"status" db:"status"`
	Code      string    `json:"code,omitempty" db:"code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type OrderHistoryEvent struct {
	ID        int64     `db:"id" json:"id,omitempty"`
	OrderID   string    `db:"order_id" json:"order_id,omitempty"`
	Status    string    `db:"status" json:"status,omitempty"`
	EventType string    `db:"event_type" json:"event_type,omitempty"`
	Amount    float64   `db:"amount" json:"amount"`
	CreatedAt time.Time `db:"created_at" json:"created_at,omitempty"`
}

type Key struct {
	ID        int    `db:"id"`
	SKU       string `db:"sku"`
	Code      string `db:"code"`
	Status    string `db:"status"`
	RequestID string `db:"request_id"`
}

var (
	instance *sqlx.DB
	once     sync.Once
)

func GetDB() (*sqlx.DB, error) {
	const op = "db.GetDB"
	var initerr error
	once.Do(func() {
		db, err := sqlx.Connect("postgres", os.Getenv("POSTGRES_URL"))
		if err != nil {
			initerr = fmt.Errorf("%s: failed to connect to database: %w", op, err)
			return
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)

		instance = db
	})

	if initerr != nil {
		return nil, initerr
	}

	return instance, nil
}
