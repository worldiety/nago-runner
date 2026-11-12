// Copyright (c) 2025 worldiety GmbH
//
// This file is part of the NAGO Low-Code Platform.
// Licensed under the terms specified in the LICENSE file.
//
// SPDX-License-Identifier: Custom-License

// Package instlock serializes the operations on a single instance which stop, replace or start it. A restore
// stops an instance and replaces its data, thus the configuration apply must never restart, update or remove the
// same instance at the same time.
package instlock

import (
	"strings"
	"sync"
)

var (
	mutex sync.Mutex
	cond  = sync.NewCond(&mutex)
	held  = map[string]bool{}
)

func key(id string) string {
	// the service name is the lower-cased instance id
	return strings.ToLower(id)
}

// Lock waits until the instance is free and locks it.
func Lock(id string) {
	k := key(id)
	mutex.Lock()
	defer mutex.Unlock()

	for held[k] {
		cond.Wait()
	}

	held[k] = true
}

// TryLock locks the instance, if it is free and returns false otherwise.
func TryLock(id string) bool {
	k := key(id)
	mutex.Lock()
	defer mutex.Unlock()

	if held[k] {
		return false
	}

	held[k] = true
	return true
}

// Unlock frees the instance.
func Unlock(id string) {
	k := key(id)
	mutex.Lock()
	defer mutex.Unlock()

	delete(held, k)
	cond.Broadcast()
}
