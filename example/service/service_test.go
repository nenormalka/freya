package service_test

import (
	"context"
	"fmt"
	"testing"

	"freya/example/repo"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/nenormalka/freya"
	"github.com/nenormalka/freya/types"
)

type (
	mockedTypeInt    int
	mockedTypeStruct struct {
		mt mockedTypeInt
	}
)

func TestMockRunWithDefaultModules(t *testing.T) {
	require.NoError(t, freya.
		NewMockEngine(true, freya.WithModulesOpt(types.Module{
			{
				CreateFunc: repo.NewRepo,
			},
		})).
		Run(func(l *zap.Logger, repo *repo.Repo) error {
			date, err := repo.GetNow(context.Background())
			if err != nil {
				return fmt.Errorf("get now %w", err)
			}

			l.Info(date)

			return nil
		}),
	)
}

func TestMockRunWithoutDefaultModules(t *testing.T) {
	require.NoError(t, freya.
		NewMockEngine(false, freya.WithModulesOpt(types.Module{
			{
				CreateFunc: func() *zap.Logger {
					return zap.NewNop()
				},
			},
			{
				CreateFunc: func() mockedTypeInt {
					return mockedTypeInt(100_000)
				},
			},
			{
				CreateFunc: func(mti mockedTypeInt) mockedTypeStruct {
					return mockedTypeStruct{mt: mti}
				},
			},
		})).
		Run(func(l *zap.Logger, mts mockedTypeStruct) {
			l.Info("mockedTypeStruct", zap.Any("mockedTypeStruct", mts))
		}),
	)
}

func TestMockRunTest(t *testing.T) {
	engine := freya.NewMockEngine(true, freya.WithModulesOpt(types.Module{
		{
			CreateFunc: repo.NewRepo,
		},
	}))

	engine.RunTest(t, "test", func(l *zap.Logger, repo *repo.Repo) error {
		date, err := repo.GetNow(context.Background())
		l.Info("date", zap.String("date", date))
		require.NoError(t, err)
		require.NotEqual(t, "", date)

		return nil
	})
}
