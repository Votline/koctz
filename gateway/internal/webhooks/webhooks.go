// Package webhooks webhooks.go provides services.Service implementation
// and registred endpoints for webhooks service
package webhooks

import (
	"context"
	"net/http"
	"time"

	"koctz/internal/services"

	"go.uber.org/zap"
)

type webhooksservice struct {
	name       string
	ctxTimeout time.Duration
	log        *zap.Logger
}

func NewWBH(mux *http.ServeMux, log *zap.Logger, ctxTimeout time.Duration) (services.Service, error) {
	const op = "webhooks.NewWBH"

	oss := &webhooksservice{
		name:       "webhooks_service",
		ctxTimeout: ctxTimeout,
		log:        log,
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
