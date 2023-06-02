package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/nenormalka/freya/types"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type (
	Server struct {
		server *http.Server
		logger *zap.Logger
		cfg    Config
	}

	CustomServer interface {
		GetServerName() string
		StartServer(r *mux.Router) error
	}
)

func NewHTTP(config Config, logger *zap.Logger, customServerList CustomServerList) (*Server, error) {
	runtime.SetMutexProfileFraction(100)
	runtime.SetBlockProfileRate(100)

	r := mux.NewRouter()

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	r.Handle("/metrics", promhttp.Handler())

	r.Handle("/health", Handler(
		WithReleaseID(config.ReleaseID),
	))

	for _, customServer := range customServerList.CustomServers {
		logger.Info(fmt.Sprintf("register http server: `%s`", customServer.GetServerName()))
		if err := customServer.StartServer(r); err != nil {
			return nil, fmt.Errorf("set http routes error %w", err)
		}
	}

	return &Server{
		server: &http.Server{
			Handler:     r,
			Addr:        config.ListenAddr,
			IdleTimeout: config.KeepaliveTime + config.KeepaliveTimeout,
		},
		logger: logger,
		cfg:    config,
	}, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("HTTP server started, listening on address: ", zap.String("http start", s.cfg.ListenAddr))

	return types.StartServerWithWaiting(ctx, func(errCh chan error) {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("http server err", zap.Error(err))
			errCh <- err
		}
	})
}

func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
