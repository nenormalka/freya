package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nenormalka/freya/conns/consul/config"

	"github.com/hashicorp/consul/api"
)

var (
	ErrEmptySession = errors.New("empty session")
)

type (
	Session struct {
		cli        *api.Client
		sessionID  string
		sessionKey string
		ttl        string
		stopCh     chan struct{}
	}
)

func NewSession(cli *api.Client, cfg config.Config) *Session {
	return &Session{
		cli:        cli,
		sessionKey: fmt.Sprintf("service/%s/leader", cfg.ServiceName),
		ttl:        cfg.SessionTTL,
		stopCh:     make(chan struct{}),
	}
}

func (s *Session) Create(ctx context.Context) (string, error) {
	opt := &api.WriteOptions{}

	sessionID, _, err := s.cli.Session().CreateNoChecks(&api.SessionEntry{
		Name:      s.sessionKey,
		Behavior:  "delete",
		TTL:       s.ttl,
		LockDelay: 2 * time.Second,
	}, opt.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	s.sessionID = sessionID

	return sessionID, nil
}

func (s *Session) Destroy(ctx context.Context) error {
	if s.sessionID == "" {
		return ErrEmptySession
	}

	opt := &api.WriteOptions{}

	_, err := s.cli.Session().Destroy(s.sessionID, opt.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}

	close(s.stopCh)

	return nil
}

func (s *Session) Renew(ctx context.Context) <-chan error {
	errCh := make(chan error)
	go func() {
		if s.sessionID == "" {
			errCh <- ErrEmptySession

			return
		}

		opt := &api.WriteOptions{}

		if err := s.cli.Session().RenewPeriodic(
			s.ttl,
			s.sessionID,
			opt.WithContext(ctx),
			s.stopCh,
		); err != nil {
			select {
			case errCh <- fmt.Errorf("failed to renew session: %w", err):
			default:
			}
		}
	}()

	return errCh
}

func (s *Session) SessionID() string {
	return s.sessionID
}

func (s *Session) SessionKey() string {
	return s.sessionKey
}
