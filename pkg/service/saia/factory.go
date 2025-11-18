package saia

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	manager "github.com/splunk/splunk-ai-operator/pkg/service"
)

type saiaManagerFactory struct {
	log logr.Logger
}

// NewManagerFactory creates a new manager factory to create manager interface.
// Returns nil if initialization fails - callers should check for nil.
func NewManagerFactory() manager.Factory {
	factory := saiaManagerFactory{}
	err := factory.init()
	if err != nil {
		// Log the error since we can't return it from this signature
		// In production, consider using a logger
		panic(fmt.Sprintf("failed to initialize SAIA manager factory: %v", err))
	}
	return &factory
}

func (f *saiaManagerFactory) init() error {
	return nil
}

func (f *saiaManagerFactory) newManager(ctx context.Context) (manager.Manager, error) {
	newManager := &saiaManager{
		log: f.log,
	}
	return newManager, nil
}

// NewService implements the Factory interface.
// Returns a new SAIA service manager using the provided context.
func (f *saiaManagerFactory) NewService(ctx context.Context) (manager.Manager, error) {
	return f.newManager(ctx)
}
