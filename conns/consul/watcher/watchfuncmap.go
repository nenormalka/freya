package watcher

import "sync"

type (
	watchFunc interface {
		WatchKeyFunc | WatchPrefixKeyFunc
	}

	watchFuncMap[M watchFunc] struct {
		mu *sync.RWMutex
		m  map[string]M
	}
)

func newWatchFuncMap[M watchFunc]() *watchFuncMap[M] {
	return &watchFuncMap[M]{
		mu: &sync.RWMutex{},
		m:  make(map[string]M),
	}
}

func (wfm *watchFuncMap[M]) get(key string) (M, bool) {
	wfm.mu.RLock()
	defer wfm.mu.RUnlock()

	if f, ok := wfm.m[key]; ok {
		return f, true
	}

	return nil, false
}

func (wfm *watchFuncMap[M]) set(key string, f M) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	wfm.m[key] = f
}

func (wfm *watchFuncMap[M]) delete(key string) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	delete(wfm.m, key)
}
