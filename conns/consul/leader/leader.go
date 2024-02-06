package leader

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	lilith "github.com/nenormalka/lilith/patterns"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/conns/consul/config"
	"github.com/nenormalka/freya/conns/consul/lock"
	"github.com/nenormalka/freya/conns/consul/session"
	"github.com/nenormalka/freya/conns/consul/watcher"
)

type (
	Leader struct {
		locker  *lock.Locker
		session *session.Session
		watcher *watcher.Watcher

		log       *zap.Logger
		stopCh    chan struct{}
		isLeader  bool
		mu        sync.RWMutex
		leaderTTL time.Duration
	}
)

func NewLeader(
	locker *lock.Locker,
	session *session.Session,
	watcher *watcher.Watcher,
	logger *zap.Logger,
	cfg config.Config,
) *Leader {
	return &Leader{
		locker:    locker,
		session:   session,
		watcher:   watcher,
		log:       logger,
		stopCh:    make(chan struct{}),
		isLeader:  false,
		mu:        sync.RWMutex{},
		leaderTTL: cfg.LeaderTTL,
	}
}

func (l *Leader) Start(ctx context.Context) error {
	var err error

	defer func() {
		if err != nil {
			if errS := l.Stop(ctx); errS != nil {
				l.log.Error("failed to stop leader", zap.Error(errS))
			}
		}
	}()

	if _, err = l.session.Create(ctx); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	if err = l.tryLock(ctx); err != nil {
		return fmt.Errorf("failed to try lock: %w", err)
	}

	if err = l.startWatch(ctx); err != nil {
		return fmt.Errorf("failed to start watch: %w", err)
	}

	l.startRenew(ctx)
	l.startChecker(ctx)

	return nil
}

func (l *Leader) Stop(ctx context.Context) error {
	var err error
	if err = l.session.Destroy(ctx); err != nil {
		err = errors.Join(err, fmt.Errorf("failed to destroy session: %w", err))
	}

	if _, errUn := l.locker.Release(ctx, l.session.SessionKey(), l.session.SessionID()); errUn != nil {
		err = errors.Join(err, fmt.Errorf("failed to release lock: %w", errUn))
	}

	if err = l.watcher.Stop(ctx); err != nil {
		err = errors.Join(err, fmt.Errorf("failed to stop watcher: %w", err))
	}

	close(l.stopCh)

	return err
}

func (l *Leader) IsLeader() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.isLeader
}

func (l *Leader) tryLock(ctx context.Context) error {
	isLeader, err := l.locker.Acquire(ctx, l.session.SessionKey(), l.session.SessionID())
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	l.mu.Lock()
	l.isLeader = isLeader
	l.mu.Unlock()

	return nil
}

func (l *Leader) startWatch(ctx context.Context) error {
	if err := l.watcher.WatchKeys(watcher.WatchKeys{
		l.session.SessionKey(): func(_, sessionID string, _ []byte) {
			if sessionID == l.session.SessionID() {
				l.mu.Lock()
				l.isLeader = true
				l.mu.Unlock()

				return
			}

			if err := l.tryLock(ctx); err != nil {
				l.log.Error("failed to try lock", zap.Error(err))
			}
		},
	}); err != nil {
		return fmt.Errorf("failed to watch keys: %w", err)
	}

	if err := l.watcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	return nil
}

func (l *Leader) startRenew(ctx context.Context) {
	errCh := l.session.Renew(ctx)
	go func() {
		for {
			select {
			case <-l.stopCh:
				return
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil && errors.Is(err, context.Canceled) {
					l.log.Error("failed to renew session", zap.Error(err))
				}
			}
		}
	}()
}

func (l *Leader) startChecker(ctx context.Context) {
	lilith.Ticker(ctx, l.leaderTTL, func() {
		sessionID, err := l.locker.KeyOwner(ctx, l.session.SessionKey())
		if err != nil {
			l.log.Error("failed to get key owner", zap.Error(err))
			return
		}

		if sessionID != "" {
			l.mu.Lock()
			l.isLeader = sessionID == l.session.SessionID()
			l.mu.Unlock()

			return
		}

		if err = l.tryLock(ctx); err != nil {
			l.log.Error("failed to try lock", zap.Error(err))

			return
		}
	})
}
