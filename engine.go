package freya

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	sentry2 "github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	apm2 "go.elastic.co/apm/v2"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"

	"github.com/nenormalka/freya/apm"
	"github.com/nenormalka/freya/communication"
	"github.com/nenormalka/freya/config"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/grpc"
	"github.com/nenormalka/freya/http"
	"github.com/nenormalka/freya/logger"
	"github.com/nenormalka/freya/sentry"
	"github.com/nenormalka/melissa"
	"github.com/nenormalka/melissa/types"
)

const (
	flushTTL = 2 * time.Second
)

type (
	Engine struct {
		engine *melissa.Engine
	}
)

var defaultModules = types.Module{
	{CreateFunc: logger.NewLogger},
}.
	Append(config.Module).
	Append(http.Module).
	Append(grpc.Module).
	Append(apm.Module).
	Append(sentry.Module).
	Append(conns.Module).
	Append(communication.Module)

func NewEngine(modules types.Module) *Engine {
	e := &Engine{}
	e.engine = melissa.NewEngine(e.mainFunc(), modules.Append(defaultModules))

	return e
}

func (e *Engine) Run() {
	godotenv.Overload()
	e.engine.Run()
}

func (e *Engine) mainFunc() any {
	return func(
		ctx context.Context,
		app *melissa.App,
		logger *zap.Logger,
		tracer *apm2.Tracer,
		conns *conns.Conns,
		sentryHub *sentry2.Hub,
	) {
		var err error
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic", zap.Error(fmt.Errorf("recover panic %v", r)))
			}

			conns.Close()

			if errl := logger.Sync(); errl != nil && !errors.Is(errl, syscall.ENOTTY) && !errors.Is(errl, syscall.EINVAL) {
				logger.Error("can not stop logger", zap.Error(errl))
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
			if err != nil {
				os.Exit(1)
			}
		}()

		if err = app.Run(ctx); err != nil {
			logger.Error("failed run app", zap.Error(err))
		}
	}
}
