// Package router router.go contains implementation 'Server'
// interface by 'HTTPServer' struct
// Creates and manages the main http server
package router

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"koctz/internal/orders"
	"koctz/internal/services"
	"koctz/internal/suppliers"
	supplierclient "koctz/internal/suppliersclient"
	"koctz/internal/webhooks"

	"go.uber.org/zap"
)

type Server interface {
	Init() error
	Start() error
	Shutdown(ctx context.Context) error
}

type HTTPServer struct {
	srv  *http.Server
	log  *zap.Logger
	svcs []services.Service
}

func (s *HTTPServer) Init(log *zap.Logger) error {
	const op = "router.Init"

	s.log = log
	handler, err := s.registerServices()
	if err != nil {
		return fmt.Errorf("%s: register: %w", op, err)
	}
	s.srv = &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	return nil
}

func (s *HTTPServer) Start() error {
	const op = "router.Start"

	s.log.Debug("Starting...",
		zap.String("op", op),
		zap.String("addr", s.srv.Addr))

	if err := s.srv.ListenAndServe(); err != nil {
		return fmt.Errorf("%s: listen and serve: %w", op, err)
	}
	return nil
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	const op = "router.Shutdown"

	s.log.Debug("Shutdowning http server...", zap.String("op", op))
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s: shutdown: %w", op, err)
	}

	return nil
}

func (s *HTTPServer) registerServices() (http.Handler, error) {
	const op = "router.registerServices"

	mux := http.NewServeMux()

	ctxTimeout := time.Duration(getEnvInt("CONTEXT_TIMEOUT", 10)) * time.Second

	oss, err := orders.NewOS(mux, s.log, ctxTimeout)
	if err != nil {
		return nil, fmt.Errorf("%s: create orders: %w", op, err)
	}

	spl, err := suppliers.NewSPL(mux, s.log, ctxTimeout)
	if err != nil {
		return nil, fmt.Errorf("%s: create suppliers: %w", op, err)
	}

	splc, err := supplierclient.NewSPLC(
		"http://localhost:8080/suppliers/a/issue",
		"http://localhost:8080/suppliers/b/issue",
		ctxTimeout, s.log)
	if err != nil {
		return nil, fmt.Errorf("%s: create suppliersclient: %w", op, err)
	}

	wbh, err := webhooks.NewWBH(mux, s.log, ctxTimeout, splc)
	if err != nil {
		return nil, fmt.Errorf("%s: create webhooks: %w", op, err)
	}

	s.svcs = append(s.svcs, oss, wbh, spl)

	return mux, nil
}

func getEnvInt(key string, def int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return def
	}
	valInt, err := strconv.Atoi(valStr)
	if err != nil {
		return def
	}
	return valInt
}
