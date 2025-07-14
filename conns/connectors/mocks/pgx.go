package mocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"

	dbtypes "github.com/nenormalka/freya/conns/postgres/types"
)

type (
	PGXMock struct {
		Mock *mock
	}

	mock struct {
		pgxmock.PgxPoolIface
	}
)

func NewPGXMock() (*PGXMock, error) {
	m, err := pgxmock.NewPool()
	if err != nil {
		return nil, fmt.Errorf("failed to create mock: %w", err)
	}

	return &PGXMock{Mock: &mock{PgxPoolIface: m}}, nil
}

func (p *PGXMock) CallContext(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, db dbtypes.PgxConn) error,
) error {
	return callFunc(ctx, dbtypes.PgxConn{PgxQuerier: p.Mock})
}

func (p *PGXMock) CallTransaction(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, tx dbtypes.PgxTx) error,
) error {
	tx, err := p.Mock.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	if err = callFunc(ctx, dbtypes.PgxTx{PgxTransactor: tx}); err != nil {
		if rErr := tx.Rollback(ctx); rErr != nil {
			return errors.Join(err, rErr)
		}

		return err
	}

	return tx.Commit(ctx)
}

func (p *PGXMock) CloseDB() error {
	p.Mock.ExpectClose()
	p.Mock.Close()

	return nil
}

func (m *mock) Select(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Select(ctx, m, dst, query, args...)
}

func (m *mock) Get(ctx context.Context, dst any, query string, args ...any) error {
	return pgxscan.Get(ctx, m, dst, query, args...)
}
