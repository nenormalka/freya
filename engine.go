package freya

import (
	"context"
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/chapsuk/grace"
	sentry2 "github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	apm2 "go.elastic.co/apm/v2"
	"go.uber.org/dig"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/apm"
	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/freya/http"
	"github.com/nenormalka/freya/logger"
	"github.com/nenormalka/freya/sentry"
	"github.com/nenormalka/freya/types"
)

const (
	flushTTL = 2 * time.Second
)

type Engine struct {
	container *dig.Container
}

var defaultModules = types.Module{
	{CreateFunc: ServiceAdapter},
	{CreateFunc: NewApp},
	{CreateFunc: types.NewServicePool},
	{CreateFunc: types.NewServerPool},
	{CreateFunc: NewShutdownContext},
	{CreateFunc: logger.NewLogger},
}.
	Append(config.Module).
	Append(http.Module).
	Append(grpc.Module).
	Append(apm.Module).
	Append(sentry.Module).
	Append(conns.Module)

func NewShutdownContext() context.Context {
	return grace.ShutdownContext(context.Background())
}

func NewEngine(modules types.Module) *Engine {
	e := &Engine{
		container: dig.New(),
	}

	e.provide(append(defaultModules, modules...))

	godotenv.Overload()

	return e
}

func (e *Engine) Run() {
	if err := e.container.Invoke(e.mainFunc()); err != nil {
		log.Fatal("invoke err", err.Error())
	}
}

func (e *Engine) provide(m types.Module) {
	for _, c := range m {
		if err := e.container.Provide(c.CreateFunc, c.Options...); err != nil {
			log.Fatal("provide err ", err.Error())
		}
	}
}

func (e *Engine) mainFunc() interface{} {
	return func(
		ctx context.Context,
		app types.App,
		logger *zap.Logger,
		tracer *apm2.Tracer,
		conns *conns.Conns,
		sentryHub *sentry2.Hub,
	) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic", zap.Error(fmt.Errorf("recover panic %v", r)))
			}

			conns.Close()

			if err := logger.Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) && !errors.Is(err, syscall.EINVAL) {
				logger.Error("can not stop logger", zap.Error(err))
			}

			abortCh := make(chan struct{})
			defer close(abortCh)
			go func() {
				<-time.After(flushTTL)
				abortCh <- struct{}{}
			}()

			tracer.Flush(abortCh)
			sentryHub.Flush(flushTTL)

			logger.Info("container stopped")
		}()

		if err := app.Run(ctx); err != nil {
			logger.Error("failed run app", zap.Error(err))
		}
	}
}
