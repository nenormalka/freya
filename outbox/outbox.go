package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nenormalka/freya/conns/connectors"
	dk "github.com/nenormalka/freya/outbox/datakeeper/sqlx"
	"github.com/nenormalka/freya/outbox/models"
	"github.com/nenormalka/freya/ticker"
	"github.com/nenormalka/freya/types"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	lilith "github.com/nenormalka/lilith/methods"

	"go.uber.org/zap"
)

var (
	ErrEmptyTicker        = errors.New("empty ticker")
	ErrEmptyDataKeeper    = errors.New("empty data keeper")
	ErrEmptyHandlers      = errors.New("empty handlers")
	ErrDoJobPeriod        = errors.New("do job period is empty")
	ErrUpdateLockedPeriod = errors.New("update locked period is empty")
	ErrRemoveOldPeriod    = errors.New("remove old period is empty")
	ErrSetFailedPeriod    = errors.New("set failed period is empty")
	ErrEmptyIsAvailable   = errors.New("is available is empty")
	ErrEmptyOutbox        = errors.New("empty outbox")
)

const (
	defaultDoJobTime        = 10 * time.Minute
	defaultUpdateLockedTime = 1 * time.Hour
	defaultRemoveOldTime    = 24 * time.Hour
	defaultsSetFailedTime   = 1 * time.Hour
)

type (
	Handler             func(ctx context.Context, data []*models.Data) ([]string, error)
	TypedHandler[T any] func(ctx context.Context, data []T) ([]string, error)
	IsAvailable         func() bool

	Ticker interface {
		Start(ctx context.Context) error
		Stop(ctx context.Context) error
		AddJob(time string, job models.Job) error
	}

	DataKeeper interface {
		Init(ctx context.Context) error
		SaveData(ctx context.Context, data *models.Data) error
		GetData(ctx context.Context) ([]*models.Data, error)
		UpdateFailedData(ctx context.Context, codes []string) error
		UpdateProcessedData(ctx context.Context, codes []string) error
		UpdateLockedData(ctx context.Context) error
		RemoveOldData(ctx context.Context) error
		SetFailedData(ctx context.Context) error
	}

	Outbox struct {
		ticker Ticker
		dk     DataKeeper

		logger *zap.Logger

		handlers    map[models.DataSource]Handler
		isAvailable IsAvailable

		tickerConfig
	}

	tickerConfig struct {
		doJobPeriod        string
		updateLockedPeriod string
		removeOldPeriod    string
		setFailedPeriod    string
	}

	OutboxOption func(o *Outbox)
)

func WithAvailableOutboxOpt(a IsAvailable) OutboxOption {
	return func(o *Outbox) {
		o.isAvailable = a
	}
}

func WithSetFailedPeriodOutboxOpt(p string) OutboxOption {
	return func(o *Outbox) {
		o.setFailedPeriod = p
	}
}

func WithRemoveOldPeriodOutboxOpt(p string) OutboxOption {
	return func(o *Outbox) {
		o.removeOldPeriod = p
	}
}

func WithDoJobPeriodOutboxOpt(p string) OutboxOption {
	return func(o *Outbox) {
		o.doJobPeriod = p
	}
}

func WithUpdateLockedPeriodOutboxOpt(p string) OutboxOption {
	return func(o *Outbox) {
		o.updateLockedPeriod = p
	}
}

func WithTickerOutboxOpt(t Ticker) OutboxOption {
	return func(o *Outbox) {
		o.ticker = t
	}
}

func WithDataKeeperOutboxOpt(dk DataKeeper) OutboxOption {
	return func(o *Outbox) {
		o.dk = dk
	}
}

func WithHandlersOutboxOpt(handlers map[models.DataSource]Handler) OutboxOption {
	return func(o *Outbox) {
		o.handlers = handlers
	}
}

