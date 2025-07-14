package postgrelock

import (
	"context"
	"errors"
	"fmt"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/conns/postgres/types"
)

type (
	PGXLock struct {
		c CommonLock[types.PgxConn, types.PgxTx]
	}
)

func NewPGXLock(
	db connectors.DBConnector[types.PgxConn, types.PgxTx],
	opts ...LockOption,
) (*PGXLock, error) {
	return &PGXLock{
		c: NewCommonLock(db, opts...),
	}, nil
}

func (pl *PGXLock) DoUnderLock(ctx context.Context, key string, f func(ctx context.Context) error) error {
	if err := pl.c.DoUnderLock(ctx, key, func(ctx context.Context, lockQuery, unlockQuery string, db types.PgxConn) error {
		var (
			allowed = false
			err     error
		)

		if err = db.Get(ctx, &allowed, lockQuery); err != nil {
			return fmt.Errorf("pgx get under lock err %w", err)
		}

		if !allowed {
			return ErrNotAllow
		}

		defer func() {
			if _, errC := db.Exec(ctx, unlockQuery); errC != nil {
				err = errors.Join(err, fmt.Errorf("pgx exec unlock under lock err %w", errC))
			}
		}()

		if err = f(ctx); err != nil {
			return fmt.Errorf("f under lock err %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("pgx do under lock err %w", err)
	}

	return nil
}

func (pl *PGXLock) DoUnderLockTx(ctx context.Context, key string, f func(ctx context.Context) error) error {
	if err := pl.c.DoUnderLockTx(ctx, key, func(ctx context.Context, lockQuery string, db types.PgxTx) error {
		var (
			allowed = false
			err     error
		)

		if err = db.Get(ctx, &allowed, lockQuery); err != nil {
			return fmt.Errorf("pgx get under lock err %w", err)
		}

		if !allowed {
			return ErrNotAllow
		}

		if err = f(ctx); err != nil {
			return fmt.Errorf("f under lock err %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("pgx do under lock tx err %w", err)
	}

	return nil
}
