// Package orders orders.go provides services.Service implementation
// and registred endpoints for orders service
package orders

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"koctz/internal/db"
	"koctz/internal/services"

	"go.uber.org/zap"
)

type ordersservice struct {
	name       string
	ctxTimeout time.Duration
	log        *zap.Logger
	db         db.OrdersRepository
}

func NewOS(mux *http.ServeMux, log *zap.Logger, ctxTimeout time.Duration) (services.Service, error) {
	const op = "orders.NewOS"

	db, err := db.NewOrdersPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: get orders db: %w", op, err)
	}

	oss := &ordersservice{
		name:       "orders_service",
		ctxTimeout: ctxTimeout,
		log:        log,
		db:         db,
	}

	oss.registerRoutes(mux)

	log.Debug("Successfully created orders service",
		zap.String("op", op))

	return oss, nil
}

func (s *ordersservice) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/orders", s.NewOrder)
	mux.HandleFunc("/api/{id}", s.GetOrder)
}

func (s *ordersservice) GetName() string {
	return s.name
}

func (s *ordersservice) Close(ctx context.Context) error {
	return nil
}
