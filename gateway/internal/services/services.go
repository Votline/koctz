// Package services provides service interfac and
// execution methods
package services

import (
	"context"
)

type Service interface {
	// GetName returns service name
	GetName() string

	// Close gracefully shutdowns service
	Close(ctx context.Context) error
}
