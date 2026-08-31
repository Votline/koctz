// Package orders orders.go provides services.Service implementation
// and registred endpoints for orders service
package orders

import (
	"context"
	"net/http"

	"koctz/internal/services"

	"go.uber.org/zap"
)

type ordersservice struct {
	name string
	log  *zap.Logger
}

func NewOS(mux *http.ServeMux, log *zap.Logger) (services.Service, error) {
	const op = "orders.NewOS"

	oss := &ordersservice{
		name: "orders_service",
		log:  log,
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
