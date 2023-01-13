// SPDX-FileCopyrightText: 2023 KåPI Tvätt AB <peter.magnusson@rikstvatt.se>
//
// SPDX-License-Identifier: MIT License

package nlock

import (
	"errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type Lock struct {
	name        string
	kv          nats.KeyValue
	onRelease   func(key string)
	onLost      func(key string)
	onAcquired  func(l *Lock)
	rev         uint64
	managerName string
	ttl         time.Duration
	state       string
	stopRefresh chan struct{}
	stopWatch   chan struct{}
	w           nats.KeyWatcher
}

func newLock(kv nats.KeyValue, managerName, name string) (*Lock, error) {
	if name == "" {
		return nil, errors.New("key can not be empty")
	}
	if kv == nil {
		return nil, errors.New("kv can not be nil")
	}
	if managerName == "" {
		return nil, errors.New("manager name can not be empty")
	}
	l := &Lock{
		name:        name,
		kv:          kv,
		ttl:         30 * time.Second,
		state:       "new",
		managerName: managerName,
	}
	logger.Debug("lock '%s' created", name)

	if err := l.register(); err != nil {
		logger.Warning("failed to register %s: %v", l.name, err)
	}
	go l.watch()
	return l, nil
}

func (l *Lock) setState(v string) {
	logger.Debug("changing state from %s to %s", l.state, v)
	l.state = v
}

func (l *Lock) register() error {
	oldstate := l.state
	if l.state == "registering" {
		return errors.New("already registering")
	}
	logger.Debug("registering '%s'", l.name)
	l.setState("registering")
	defer func() {
		go l.refresh()
	}()
	rev, err := l.kv.Create(l.name, []byte(l.managerName))
	if err != nil {
		logger.Warning("failed to register %s: %v", l.name, err)
		l.setState(oldstate)
		return err
	}
	l.setState("registerd")
	l.rev = rev
	logger.Debug("registerd '%s' with rev %d", l.name, rev)
	if l.onAcquired != nil {
		l.onAcquired(l)
	}
	return nil
}

func (l *Lock) refresh() {
	if l.stopRefresh != nil {
		return
	}
	timeout := l.ttl / 2
	logger.Debug("refresh started for '%s' with %s timeout", l.name, timeout)

	l.stopRefresh = make(chan struct{})
	ticker := time.NewTicker(timeout)
	bvalue := []byte(l.managerName)

	defer func() {
		l.stopRefresh = nil
	}()

	for {
		select {
		case <-ticker.C:
			logger.Debug("on tick. state=%s, rev=%d, managerName=%s", l.state, l.rev, l.managerName)
			switch l.state {
			case "registerd":
				if nrev, err := l.kv.Update(l.name, bvalue, l.rev); err != nil {
					logger.Warning("update error for %s: %v", l.name, err)
					l.lost()
				} else {
					// logger.Debug("refreshed '%s' with rev %d", l.name, l.rev)
					l.rev = nrev
				}
			case "lost", "new":
				if err := l.register(); err != nil {
					logger.Warning("error while register: %v", err)
				}
			default:
				logger.Debug("unhandled state %s", l.state)
			}
		case <-l.stopRefresh:
			return
		}
	}
}

func (l *Lock) watch() {
	if l.w != nil {
		logger.Warning("watcher for %s is already running", l.name)
		return
	}
	l.stopWatch = make(chan struct{})
	var err error
	l.w, err = l.kv.Watch(l.name)

	if err != nil {
		logger.Error("error getting watcher %s: %v", l.name, err)
		return
	}
	for {
		select {
		case kve := <-l.w.Updates():
			if kve != nil {
				switch kve.Operation() {
				case nats.KeyValueDelete:
					logger.Debug("delete for '%s' with state '%s'. value=%s", l.name, l.state, string(kve.Value()))
					if !strings.HasPrefix(l.state, "rel") {
						l.register()
					}
					// default:
					// 	logger.Debug("%s @ %d -> %q (op: %s)\n", kve.Key(), kve.Revision(), string(kve.Value()), kve.Operation())
				}
			}
		case <-l.stopWatch:
			logger.Debug("stop watching '%s'", l.name)
			if l.w != nil {
				l.w.Stop()
				l.w = nil
			}
			return
		}
	}
}

func (l *Lock) deleteKV() {
	if l.rev != 0 {
		err := l.kv.Delete(l.name, nats.LastRevision(l.rev))
		if err != nil {
			logger.Info("could not delete '%s':%v", l.name, err)
		}
		l.rev = 0
	}
}

func (l *Lock) lost() {
	l.setState("lost")
	logger.Debug("lost key %s", l.name)
	l.deleteKV()
	l.rev = 0
	if l.onLost != nil {
		l.onLost(l.name)
	}
}

func (l *Lock) Release() {
	logger.Debug("releaseing '%s' from state %s", l.name, l.state)
	if l.state == "registerd" {
		l.setState("releasing")
	}
	if l.stopRefresh != nil {
		logger.Debug("closing refresh")
		close(l.stopRefresh)
		l.stopRefresh = nil
	}
	if l.stopWatch != nil {
		logger.Debug("closing watch")
		close(l.stopWatch)
		l.stopWatch = nil
	}
	l.setState("released")
	l.deleteKV()
	if l.onRelease != nil {
		l.onRelease(l.name)
	}
}

func (l *Lock) HasAcquisition() bool {
	return l.state == "registerd" && l.rev > 0
}
