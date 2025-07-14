package postgrelock

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/nenormalka/freya/conns/connectors"
)

type (
	SQLXLock struct {
		c CommonLock[*sqlx.DB, *sqlx.Tx]
	}
)

func NewSQLXLock(
	db connectors.DBConnector[*sqlx.DB, *sqlx.Tx],
	opts ...LockOption,
) (*SQLXLock, error) {
	return &SQLXLock{
		c: NewCommonLock(db, opts...),
	}, nil
}

func (sl *SQLXLock) DoUnderLock(ctx context.Context, key string, f func(ctx context.Context) error) error {
	if err := sl.c.DoUnderLock(ctx, key, func(ctx context.Context, lockQuery, unlockQuery string, db *sqlx.DB) error {
		var (
			allowed = false
			err     error
		)

		if err = db.Get(&allowed, lockQuery); err != nil {
			return fmt.Errorf("sqlx get under lock err %w", err)
		}

		if !allowed {
			return ErrNotAllow
		}

		defer func() {
			if _, errC := db.Exec(unlockQuery); errC != nil {
				err = errors.Join(err, fmt.Errorf("sqlx exec unlock under lock err %w", errC))
			}
		}()

		if err = f(ctx); err != nil {
			return fmt.Errorf("f under lock err %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("sqlx do under lock err %w", err)
	}

	return nil
}

func (sl *SQLXLock) DoUnderLockTx(ctx context.Context, key string, f func(ctx context.Context) error) error {
	if err := sl.c.DoUnderLockTx(ctx, key, func(ctx context.Context, lockQuery string, db *sqlx.Tx) error {
		var (
			allowed = false
			err     error
		)

		if err = db.Get(&allowed, lockQuery); err != nil {
			return fmt.Errorf("sqlx get under lock err %w", err)
		}

		if !allowed {
			return ErrNotAllow
		}

		if err = f(ctx); err != nil {
			return fmt.Errorf("f under lock err %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("sqlx do under lock err tx %w", err)
	}

	return nil
}
