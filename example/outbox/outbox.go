package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/consul"
	"github.com/nenormalka/freya/outbox"
	"github.com/nenormalka/freya/outbox/models"

	"go.uber.org/zap"
)

const (
	source = "test"
)

type (
	Outbox struct {
		logger *zap.Logger
		outbox *outbox.Outbox
		leader consul.Leader
	}
)

func NewOutbox(logger *zap.Logger, c *conns.Conns) (*Outbox, error) {
	db, err := c.GetSQLConnByName(connectors.DefaultDBConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get sql connection: %w", err)
	}

	csl, err := c.GetConsul()
	if err != nil {
		return nil, fmt.Errorf("failed to get consul connection: %w", err)
	}

	leader := csl.Leader()

	o, err := outbox.NewDefaultOutbox(logger, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox: %w", err)
	}

	ob := &Outbox{
		logger: logger,
		outbox: o,
		leader: leader,
	}

	o.AddHandler(source, ob.testHandler)
	o.AddIsAvailable(leader.IsLeader)

	return ob, nil
}

func (o *Outbox) Start(ctx context.Context) error {
	if err := o.leader.Start(ctx); err != nil {
		return fmt.Errorf("failed to start leader: %w", err)
	}

	if err := o.outbox.Start(ctx); err != nil {
		return fmt.Errorf("failed to start outbox: %w", err)
	}

	go o.spam(ctx)

	return nil
}

func (o *Outbox) Stop(ctx context.Context) error {
	if err := o.leader.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop leader: %w", err)
	}

	if err := o.outbox.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop outbox: %w", err)
	}

	return nil
}

func (o *Outbox) spam(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	i := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := struct {
				Count int
			}{
				Count: i,
			}

			b, err := json.Marshal(data)
			if err != nil {
				o.logger.Error("failed to marshal data", zap.Error(err))
				continue
			}

			if err = o.outbox.SaveData(ctx, source, b); err != nil {
				o.logger.Error("failed to save data", zap.Error(err))
			}
			i++
		}
	}
}

func (o *Outbox) testHandler(_ context.Context, data []*models.Data) ([]string, error) {
	failed := make([]string, 0)
	for i, d := range data {
		if i%5 == 0 {
			failed = append(failed, d.Code)
		}

		o.logger.Info(
			"------------ test handler outbox ------------",
			zap.String("code", d.Code),
			zap.ByteString("data", d.Data),
		)
	}

	return failed, nil
}
