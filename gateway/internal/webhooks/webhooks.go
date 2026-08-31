// Package webhooks webhooks.go provides services.Service implementation
// and registred endpoints for webhooks service
package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"koctz/internal/db"
	"koctz/internal/services"
	supplierclient "koctz/internal/suppliersclient"

	"go.uber.org/zap"
)

type webhooksservice struct {
	name       string
	ctxTimeout time.Duration
	log        *zap.Logger
	pdb        db.PaymentRepository
	odb        db.OrdersRepository
	kdb        db.KeysRepository
	splc       supplierclient.Client
}

func NewWBH(mux *http.ServeMux, log *zap.Logger, ctxTimeout time.Duration, supClt supplierclient.Client) (services.Service, error) {
	const op = "webhooks.NewWBH"

	pdb, err := db.NewPaymentsPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: create pdb: %w", op, err)
	}

	odb, err := db.NewOrdersPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: create odb: %w", op, err)
	}

	kdb, err := db.NewKeysPsql(log)
	if err != nil {
		return nil, fmt.Errorf("%s: create kdb: %w", op, err)
	}

	oss := &webhooksservice{
		name:       "webhooks_service",
		ctxTimeout: ctxTimeout,
		log:        log,
		pdb:        pdb,
		odb:        odb,
		kdb:        kdb,
		splc:       supClt,
	}

	oss.registerRoutes(mux)

	log.Debug("Successfully created webhooks service",
		zap.String("op", op))

	return oss, nil
}

func (s *webhooksservice) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/webhook/payment", s.Payment)
}

func (s *webhooksservice) GetName() string {
	return s.name
}

func (s *webhooksservice) Close(ctx context.Context) error {
	return nil
}
