package sqlx

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/outbox/models"
)

const (
	maxAttemptsDefault = 10

	defaultUpdateLockedTime = 30 * time.Minute
	defaultRemoveOld        = 72 * time.Hour
)

type (
	DataKeeper struct {
		db connectors.DBConnector[*sqlx.DB, *sqlx.Tx]

		maxAttempts int
		lockedTime  time.Duration
		removeOld   time.Duration
	}

	DataKeeperOption func(dk *DataKeeper)
)

func WithLockedTimeDataKeeperOpt(lockedTime time.Duration) DataKeeperOption {
	return func(dk *DataKeeper) {
		dk.lockedTime = lockedTime
	}
}

func WithRemoveOldDataDataKeeperOpt(removeOld time.Duration) DataKeeperOption {
	return func(dk *DataKeeper) {
		dk.removeOld = removeOld
	}
}

func WithMaxAttemptsDataKeeperOpt(maxAttempts int) DataKeeperOption {
	return func(dk *DataKeeper) {
		dk.maxAttempts = maxAttempts
	}
}

func NewDataKeeper(
	db connectors.DBConnector[*sqlx.DB, *sqlx.Tx],
	opts ...DataKeeperOption,
) *DataKeeper {
	dk := &DataKeeper{
		db:          db,
		maxAttempts: maxAttemptsDefault,
		lockedTime:  defaultUpdateLockedTime,
		removeOld:   defaultRemoveOld,
	}

	for _, opt := range opts {
		opt(dk)
	}

	return dk
}

func (dk *DataKeeper) Init(ctx context.Context) error {
	if err := dk.db.CallContext(ctx, "init", func(ctx context.Context, db *sqlx.DB) error {
		_, err := db.ExecContext(ctx, initTableSQL)
		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to init data keeper: %w", err)
	}

	return nil
}

func (dk *DataKeeper) SaveData(ctx context.Context, data *models.Data) error {
	if err := dk.db.CallContext(ctx, "save_data", func(ctx context.Context, db *sqlx.DB) error {
		_, err := db.ExecContext(ctx, insertDataSQL, data.Code, data.Data, data.Source)
		if err != nil {
			return fmt.Errorf("failed to save data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}

	return nil
}

func (dk *DataKeeper) GetData(ctx context.Context) ([]*models.Data, error) {
	data := make([]*models.Data, 0)

	if err := dk.db.CallContext(ctx, "get_data", func(ctx context.Context, db *sqlx.DB) error {
		dataDB := make([]Data, 0)
		if err := db.SelectContext(
			ctx,
			&dataDB,
			selectDataSQL,
			models.DataStatusProcessing,
			dk.maxAttempts,
		); err != nil {
			return fmt.Errorf("failed to select data: %w", err)
		}

		for _, d := range dataDB {
			data = append(data, &models.Data{
				Code:   d.Code,
				Source: models.DataSource(d.Source),
				Data:   d.Data,
			})
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	return data, nil
}

func (dk *DataKeeper) UpdateFailedData(ctx context.Context, codes []string) error {
	if err := dk.db.CallContext(ctx, "update_failed_data", func(ctx context.Context, db *sqlx.DB) error {
		params, args := getParamsAndArgs(codes, 2)
		query := fmt.Sprintf(updateFailedDataSQL, params)
		if _, err := db.ExecContext(ctx, query, append([]any{models.DataStatusNew}, args...)...); err != nil {
			return fmt.Errorf("failed to update data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

func (dk *DataKeeper) UpdateProcessedData(ctx context.Context, codes []string) error {
	if err := dk.db.CallContext(ctx, "update_processed_data", func(ctx context.Context, db *sqlx.DB) error {
		params, args := getParamsAndArgs(codes, 2)
		query := fmt.Sprintf(updateProcessedDataSQL, params)
		if _, err := db.ExecContext(ctx, query, append([]any{models.DataStatusSent}, args...)...); err != nil {
			return fmt.Errorf("failed to update data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

func (dk *DataKeeper) UpdateLockedData(ctx context.Context) error {
	if err := dk.db.CallContext(ctx, "update_locked_data", func(ctx context.Context, db *sqlx.DB) error {
		if _, err := db.ExecContext(
			ctx,
			updateLockedDataSQL,
			models.DataStatusNew,
			time.Now().Add(-dk.lockedTime),
		); err != nil {
			return fmt.Errorf("failed to update data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to update data: %w", err)
	}

	return nil
}

func (dk *DataKeeper) RemoveOldData(ctx context.Context) error {
	if err := dk.db.CallContext(ctx, "remove_old_data", func(ctx context.Context, db *sqlx.DB) error {
		if _, err := db.ExecContext(ctx, removeOldDataSQL, time.Now().Add(-dk.removeOld)); err != nil {
			return fmt.Errorf("failed to remove old data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to remove old data: %w", err)
	}

	return nil
}

func (dk *DataKeeper) SetFailedData(ctx context.Context) error {
	if err := dk.db.CallContext(ctx, "set_failed_data", func(ctx context.Context, db *sqlx.DB) error {
		if _, err := db.ExecContext(ctx, setFailedDataSQL, models.DataStatusFailed, dk.maxAttempts); err != nil {
			return fmt.Errorf("failed to set failed data: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to set failed data: %w", err)
	}

	return nil
}

func getParamsAndArgs(args []string, start int) (string, []any) {
	if len(args) == 0 {
		return "", nil
	}

	q := make([]string, 0, len(args))
	a := make([]any, 0, len(args))

	for indx := range args {
		q = append(q, fmt.Sprintf("$%d", indx+start))
		a = append(a, args[indx])
	}

	return strings.Join(q, ","), a
}
