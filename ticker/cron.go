package ticker

import (
	"context"
	"fmt"

	"github.com/nenormalka/freya/outbox/models"

	"github.com/robfig/cron/v3"
)

type (
	CronTicker struct {
		cron *cron.Cron
		jobs []*JobCronTicker
	}

	JobCronTicker struct {
		time string
		j    models.Job
	}

	CronTickerOption func(ct *CronTicker)
)

func WithJobsCronTickerOpt(jobs ...*JobCronTicker) CronTickerOption {
	return func(t *CronTicker) {
		t.jobs = jobs
	}
}

func NewCronTicker(opts ...CronTickerOption) *CronTicker {
	ct := &CronTicker{
		cron: cron.New(
			cron.WithChain(
				cron.Recover(cron.DefaultLogger),
				cron.SkipIfStillRunning(cron.DefaultLogger),
			),
		),
	}

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}

func (ct *CronTicker) AddJob(t string, job models.Job) error {
	ct.jobs = append(ct.jobs, &JobCronTicker{
		time: t,
		j:    job,
	})

	return nil
}

func (ct *CronTicker) Start(ctx context.Context) error {
	if len(ct.jobs) == 0 {
		return ErrEmptyJobs
	}

	for _, job := range ct.jobs {
		if _, err := ct.cron.AddFunc(job.time, func() {
			job.j(ctx)
		}); err != nil {
			return fmt.Errorf("error adding job: %w", err)
		}
	}

	ct.cron.Start()

	return nil
}

func (ct *CronTicker) Stop(_ context.Context) error {
	ct.cron.Stop()

	return nil
}
