package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nenormalka/freya/conns/connectors"
	"github.com/nenormalka/freya/types"

	"github.com/kelseyhightower/envconfig"
	lilith "github.com/nenormalka/lilith/methods"
	"gopkg.in/yaml.v3"
)

type (
	Configure func(cfg *Config) error

	Config struct {
		HTTP            HTTPServerConfig    `yaml:"http"`
		GRPC            GRPCServerConfig    `yaml:"grpc"`
		APM             ElasticAPMConfig    `yaml:"apm"`
		Kafka           KafkaConfig         `yaml:"kafka"`
		DB              []DB                `yaml:"db"`
		ElasticSearch   ElasticSearch       `yaml:"elastic_search"`
		Sentry          Sentry              `yaml:"sentry"`
		CouchbaseConfig CouchbaseConfig     `yaml:"couchbase"`
		ConsulConfig    ConsulConfig        `yaml:"consul"`
		RedisConfig     RedisConfig         `yaml:"redis"`
		ThrottleConfig  ThrottleConfig      `yaml:"throttle"`
		Communication   CommunicationConfig `yaml:"communication"`

		ReleaseID string
		Env       string `envconfig:"ENV" default:"development" required:"true" yaml:"env"`
		LogLevel  string `envconfig:"LOG_LEVEL" default:"info" yaml:"log_level"`
		AppName   string `envconfig:"APP_NAME" yaml:"app_name"`

		// DebugLog включает/выключает полные логи ответов (response payload).
		DebugLog            bool `envconfig:"DEBUG_LOG" default:"false" yaml:"debug_log"`
		EnableServerMetrics bool `envconfig:"ENABLE_SERVER_METRICS" default:"true" yaml:"enable_server_metrics"`
	}

	ThrottleConfig struct {
		EnableThrottle bool   `envconfig:"THROTTLE_ENABLE" default:"false" yaml:"enable_throttle"`
		Methods        string `envconfig:"THROTTLE_METHODS" yaml:"methods"`
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
		MaxReceiveMessageSize    int           `envconfig:"GRPC_MAX_RECEIVE_MESSAGE_SIZE" default:"10485760" yaml:"max_receive_message_size"` // 10MB
		MaxSendMessageSize       int           `envconfig:"GRPC_MAX_SEND_MESSAGE_SIZE" default:"104857600" yaml:"max_send_message_size"`      // 100MB
		WithMetadata             bool          `envconfig:"GRPC_WITH_METADATA" default:"false" yaml:"with_metadata"`
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

	KafkaConfig struct {
		Addresses   string `envconfig:"KAFKA_ADDRESSES" yaml:"addresses"`
		SkipErrors  string `envconfig:"KAFKA_SKIP_ERRORS" yaml:"skip_errors"`
		EnableDebug bool   `envconfig:"KAFKA_ENABLE_DEBUG" default:"false" yaml:"enable_debug"`
	}

	DB struct {
		DSN  string `yaml:"dsn"`
		Name string `yaml:"name"`
		// ConnType sqlx|pgx
		ConnType           string        `yaml:"conntype"`
		MaxOpenConnections int           `yaml:"max_open_connections"`
		MaxIdleConnections int           `yaml:"max_idle_connections"`
		ConnMaxLifetime    time.Duration `yaml:"conn_max_lifetime"`
		// DBType postgres/mssql/mysql
		DBType string `yaml:"db_type"`
	}

	CouchbaseConfig struct {
		DSN         string `envconfig:"COUCHBASE_DSN" yaml:"dsn"`
		User        string `envconfig:"COUCHBASE_USER" yaml:"user"`
		Password    string `envconfig:"COUCHBASE_PWD" yaml:"password"`
		Buckets     string `envconfig:"COUCHBASE_BUCKET" yaml:"bucket"`
		EnableDebug bool   `envconfig:"COUCHBASE_ENABLE_DEBUG" default:"false" yaml:"enable_debug"`
	}

	ConsulConfig struct {
		Address            string        `envconfig:"CONSUL_ADDRESS" yaml:"address"`
		Scheme             string        `envconfig:"CONSUL_SCHEME" default:"http" yaml:"proto"`
		Token              string        `envconfig:"CONSUL_TOKEN" yaml:"token"`
		InsecureSkipVerify bool          `envconfig:"CONSUL_INSECURE_SKIP_VERIFY" default:"true" yaml:"insecure_skip_verify"`
		SessionTTL         string        `envconfig:"CONSUL_SESSION_TTL" default:"20s" yaml:"session_ttl"`
		LeaderTTL          time.Duration `envconfig:"CONSUL_LEADER_TTL" default:"15s" yaml:"leader_ttl"`
		ConsulServiceName  string        `envconfig:"CONSUL_SERVICE_NAME" yaml:"service_name"`
	}

	RedisConfig struct {
		KeyDBDSN string `envconfig:"KEYDB_DSN" yaml:"key_dsn"`
		RedisDSN string `envconfig:"REDIS_DSN" yaml:"redis_dsn"`
		PoolSize int    `envconfig:"REDIS_POOL_SIZE" default:"10" yaml:"pool_size"`
	}

	CommunicationConfig struct {
		Enabled bool `envconfig:"COMMUNICATION_ENABLED" default:"false" yaml:"enabled"`
		// TransportType (kafka,grpc)
		TransportType string `envconfig:"COMMUNICATION_TRANSPORT_TYPE" default:"grpc" yaml:"transport_type"`
		KafkaTopic    string `envconfig:"COMMUNICATION_KAFKA_TOPIC" yaml:"kafka_topic"`
		// GRPCAddresses адреса grpc, которые потому разбираются в map[string(serviceName)][]string(addresses)
		// Например: "service1=address1:port1,address2:port2;service2=address3:port3"
		GRPCAddresses string `envconfig:"COMMUNICATION_GRPC_ADDRESS" yaml:"grpc_address"`
	}
)

const (
	yamlPathConfig       = "CONFIG_YAML_FILE"
	defaultDBDSN         = "DB_DSN"
	mssqlDBDSN           = "MSSQL_DSN"
	maxOpenConnectionsDB = 25
	maxIdleConnectionsDB = 5
)

var (
	flagConfig = flag.String("config-file", "", "config file")
)

var (
	errConfigFileNotExists = errors.New("config file not exists")
)

func NewConfig(configurators []Configure, info *types.AppInfo) (*Config, error) {
	types.SetAppInfo(info)

	cfg := &Config{}
	cfg.ReleaseID = types.GetAppVersion()

	for _, configurator := range configurators {
		if err := configurator(cfg); err != nil {
			return nil, fmt.Errorf("create config err %w", err)
		}
	}

	types.SetApplicationMetrics()

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

	defer func() {
		if err = f.Close(); err != nil {
			fmt.Printf("error closing config file: %v", err)
		}
	}()

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

func getEnvParamInt(param string, defaultValue int) int {
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

func getEnvParamStr(param string, defaultValue string) string {
	envParam := os.Getenv(param)
	if envParam == "" {
		return defaultValue
	}

	return envParam
}

func getDBConnsENV() []DB {
	var dbConns []DB

	maxOpenConnections := getEnvParamInt("DB_MAX_OPEN_CONNECTIONS", maxOpenConnectionsDB)
	maxIdleConnections := getEnvParamInt("DB_MAX_IDLE_CONNECTIONS", maxIdleConnectionsDB)
	connType := getEnvParamStr("DB_TYPE", "")
	if connType == "" {
		connType = connectors.SqlxConnType
	}

	for _, pair := range os.Environ() {
		if !strings.HasPrefix(pair, defaultDBDSN) && !strings.HasPrefix(pair, mssqlDBDSN) {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		dbType := connectors.PostgresDBType
		name := ""

		if parts[0] == defaultDBDSN {
			name = connectors.DefaultDBConn
		} else if strings.HasPrefix(parts[0], mssqlDBDSN) {
			name = strings.ToLower(strings.TrimPrefix(parts[0], mssqlDBDSN+"_"))
			dbType = connectors.MsSQLDBType
		} else {
			name = strings.ToLower(strings.TrimPrefix(parts[0], defaultDBDSN+"_"))
		}

		dbConns = append(dbConns, DB{
			DSN:                parts[1],
			Name:               name,
			MaxOpenConnections: maxOpenConnections,
			MaxIdleConnections: maxIdleConnections,
			ConnType:           lilith.Ternary(dbType == connectors.MsSQLDBType, connectors.SqlxConnType, connType),
			DBType:             dbType,
			ConnMaxLifetime:    time.Minute * 5,
		})
	}

	return dbConns
}
