package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type (
	ReleaseID string

	Config struct {
		HTTP          HTTPServerConfig
		GRPC          GRPCServerConfig
		APM           ElasticAPMConfig
		Postgres      PostgresConfig
		ElasticSearch ElasticSearch
		Sentry        Sentry

		ReleaseID string
		Env       string `envconfig:"ENV" default:"development" required:"true"`
		LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
		AppName   string `envconfig:"APP_NAME" required:"true"`

		// DebugLog включает/выключает полные логи ответов (response payload).
		DebugLog bool `envconfig:"DEBUG_LOG" default:"false"`
	}

	Sentry struct {
		DSN string `envconfig:"SENTRY_DSN"`
	}

	ElasticSearch struct {
		// DSN представляет из себя строку, где URI:port нод кластера перечислены через запятую без пробелов
		// Например: 'http://es1.localhost.com:9200,http://es2.localhost.com:9200'
		DSN        string `envconfig:"ELASTIC_SEARCH_DSN"`
		MaxRetries int    `envconfig:"ELASTIC_SEARCH_MAX_RETRIES" default:"5" required:"true"`
		WithLogger bool   `envconfig:"ELASTIC_WITH_LOGGER" default:"false" required:"true"`
	}

	GRPCServerConfig struct {
		ListenAddr               string        `envconfig:"GRPC_LISTEN_ADDR" required:"true"`
		KeepaliveTime            time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"30s"`
		KeepaliveTimeout         time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"10s"`
		RegisterReflectionServer bool          `envconfig:"GRPC_REGISTER_REFLECTION_SERVER" default:"true"`
	}

	HTTPServerConfig struct {
		ListenAddr       string        `envconfig:"HTTP_LISTEN_ADDR" required:"true"`
		KeepaliveTime    time.Duration `envconfig:"HTTP_KEEPALIVE_TIME" default:"30s"`
		KeepaliveTimeout time.Duration `envconfig:"HTTP_KEEPALIVE_TIMEOUT" default:"10s"`
	}

	ElasticAPMConfig struct {
		ServiceName string `envconfig:"ELASTIC_APM_SERVICE_NAME" required:"true"`
		ServerURL   string `envconfig:"ELASTIC_APM_SERVER_URL" required:"true"`
		Environment string `envconfig:"ELASTIC_APM_ENVIRONMENT" required:"true"`
	}

	PostgresConfig struct {
		DSN                string        `envconfig:"DB_DSN"`
		MaxOpenConnections int           `envconfig:"DB_MAX_OPEN_CONNECTIONS" default:"25" required:"true"`
		MaxIdleConnections int           `envconfig:"DB_MAX_IDLE_CONNECTIONS" default:"25" required:"true"`
		ConnMaxLifetime    time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m" required:"true"`
	}
)

func NewConfig(id ReleaseID) (*Config, error) {
	config := &Config{}

	if err := envconfig.Process("", config); err != nil {
		return nil, fmt.Errorf("create config err %w", err)
	}

	config.ReleaseID = string(id)

	return config, nil
}
