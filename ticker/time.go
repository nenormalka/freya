package ticker

import (
	"context"
	"fmt"
	"time"

	"github.com/nenormalka/freya/outbox/models"

	lilith "github.com/nenormalka/lilith/patterns"
)

type (
	TimeTicker struct {
		jobs []*JobTimeTicker
	}

	TimeTickerOption func(tt *TimeTicker)

	JobTimeTicker struct {
		time time.Duration
		j    models.Job
	}
)

func WithJobsTimeTickerOpt(jobs ...*JobTimeTicker) TimeTickerOption {
	return func(t *TimeTicker) {
		t.jobs = jobs
	}
}

func NewTimeTicker(opts ...TimeTickerOption) *TimeTicker {
	tt := &TimeTicker{}

	for _, opt := range opts {
		opt(tt)
	}

	return tt
}

func (tt *TimeTicker) AddJob(t string, job models.Job) error {
	d, err := time.ParseDuration(t)
	if err != nil {
		return fmt.Errorf("time.ParseDuration: %w", err)
	}

	tt.jobs = append(tt.jobs, &JobTimeTicker{
		time: d,
		j:    job,
	})

	return nil
}

func (tt *TimeTicker) Start(ctx context.Context) error {
	if len(tt.jobs) == 0 {
		return ErrEmptyJobs
	}

	for _, j := range tt.jobs {
		lilith.Ticker(ctx, j.time, func() {
			j.j(ctx)
		})
	}

	return nil
}

func (tt *TimeTicker) Stop(_ context.Context) error {
	return nil
}