func AddTypedHandler[T any](
	o *Outbox,
	source models.DataSource,
	h TypedHandler[T],
) error {
	if o == nil {
		return ErrEmptyOutbox
	}

	o.AddHandler(source, func(ctx context.Context, data []*models.Data) ([]string, error) {
		dataTyped := make([]T, 0, len(data))

		for _, d := range data {
			var dTyped T
			if err := json.Unmarshal(d.Data, &dTyped); err != nil {
				return nil, fmt.Errorf("failed to unmarshal data: %w", err)
			}

			dataTyped = append(dataTyped, dTyped)
		}

		codes, err := h(ctx, dataTyped)
		if err != nil {
			return nil, fmt.Errorf("failed to handle data: %w", err)
		}

		return codes, nil
	})

	return nil
}

func TypedSaveData[T any](
	ctx context.Context,
	o *Outbox,
	source models.DataSource,
	data T,
) error {
	if o == nil {
		return ErrEmptyOutbox
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return o.SaveData(ctx, source, dataBytes)
}

func NewDefaultOutbox(
	logger *zap.Logger,
	db connectors.DBConnector[*sqlx.DB, *sqlx.Tx],
	opts ...OutboxOption,
) (*Outbox, error) {
	o, err := NewOutbox(logger, append(
		[]OutboxOption{
			WithDoJobPeriodOutboxOpt(defaultDoJobTime.String()),
			WithUpdateLockedPeriodOutboxOpt(defaultUpdateLockedTime.String()),
			WithRemoveOldPeriodOutboxOpt(defaultRemoveOldTime.String()),
			WithSetFailedPeriodOutboxOpt(defaultsSetFailedTime.String()),
			WithDataKeeperOutboxOpt(dk.NewDataKeeper(db)),
			WithAvailableOutboxOpt(func() bool { return true }),
			WithTickerOutboxOpt(ticker.NewTimeTicker()),
		}, opts...,
	)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox: %w", err)
	}

	return o, nil
}

func NewOutbox(logger *zap.Logger, opts ...OutboxOption) (*Outbox, error) {
	o := &Outbox{
		logger:   logger,
		handlers: make(map[models.DataSource]Handler),
	}

	for _, opt := range opts {
		opt(o)
	}

	return o, nil
}

func (o *Outbox) SaveData(ctx context.Context, source models.DataSource, data []byte) error {
	if err := o.dk.SaveData(ctx, &models.Data{
		Code:   uuid.New().String(),
		Source: source,
		Data:   data,
	}); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	types.OutboxMetricsF(string(source), models.DataStatusNew, 1)

	return nil
}

func (o *Outbox) Start(ctx context.Context) error {
	if err := o.validate(); err != nil {
		return fmt.Errorf("failed to validate: %w", err)
	}

	if err := o.dk.Init(ctx); err != nil {
		return fmt.Errorf("failed to init data keeper: %w", err)
	}

	if err := o.addJobs(); err != nil {
		return fmt.Errorf("failed to add jobs: %w", err)
	}

	if err := o.ticker.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ticker: %w", err)
	}

	return nil
}

func (o *Outbox) Stop(ctx context.Context) error {
	if err := o.ticker.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop ticker: %w", err)
	}

	return nil
}

func (o *Outbox) AddHandler(source models.DataSource, handler Handler) {
	o.handlers[source] = handler
}

func (o *Outbox) AddHandlers(handlers map[models.DataSource]Handler) {
	o.handlers = handlers
}

func (o *Outbox) AddTicker(t Ticker) {
	o.ticker = t
}

func (o *Outbox) AddDataKeeper(dk DataKeeper) {
	o.dk = dk
}

func (o *Outbox) AddDoJobPeriod(p string) {
	o.doJobPeriod = p
}

func (o *Outbox) AddUpdateLockedPeriod(p string) {
	o.updateLockedPeriod = p
}

func (o *Outbox) AddRemoveOldPeriod(p string) {
	o.removeOldPeriod = p
}

func (o *Outbox) AddSetFailedPeriod(p string) {
	o.setFailedPeriod = p
}

func (o *Outbox) AddIsAvailable(a IsAvailable) {
	o.isAvailable = a
}

