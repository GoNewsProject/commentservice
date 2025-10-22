package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Name         string `yaml:"name"`
	Env          string `yaml:"env"`
	Version      string `yaml:"version"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DBConfig struct {
	Host                string        `yaml:"host"`
	Port                int           `yaml:"port"`
	User                string        `yaml:"user"`
	Password            string        `yaml:"password"`
	Name                string        `yaml:"name"`
	Migrations          string        `yaml:"migrations"`
	SSLMode             string        `yaml:"sslmode"`
	PoolMaxConns        string        `yaml:"pool_max_conns"`
	PoolMinConns        string        `yaml:"pool_min_conns"`
	PoolMaxConLifeTime  string        `yaml:"pool_max-conn_lifetime"`
	PoolMaxConnIdletime string        `yaml:"pool_max_conn_idle_time"`
	ConnetTimeout       time.Duration `yaml:"connect_timeout"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type KafkaConfig struct {
	Brokers       []string          `yaml:"brokers"`
	Topics        map[string]string `yaml:"topics"`
	ConsumerGroup string            `yaml:"consumer_group"`
}

type Config struct {
	App     AppConfig     `yaml:"app"`
	HTTP    HTTPConfig    `yaml:"http"`
	DB      DBConfig      `yaml:"database"`
	Logging LoggingConfig `yaml:"logging"`
	Kafka   KafkaConfig   `yaml:"kafka"`
}

func (c *Config) GetAppName() string {
	return c.App.Name
}

func (c *Config) GetAppEnv() string {
	return c.App.Env
}

func (c *Config) GetAppVersion() string {
	return c.App.Version
}

func (c *Config) GetHost() string {
	return c.HTTP.Host
}

func (c *Config) GetPort() int {
	return c.HTTP.Port
}

func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.App.ReadTimeout) * time.Second
}

func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.App.WriteTimeout) * time.Second
}

func (c *Config) GetTopic(name string) (string, error) {
	if topic, ok := c.Kafka.Topics[name]; ok {
		return topic, nil
	}
	return "", fmt.Errorf("topic %s not found", name)
}

// LoadConfig загружает конфиг из файла.
func (c *Config) LoadConfig(configPath string) (*Config, error) {
	appEnv := os.Getenv("APP_ENV")

	if appEnv != "prod" && appEnv != "production" {
		if err := godotenv.Load(); err != nil {
			if err = godotenv.Load("./comments/.env"); err != nil {
				return nil, fmt.Errorf("failed to load .env file: %w", err)
			}
		}
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err = yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	return &cfg, nil
}
