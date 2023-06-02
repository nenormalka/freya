package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	healthCheckStatusPass healthCheckStatus = "pass"
	healthCheckStatusFail healthCheckStatus = "fail"
)

type (
	healthCheckStatus string

	response struct {
		Status    healthCheckStatus          `json:"status"`
		ReleaseID string                     `json:"releaseId,omitempty"`
		Errors    map[string]string          `json:"errors,omitempty"`
		Checks    map[string][]checkResponse `json:"checks"`
		Output    string                     `json:"output,omitempty"`
	}

	checkResponse struct {
		Status healthCheckStatus `json:"status"`
		Output string            `json:"output,omitempty"`
		Time   string            `json:"time"`
	}

	health struct {
		checkers  map[string]Checker
		observers map[string]Checker
		timeout   time.Duration
		releaseID string
	}

	// Checker checks the status of the dependency and returns error.
	// In case the dependency is working as expected, return nil.
	Checker interface {
		Check(ctx context.Context) error
	}

	CheckerFunc func(ctx context.Context) error

	// Option adds optional parameter for the HealthcheckHandlerFunc
	Option func(*health)

	timeoutChecker struct {
		checker Checker
	}
)

// Check Implements the Checker interface to allow for any func() error method
// to be passed as a Checker
func (c CheckerFunc) Check(ctx context.Context) error {
	return c(ctx)
}

// Handler returns an http.Handler
func Handler(opts ...Option) http.Handler {
	h := &health{
		checkers:  make(map[string]Checker),
		observers: make(map[string]Checker),
		timeout:   30 * time.Second,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HandlerFunc returns an http.HandlerFunc to mount the API implementation at a specific route
func HandlerFunc(opts ...Option) http.HandlerFunc {
	return Handler(opts...).ServeHTTP
}

// WithChecker adds a status checker that needs to be added as part of healthcheck. i.e database, cache or any external dependency
func WithChecker(name string, s Checker) Option {
	return func(h *health) {
		h.checkers[name] = &timeoutChecker{s}
	}
}

// WithObserver adds a status checker but it does not fail the entire status.
func WithObserver(name string, s Checker) Option {
	return func(h *health) {
		h.observers[name] = &timeoutChecker{s}
	}
}

// WithTimeout configures the global timeout for all individual checkers.
func WithTimeout(timeout time.Duration) Option {
	return func(h *health) {
		h.timeout = timeout
	}
}

func WithReleaseID(releaseID string) Option {
	return func(h *health) {
		h.releaseID = releaseID
	}
}

func ReadReleaseIDFromPath(path string) (string, error) {
	payload, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	ss := []string{"\n", "\\n"}
	res := string(payload)
	for _, s := range ss {
		res = strings.ReplaceAll(res, s, "")
	}
	return res, nil
}

func (h *health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nCheckers := len(h.checkers) + len(h.observers)
	checks := make(map[string][]checkResponse)

	status := healthCheckStatusPass
	output := ""

	ctx, cancel := context.Background(), func() {}
	if h.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
	}
	defer cancel()

	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(nCheckers)

	for key, checker := range h.checkers {
		go func(key string, checker Checker) {
			err := checker.Check(ctx)
			mutex.Lock()
			checks[key] = []checkResponse{checkerResponseFromError(err)}
			if err != nil {
				status = healthCheckStatusFail
				output = fmt.Sprintf("%s:%v", key, err)
			}
			mutex.Unlock()
			wg.Done()
		}(key, checker)
	}
	for key, observer := range h.observers {
		go func(key string, observer Checker) {
			err := observer.Check(ctx)
			mutex.Lock()
			checks[key] = []checkResponse{checkerResponseFromError(err)}
			if err != nil {
				status = healthCheckStatusFail
				output = fmt.Sprintf("%s: %v", key, err)
			}
			mutex.Unlock()
			wg.Done()
		}(key, observer)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatusCodeFromHealthCheckStatus(status))
	json.NewEncoder(w).Encode(response{
		ReleaseID: h.releaseID,
		Status:    status,
		Checks:    checks,
		Output:    output,
	})
}

func (t *timeoutChecker) Check(ctx context.Context) error {
	checkerChan := make(chan error)
	go func() {
		checkerChan <- t.checker.Check(ctx)
	}()
	select {
	case err := <-checkerChan:
		return err
	case <-ctx.Done():
		return errors.New("max check time exceeded")
	}
}

func httpStatusCodeFromHealthCheckStatus(s healthCheckStatus) int {
	if s == healthCheckStatusPass {
		return http.StatusOK
	}
	return http.StatusServiceUnavailable
}

func checkerResponseFromError(err error) checkResponse {
	res := checkResponse{
		Status: healthCheckStatusPass,
		Time:   time.Now().UTC().String(),
	}
	if err != nil {
		res.Status = healthCheckStatusFail
		res.Output = err.Error()
	}
	return res
}