func (o *Outbox) addJobs() error {
	if err := o.ticker.AddJob(o.doJobPeriod, func(ctx context.Context) {
		if !o.isAvailable() {
			return
		}
		o.processingMessages(ctx)
	}); err != nil {
		return fmt.Errorf("failed to add do job: %w", err)
	}

	if err := o.ticker.AddJob(o.updateLockedPeriod, func(ctx context.Context) {
		if !o.isAvailable() {
			return
		}

		if err := o.dk.UpdateLockedData(ctx); err != nil {
			o.logger.Error("failed to update locked data", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("failed to add update locked data: %w", err)
	}

	if err := o.ticker.AddJob(o.removeOldPeriod, func(ctx context.Context) {
		if !o.isAvailable() {
			return
		}

		if err := o.dk.RemoveOldData(ctx); err != nil {
			o.logger.Error("failed to remove old data", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("failed to add remove old data: %w", err)
	}

	if err := o.ticker.AddJob(o.setFailedPeriod, func(ctx context.Context) {
		if !o.isAvailable() {
			return
		}

		if err := o.dk.SetFailedData(ctx); err != nil {
			o.logger.Error("failed to set failed data", zap.Error(err))
		}
	}); err != nil {
		return fmt.Errorf("failed to add set failed data: %w", err)
	}

	return nil
}

func (o *Outbox) processingMessages(ctx context.Context) {
	data, err := o.dk.GetData(ctx)
	if err != nil {
		o.logger.Error("failed to get data", zap.Error(err))
	}

	if len(data) == 0 {
		return
	}

	dataBySource := make(map[models.DataSource][]*models.Data)
	dataCodesAll := make([]string, 0, len(data))
	dataCodesFailed := make([]string, 0)

	for _, d := range data {
		dataBySource[d.Source] = append(dataBySource[d.Source], d)
		dataCodesAll = append(dataCodesAll, d.Code)
	}

	for source, handler := range o.handlers {
		msgs, ok := dataBySource[source]
		if !ok || len(msgs) == 0 {
			continue
		}

		codes, errH := handler(ctx, msgs)
		if errH != nil {
			o.logger.Error("failed to handle msgs", zap.Error(errH))
		}

		dataCodesFailed = append(dataCodesFailed, codes...)

		if len(codes) > 0 {
			types.OutboxMetricsF(string(source), models.DataStatusFailed, float64(len(codes)))
		}

		if len(codes) != len(msgs) {
			types.OutboxMetricsF(string(source), models.DataStatusSent, float64(len(msgs)-len(codes)))
		}
	}

	if len(dataCodesFailed) > 0 {
		if errD := o.dk.UpdateFailedData(ctx, dataCodesFailed); errD != nil {
			o.logger.Error("failed to update data", zap.Error(errD))
		}
	}

	if len(dataCodesAll) == 0 {
		return
	}

	diff := lilith.ArrayDiff(dataCodesAll, dataCodesFailed)
	if len(diff) == 0 {
		return
	}

	if errD := o.dk.UpdateProcessedData(ctx, diff); errD != nil {
		o.logger.Error("failed to update data", zap.Error(errD))
	}
}

func (o *Outbox) validate() error {
	if o.ticker == nil {
		return ErrEmptyTicker
	}

	if o.dk == nil {
		return ErrEmptyDataKeeper
	}

	if o.isAvailable == nil {
		return ErrEmptyIsAvailable
	}

	if len(o.handlers) == 0 {
		return ErrEmptyHandlers
	}

	for _, str := range []struct {
		field string
		err   error
	}{
		{field: o.doJobPeriod, err: ErrDoJobPeriod},
		{field: o.updateLockedPeriod, err: ErrUpdateLockedPeriod},
		{field: o.removeOldPeriod, err: ErrRemoveOldPeriod},
		{field: o.setFailedPeriod, err: ErrSetFailedPeriod},
	} {
		if str.field != "" {
			continue
		}

		return str.err
	}

	return nil
}
