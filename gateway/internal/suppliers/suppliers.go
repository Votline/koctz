// Package suppliers provides stub-endpoints as suppliers
package suppliers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"koctz/internal/db"
	"koctz/internal/services"

	"go.uber.org/zap"
)

type suppliersservice struct {
	name       string
	ctxTimeout time.Duration
	log        *zap.Logger
	kdb        db.KeysRepository
}

func NewSPL(mux *http.ServeMux, log *zap.Logger, ctxTimeout time.Duration) (services.Service, error) {
	const op = "suppliers.NewSuppliers"

	kdb, err := db.NewKeysPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: create kdb: %w", op, err)
	}

	svc := &suppliersservice{
		name:       "suppliers_service",
		ctxTimeout: ctxTimeout,
		log:        log,
		kdb:        kdb,
	}

	log.Debug("Successfully created suppliers service",
		zap.String("op", op))

	svc.registerRoutes(mux)
	return svc, nil
}

func (s *suppliersservice) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/suppliers/a/issue", s.handleSupplierA)
	mux.HandleFunc("/suppliers/b/issue", s.handleSupplierB)
}

func (s *suppliersservice) GetName() string {
	return s.name
}

func (s *suppliersservice) Close(ctx context.Context) error {
	return nil
}
