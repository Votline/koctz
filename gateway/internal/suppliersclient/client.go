// Package supplierclient provides a client for the supplier
package supplierclient

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Client interface {
	IssueKey(ctx context.Context, sku, requestID string) (string, error)
}

type SuplierClient struct {
	name       string
	httpClient *http.Client
	urlA       string
	urlB       string
	timeout    time.Duration
	log        *zap.Logger
}

func NewSPLC(urlA, urlB string, timeout time.Duration, log *zap.Logger) (Client, error) {
	return &SuplierClient{
		name: "supplierclient",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		urlA:    urlA,
		urlB:    urlB,
		timeout: timeout,
		log:     log,
	}, nil
}
