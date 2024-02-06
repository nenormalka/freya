package watcher

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/consul/api"
	watchplan "github.com/hashicorp/consul/api/watch"
	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

const (
	// github.com/hashicorp/consul/api/watch/funcs.go фабрика с ключами и функциями находится тут
	typeWatcherKey       = "key"
	typeWatcherKeyPrefix = "keyprefix"
)

var (
	ErrWatchFuncNotExists     = errors.New("watchKeyFunc not exists")
	ErrWatchFuncAlreadyExists = errors.New("watchKeyFunc already exists")
)

type (
	Watcher struct {
		cli *api.Client
		log *zap.Logger

		mu *sync.RWMutex

		unwatchFunc map[string]UnwatchKeyFunc
		plans       map[string]*watchplan.Plan

		watchPrefixKeyFunc *watchFuncMap[WatchPrefixKeyFunc]
		watchKeyFunc       *watchFuncMap[WatchKeyFunc]
	}

	WatchKeyFunc       func(key, sessionID string, value []byte)
	WatchPrefixKeyFunc func(params map[string][]byte)
	UnwatchKeyFunc     func() error
	PlanHandlerFunc    func(typeWatch, key string) func(idx uint64, raw any)

	WatchKeys      map[string]WatchKeyFunc
	WatchPrefixKey map[string]WatchPrefixKeyFunc
)

func NewWatcher(cli *api.Client, logger *zap.Logger) *Watcher {
	return &Watcher{
		cli: cli,
		log: logger,
		mu:  &sync.RWMutex{},

		plans:       make(map[string]*watchplan.Plan),
		unwatchFunc: make(map[string]UnwatchKeyFunc),

		watchKeyFunc:       newWatchFuncMap[WatchKeyFunc](),
		watchPrefixKeyFunc: newWatchFuncMap[WatchPrefixKeyFunc](),
	}
}

func (w *Watcher) Start(_ context.Context) error {
	w.startPlans()

	return nil
}

func (w *Watcher) Stop(_ context.Context) error {
	var err error

	for key, f := range w.unwatchFunc {
		if errUn := f(); errUn != nil {
			err = errors.Join(err, fmt.Errorf("key: %s, unwatchFunc: %w", key, errUn))
		}
	}

	return err
}

func (w *Watcher) WatchKeys(keys WatchKeys) error {
	for key, h := range keys {
		if err := addWatchFunc[WatchKeyFunc](
			w,
			w.watchKeyFunc,
			typeWatcherKey,
			key,
			h,
		); err != nil {
			return fmt.Errorf("addWatchFunc: %w", err)
		}

		if err := addPlan[WatchKeyFunc](
			w,
			w.watchKeyFunc,
			map[string]any{
				"type": typeWatcherKey,
				"key":  key,
			},
			w.getKeyPlanHandler,
			typeWatcherKey,
			key,
		); err != nil {
			return fmt.Errorf("addPlan: %w", err)
		}
	}

	return nil
}

func (w *Watcher) WatchPrefixKeys(keys WatchPrefixKey) error {
	for key, h := range keys {
		if err := addWatchFunc[WatchPrefixKeyFunc](
			w,
			w.watchPrefixKeyFunc,
			typeWatcherKeyPrefix,
			key,
			h,
		); err != nil {
			return fmt.Errorf("addWatchFunc: %w", err)
		}

		if err := addPlan[WatchPrefixKeyFunc](
			w,
			w.watchPrefixKeyFunc,
			map[string]any{
				"type":   typeWatcherKeyPrefix,
				"prefix": key,
			},
			w.getKeyPrefixPlanHandler,
			typeWatcherKeyPrefix,
			key,
		); err != nil {
			return fmt.Errorf("addPlan: %w", err)
		}
	}

	return nil
}

func (w *Watcher) getKeyPlanHandler(typeWatch, key string) func(uint64, any) {
	return func(_ uint64, raw any) {
		if raw == nil {
			return
		}

		pair, ok := raw.(*api.KVPair)
		if !ok {
			w.log.Warn(
				"failed type assertion",
				zap.Any("raw", raw),
				zap.String("type", typeWatch),
				zap.String("key", key),
			)

			return
		}

		doWatchFunc[WatchKeyFunc](
			w,
			w.watchKeyFunc,
			typeWatch,
			key,
			func(handler WatchKeyFunc) {
				handler(pair.Key, pair.Session, pair.Value)
			},
		)
	}
}

