package freya

import (
	"fmt"
	"testing"

	lilith "github.com/nenormalka/lilith/methods"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"

	"github.com/nenormalka/freya/types"
)

type (
	MockEngine struct {
		modules types.Module
	}

	MockEngineOpt func(engine *MockEngine)
)

func WithModulesOpt(modules types.Module) MockEngineOpt {
	return func(engine *MockEngine) {
		engine.modules = append(engine.modules, modules...)
	}
}

func NewMockEngine(withDefaultModules bool, opts ...MockEngineOpt) *MockEngine {
	engine := &MockEngine{
		modules: lilith.Ternary(withDefaultModules, append(defaultModules, types.Module{
			{
				CreateFunc: func() (*types.AppInfo, error) {
					return types.GetAppInfo(nil, "mock_engine", "")
				},
			},
		}...), nil),
	}

	for _, opt := range opts {
		opt(engine)
	}

	return engine
}

func (e *MockEngine) Run(f any) error {
	return e.run(f)
}

func (e *MockEngine) RunTest(tt *testing.T, name string, f any) {
	tt.Run(name, func(tt *testing.T) {
		require.NoError(tt, e.run(f,
			types.Provider{
				CreateFunc: func() *testing.T { return tt },
			},
		))
	})
}

func (e *MockEngine) RunBenchmark(tb *testing.B, name string, f any) {
	tb.Run(name, func(tb *testing.B) {
		require.NoError(tb, e.run(f,
			types.Provider{
				CreateFunc: func() *testing.B { return tb },
			},
		))
	})
}

func (e *MockEngine) run(f any, modules ...types.Provider) error {
	var (
		d   = dig.New()
		err error
	)

	modules = append(modules, e.modules...)

	for _, m := range modules {
		if err = d.Provide(m.CreateFunc, m.Options...); err != nil {
			return fmt.Errorf("run provide: %w", err)
		}
	}

	if err = d.Invoke(f); err != nil {
		return fmt.Errorf("run invoke: %w", err)
	}

	return nil
}
