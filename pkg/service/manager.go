package service

import (
	"context"
)

// Factory is the interface for creating new Manager objects.
type Factory interface {
	NewService(ctx context.Context) (Manager, error)
}

// Manager holds the state information for talking to
// splunk Service backend.
type Manager interface {
}
