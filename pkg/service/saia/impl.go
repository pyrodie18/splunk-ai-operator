/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
... (license text omitted for brevity) ...
*/

package saia

import "github.com/go-logr/logr"

// saiaManager implements the gateway.Gateway interface
// and uses gateway to manage the host.
type saiaManager struct {
	// a logger configured for this host
	log logr.Logger
}
