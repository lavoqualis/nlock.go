// SPDX-FileCopyrightText: 2023 KåPI Tvätt AB <peter.magnusson@rikstvatt.se>
//
// SPDX-License-Identifier: MIT License

package nlock

import (
	"errors"
	"sync"

	"github.com/nats-io/nats.go"
)

type Manager struct {
	kv    nats.KeyValue
	locks map[string]*Lock
	name  string
	mu    sync.Mutex
}

func New(name string, kv nats.KeyValue) (*Manager, error) {
	return &Manager{
		kv:    kv,
		name:  name,
		locks: make(map[string]*Lock),
	}, nil
}

func (lm *Manager) createLock(key string) (*Lock, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	if _, ok := lm.locks[key]; ok {
		return nil, errors.New("lock already exists")
	}
	l, err := newLock(lm.kv, lm.name, key)
	if err != nil {
		return nil, err
	}
	l.onRelease = lm.removeLock
	l.onAcquired = func(l *Lock) {
		logger.Info("lock acquired %s", l.name)
	}
	lm.locks[key] = l
	return l, nil
}

func (lm *Manager) removeLock(key string) {
	logger.Debug("removing lock '%s' from manager %s", key, lm.name)
	lm.mu.Lock()
	defer lm.mu.Unlock()
	//TODO: any callbacks perhaps?
	delete(lm.locks, key)
}

func (lm *Manager) Claim(key string) (*Lock, error) {
	l, err := lm.createLock(key)
	if err != nil {
		return l, err
	}
	return l, nil
}
