// Package supplierclient provides a client for the supplier
package supplierclient

import (
	"context"
	"net/http"
	"time"

	"koctz/internal/services"

	"go.uber.org/zap"
)

type IssueRequest struct {
	SKU       string `json:"sku"`
	RequestID string `json:"request_id"`
}

type IssueResponse struct {
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

type Client struct {
	name       string
	httpClient *http.Client
	urlA       string
	urlB       string
	timeout    time.Duration
	log        *zap.Logger
}

func NewSPLC(urlA, urlB string, timeout time.Duration, log *zap.Logger) (services.Service, error) {
	return &Client{
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

func (c *Client) GetName() string {
	return c.name
}

func (c *Client) Close(ctx context.Context) error {
	return nil
}