func (w *Watcher) getKeyPrefixPlanHandler(typeWatch, key string) func(uint64, any) {
	return func(_ uint64, raw any) {
		if raw == nil {
			return
		}

		pairs, ok := raw.(api.KVPairs)
		if !ok {
			w.log.Warn(
				"failed type assertion",
				zap.Any("raw", raw),
				zap.String("type", typeWatch),
				zap.String("key", key),
			)

			return
		}

		params := make(map[string][]byte, len(pairs))
		for _, pair := range pairs {
			if pair.Key == "" {
				continue
			}

			params[pair.Key] = pair.Value
		}

		if len(params) == 0 {
			return
		}

		doWatchFunc[WatchPrefixKeyFunc](
			w,
			w.watchPrefixKeyFunc,
			typeWatch,
			key,
			func(handler WatchPrefixKeyFunc) {
				handler(params)
			},
		)
	}
}

func (w *Watcher) startPlans() {
	w.mu.Lock()
	defer w.mu.Unlock()

	hcl := hclog.New(&hclog.LoggerOptions{
		Name:   "consul_watcher",
		Level:  hclog.Debug,
		Output: &zapio.Writer{Log: w.log.With(zap.String("consul", "watcher")), Level: zap.DebugLevel},
	})

	for key, plan := range w.plans {
		go func(key string, plan *watchplan.Plan) {
			if err := plan.RunWithClientAndHclog(w.cli, hcl); err != nil {
				w.log.Error(
					"failed start watching plan",
					zap.String("type", plan.Type),
					zap.Error(err),
					zap.String("key", key),
				)
			}
		}(key, plan)
	}
}

func (w *Watcher) stopPlan(typeWatch, key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	plan, ok := w.plans[typeWatch+key]
	if !ok {
		return
	}

	plan.Stop()
	delete(w.plans, typeWatch+key)
}

func addWatchFunc[M watchFunc](
	w *Watcher,
	wfm *watchFuncMap[M],
	typeWatch, key string,
	h M,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := wfm.get(typeWatch + key); ok {
		return ErrWatchFuncAlreadyExists
	}

	wfm.set(typeWatch+key, h)
	w.unwatchFunc[typeWatch+key] = getUnwatchFunc[M](w, wfm, typeWatch, key)

	return nil
}

func getUnwatchFunc[M watchFunc](
	w *Watcher,
	wfm *watchFuncMap[M],
	typeWatch, key string,
) UnwatchKeyFunc {
	return func() error {
		if err := removeWatchFunc[M](
			w,
			wfm,
			typeWatch,
			key,
		); err != nil {
			return fmt.Errorf("removeWatchFunc: %w", err)
		}

		w.stopPlan(typeWatch, key)

		w.log.Info("unwatch key", zap.String("key", key), zap.String("type", typeWatch))

		return nil
	}
}

func removeWatchFunc[M watchFunc](
	w *Watcher,
	wfm *watchFuncMap[M],
	typeWatch, key string,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := wfm.get(typeWatch + key); !ok {
		return ErrWatchFuncNotExists
	}

	wfm.delete(typeWatch + key)

	return nil
}

func addPlan[M watchFunc](
	w *Watcher,
	wfm *watchFuncMap[M],
	params map[string]any,
	f PlanHandlerFunc,
	typeWatch, key string,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	plan, err := watchplan.Parse(params)
	if err != nil {
		if errU := removeWatchFunc[M](
			w,
			wfm,
			typeWatch,
			key,
		); errU != nil {
			w.log.Error("failed removeWatchFunc", zap.Error(errU))
		}

		return fmt.Errorf("watchplan parse: %w", err)
	}

	plan.Handler = f(typeWatch, key)

	w.plans[typeWatch+key] = plan

	return nil
}

func doWatchFunc[M watchFunc](
	w *Watcher,
	wfm *watchFuncMap[M],
	typeWatch, key string,
	caller func(h M),
) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	handler, ok := wfm.get(typeWatch + key)
	if !ok || handler == nil {
		w.log.Warn("watchFunc not exists", zap.String("type", typeWatch), zap.String("key", key))

		return
	}

	caller(handler)
}
