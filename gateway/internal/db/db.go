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

type OrdersRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrderByID(ctx context.Context, id string) (*Order, error)
	GetPendingOrders(ctx context.Context, limit int) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, orderID, status, code string) error
	ProcessPayment(ctx context.Context, eventID, orderID string, updateFn func(ord *Order) error) error
}

type PaymentRepository interface {
	RegisterEvent(ctx context.Context, eventID, orderID, status string) error
}

type KeysRepository interface {
	GetByRequestID(ctx context.Context, requestID string) (*Key, error)
	ReserveAndIssueKey(ctx context.Context, sku, requestID string) (*Key, error)
}

type Order struct {
	ID        string    `json:"order_id" db:"id"`
	SKU       string    `json:"sku" db:"sku"`
	Status    string    `json:"status" db:"status"`
	Code      string    `json:"code,omitempty" db:"code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
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
