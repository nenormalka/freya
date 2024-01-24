package mocks

import (
	"context"
	"fmt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

type (
	SQLXMock struct {
		db   *sqlx.DB
		Mock sqlmock.Sqlmock
	}
)

func NewSQLXMock() (*SQLXMock, error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create mock: %w", err)
	}

	return &SQLXMock{db: sqlx.NewDb(db, "sqlmock"), Mock: mock}, nil
}

func (s *SQLXMock) CallContext(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, db *sqlx.DB) error,
) error {
	return callFunc(ctx, s.db)
}

func (s *SQLXMock) CallTransaction(
	ctx context.Context,
	_ string,
	callFunc func(ctx context.Context, tx *sqlx.Tx) error,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if err = callFunc(ctx, tx); err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return rErr
		}

		return err
	}

	return tx.Commit()
}

func (s *SQLXMock) CloseDB() error {
	s.Mock.ExpectClose()
	return s.db.Close()
}
