package saia

import (
	//"context"

	"context"

	"github.com/go-logr/logr"
	manager "github.com/splunk/splunk-ai-operator/pkg/service"
	//"github.com/splunk/splunk-ai-operator/pkg/service/saia"
)

type saiaManagerFactory struct {
	log logr.Logger
}

// NewManagerFactory  new manager factory to create manager interface
func NewManagerFactory() manager.Factory {
	factory := saiaManagerFactory{}
	err := factory.init()
	if err != nil {
		return nil // FIXME we have to throw some kind of exception or error here
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
// TODO: Replace the parameters and return type with the actual signature from the manager.Factory interface.

// NewGateway returns a new Splunk Gateway using global
// configuration for finding the Splunk services.
func (f *saiaManagerFactory) NewService(ctx context.Context) (manager.Manager, error) {
	return f.newManager(ctx)
}
