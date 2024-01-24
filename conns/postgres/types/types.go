package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
)

var (
	ErrAssertionFailed = errors.New("type assertion failed")
)

type (
	PgxQuerier interface {
		pgxtype.Querier
		Get(ctx context.Context, dst any, query string, args ...any) error
		Select(ctx context.Context, dst any, query string, args ...any) error
	}

	PgxTransactor interface {
		pgx.Tx
	}

	PgxConn struct {
		PgxQuerier
	}

	PgxTx struct {
		PgxTransactor
	}

	Scanner interface {
		Scan(src any) error
	}
)

func (p *PgxTx) Select(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Select(ctx, p, dst, query, args...)
}

func (p *PgxTx) Get(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Get(ctx, p, dst, query, args...)
}

func Scan[T Scanner](src any, s T) error {
	switch v := src.(type) {
	case []byte:
		if err := json.Unmarshal(v, s); err != nil {
			return fmt.Errorf("scan []byte %w", err)
		}

		return nil
	case string:
		if err := json.Unmarshal([]byte(v), s); err != nil {
			return fmt.Errorf("scan string %w", err)
		}

		return nil
	case nil:
		return nil
	}

	return ErrAssertionFailed
}
