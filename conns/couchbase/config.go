package couchbase

import (
	"strings"

	"github.com/nenormalka/freya/config"
)

type Config struct {
	DSN         string
	User        string
	Password    string
	Buckets     []string
	AppName     string
	EnableDebug bool
}

func NewConfig(cfg *config.Config) Config {
	return Config{
		DSN:         cfg.CouchbaseConfig.DSN,
		User:        cfg.CouchbaseConfig.User,
		Password:    cfg.CouchbaseConfig.Password,
		Buckets:     strings.Split(cfg.CouchbaseConfig.Buckets, ","),
		AppName:     cfg.AppName,
		EnableDebug: cfg.CouchbaseConfig.EnableDebug,
	}
}
