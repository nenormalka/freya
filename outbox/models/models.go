package models

import "context"

type (
	DataStatus string
	DataSource string

	Data struct {
		Code   string
		Source DataSource
		Data   []byte
	}

	Job func(ctx context.Context)
)

const (
	DataStatusNew        = "new"
	DataStatusSent       = "sent"
	DataStatusProcessing = "processing"
	DataStatusFailed     = "failed"
)
