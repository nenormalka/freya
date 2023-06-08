package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"

	"github.com/nenormalka/freya/conns/connectors"
)

type (
	ReleaseID string

	Configure func(cfg *Config) error

	Config struct {
		HTTP          HTTPServerConfig `yaml:"http"`
		GRPC          GRPCServerConfig `yaml:"grpc"`
		APM           ElasticAPMConfig `yaml:"apm"`
		DB            []DB             `yaml:"db"`
		ElasticSearch ElasticSearch    `yaml:"elastic_search"`
		Sentry        Sentry           `yaml:"sentry"`

		ReleaseID string
		Env       string `envconfig:"ENV" default:"development" required:"true" yaml:"env"`
		LogLevel  string `envconfig:"LOG_LEVEL" default:"info" yaml:"log_level"`
		AppName   string `envconfig:"APP_NAME" yaml:"app_name"`

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
		ListenAddr               string        `envconfig:"GRPC_LISTEN_ADDR" required:"true" default:":9090" yaml:"listen_addr"`
		KeepaliveTime            time.Duration `envconfig:"GRPC_KEEPALIVE_TIME" default:"30s" yaml:"keepalive_time"`
		KeepaliveTimeout         time.Duration `envconfig:"GRPC_KEEPALIVE_TIMEOUT" default:"10s" yaml:"keepalive_timeout"`
		RegisterReflectionServer bool          `envconfig:"GRPC_REGISTER_REFLECTION_SERVER" default:"true" yaml:"register_reflection_server"`
	}

	HTTPServerConfig struct {
		ListenAddr       string        `envconfig:"HTTP_LISTEN_ADDR" required:"true" default:":8080" yaml:"listen_addr"`
		KeepaliveTime    time.Duration `envconfig:"HTTP_KEEPALIVE_TIME" default:"30s" yaml:"keepalive_time"`
		KeepaliveTimeout time.Duration `envconfig:"HTTP_KEEPALIVE_TIMEOUT" default:"10s" yaml:"keepalive_timeout"`
	}

	ElasticAPMConfig struct {
		ServiceName string `envconfig:"ELASTIC_APM_SERVICE_NAME" yaml:"service_name"`
		ServerURL   string `envconfig:"ELASTIC_APM_SERVER_URL" yaml:"server_url"`
		Environment string `envconfig:"ELASTIC_APM_ENVIRONMENT" yaml:"environment"`
	}

	DB struct {
		DSN                string        `yaml:"dsn"`
		Name               string        `yaml:"name"`
		MaxOpenConnections int           `yaml:"max_open_connections"`
		MaxIdleConnections int           `yaml:"max_idle_connections"`
		ConnMaxLifetime    time.Duration `yaml:"conn_max_lifetime"`
	}
)

const (
	yamlPathConfig       = "CONFIG_YAML_FILE"
	defaultDBDSN         = "DB_DSN"
	maxOpenConnectionsDB = 25
	maxIdleConnectionsDB = 5
)

var (
	flagConfig = flag.String("config-file", "", "config file")
)

var (
	errConfigFileNotExists = errors.New("config file not exists")
)

func NewConfig(configurators []Configure, releaseID ReleaseID) (*Config, error) {
	cfg := &Config{}
	cfg.ReleaseID = string(releaseID)

	for _, configurator := range configurators {
		if err := configurator(cfg); err != nil {
			return nil, fmt.Errorf("create config err %w", err)
		}
	}

	return cfg, nil
}

func loadYAML(cfg *Config) error {
	filename := ""

	for _, f := range []func(){
		func() {
			envPath := os.Getenv(yamlPathConfig)

			if envPath != "" {
				filename = envPath
			}
		},
		func() {
			flag.Parse()

			if *flagConfig != "" {
				filename = *flagConfig
			}
		},
	} {
		f()

		if filename != "" {
			break
		}
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
	err := envconfig.Process("", cfg)
	if err != nil {
		return fmt.Errorf("parse env config err %w", err)
	}

	cfg.DB = getDBConnsENV()

	return nil
}

func getEnvParam(param string, defaultValue int) int {
	envParam := os.Getenv(param)
	if envParam == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(envParam)
	if err != nil {
		return defaultValue
	}

	return value
}

func getDBConnsENV() []DB {
	var dbConns []DB

	maxOpenConnections := getEnvParam("DB_MAX_OPEN_CONNECTIONS", maxOpenConnectionsDB)
	maxIdleConnections := getEnvParam("DB_MAX_IDLE_CONNECTIONS", maxIdleConnectionsDB)

	for _, pair := range os.Environ() {
		if !strings.HasPrefix(pair, defaultDBDSN) {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := connectors.DefaultDBConn
		if parts[0] != defaultDBDSN {
			name = strings.ToLower(strings.TrimPrefix(parts[0], defaultDBDSN+"_"))
		}

		dbConns = append(dbConns, DB{
			DSN:                parts[1],
			Name:               name,
			MaxOpenConnections: maxOpenConnections,
			MaxIdleConnections: maxIdleConnections,
			ConnMaxLifetime:    time.Minute * 5,
		})
	}

	return dbConns
}
