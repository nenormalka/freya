package repo

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/nenormalka/freya/conns"
	"github.com/nenormalka/freya/conns/connectors"
	"go.uber.org/zap"
)

type (
	Repo struct {
		db     connectors.SQLConnector
		logger *zap.Logger
	}
)

const (
	selectNowSQL = `SELECT NOW();`
)

func NewRepo(conns *conns.Conns, logger *zap.Logger) (*Repo, error) {
	db, err := conns.GetSQLConn()
	if err != nil {
		return nil, fmt.Errorf("create repo err: %w", err)
	}

	return &Repo{
		db:     db,
		logger: logger,
	}, nil
}

func (r *Repo) GetNow(ctx context.Context) (string, error) {
	now := ""

	if err := r.db.CallContext(ctx, "get_now", func(ctx context.Context, db *sqlx.DB) error {
		if err := db.QueryRowContext(ctx, selectNowSQL).Scan(&now); err != nil {
			return fmt.Errorf("failed to execute query for get now: %w", err)
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("failed for get now: %w", err)
	}

	return now, nil
}
