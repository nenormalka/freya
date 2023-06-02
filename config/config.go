package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type (
	ReleaseID string

	Configurator func(cfg *Config) error

	Config struct {
		HTTP          HTTPServerConfig `yaml:"http"`
		GRPC          GRPCServerConfig `yaml:"grpc"`
		APM           ElasticAPMConfig `yaml:"apm"`
		Postgres      PostgresConfig   `yaml:"postgres"`
		ElasticSearch ElasticSearch    `yaml:"elastic_search"`
		Sentry        Sentry           `yaml:"sentry"`

		ReleaseID string
		Env       string `envconfig:"ENV" default:"development" required:"true" yaml:"env"`
		LogLevel  string `envconfig:"LOG_LEVEL" default:"info" yaml:"log_level"`
		AppName   string `envconfig:"APP_NAME" required:"true" yaml:"app_name"`

		// DebugLog включает/выключает полные логи ответов (response payload).
		DebugLog bool `envconfig:"DEBUG_LOG" default:"false" yaml:"debug_log"`
	}

	Sentry struct {
		DSN string `envconfig:"SENTRY_DSN" yaml:"sentry_dsn"`
	}

	ElasticSearch struct {
		// DSN представляет из себя строку, где URI:port нод кластера перечислены через запятую без пробелов
		// Например: 'http://es1.localhost.com:9200,http://es2.localhost.com:9200'
		DSN        string `envconfig:"ELASTIC_SEARCH_DSN" yaml:"dsn"`
		MaxRetries int    `envconfig:"ELASTIC_SEARCH_MAX_RETRIES" default:"5" required:"true" yaml:"max_retries"`
		WithLogger bool   `envconfig:"ELASTIC_WITH_LOGGER" default:"false" required:"true" yaml:"with_logger"`
	}

	GRPCServerConfig struct {
		ListenAddr               string        `envconfig:"GRPC_LISTEN_ADDR" required:"true" yaml:"listen_addr"`
		KeepaliveTime            time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"30s" yaml:"keepalive_time"`
		KeepaliveTimeout         time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"10s" yaml:"keepalive_timeout"`
		RegisterReflectionServer bool          `envconfig:"GRPC_REGISTER_REFLECTION_SERVER" default:"true" yaml:"register_reflection_server"`
	}

	HTTPServerConfig struct {
		ListenAddr       string        `envconfig:"HTTP_LISTEN_ADDR" required:"true" yaml:"listen_addr"`
		KeepaliveTime    time.Duration `envconfig:"HTTP_KEEPALIVE_TIME" default:"30s" yaml:"keepalive_time"`
		KeepaliveTimeout time.Duration `envconfig:"HTTP_KEEPALIVE_TIMEOUT" default:"10s" yaml:"keepalive_timeout"`
	}

	ElasticAPMConfig struct {
		ServiceName string `envconfig:"ELASTIC_APM_SERVICE_NAME" required:"true" yaml:"service_name"`
		ServerURL   string `envconfig:"ELASTIC_APM_SERVER_URL" required:"true" yaml:"server_url"`
		Environment string `envconfig:"ELASTIC_APM_ENVIRONMENT" required:"true" yaml:"environment"`
	}

	PostgresConfig struct {
		DSN                string        `envconfig:"DB_DSN" yaml:"dsn"`
		MaxOpenConnections int           `envconfig:"DB_MAX_OPEN_CONNECTIONS" default:"25" required:"true" yaml:"max_open_connections"`
		MaxIdleConnections int           `envconfig:"DB_MAX_IDLE_CONNECTIONS" default:"25" required:"true" yaml:"max_idle_connections"`
		ConnMaxLifetime    time.Duration `envconfig:"DB_CONN_MAX_LIFETIME" default:"5m" required:"true" yaml:"conn_max_lifetime"`
	}
)

const (
	yamlPathConfig = "CONFIG_YAML_FILE"
)

var (
	flagConfig = flag.String("config-file", "", "config file")
)

var (
	errConfigFileNotExists = errors.New("config file not exists")
)

func NewConfig(configurators []Configurator, releaseID ReleaseID) (*Config, error) {
	cfg := &Config{}
	cfg.ReleaseID = string(releaseID)

	for _, loader := range configurators {
		if err := loader(cfg); err != nil {
			return nil, fmt.Errorf("create config err %w", err)
		}
	}

	return cfg, nil
}

func loadYAML(cfg *Config) error {
	filename := ""

	envPath := os.Getenv(yamlPathConfig)

	if envPath != "" {
		filename = envPath
	}

	flag.Parse()

	if *flagConfig != "" {
		filename = *flagConfig
	}

	if filename == "" {
		return nil
	}

	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return errConfigFileNotExists
	}

	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	defer func() { _ = f.Close() }()

	if err = yaml.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("error decoding config file: %w", err)
	}

	return nil
}

func loadENV(cfg *Config) error {
	if err := envconfig.Process("", cfg); err != nil {
		return fmt.Errorf("parse env config err %w", err)
	}

	return nil
}
