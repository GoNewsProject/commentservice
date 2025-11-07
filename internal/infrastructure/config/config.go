package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	HTTP      HTTPConfig      `yaml:"http"`
	Databases DatabasesConfig `yaml:"databases"`
	Logging   LoggingConfig   `yaml:"logging"`
	Kafka     KafkaConfig     `yaml:"kafka"`
	Routes    []Route         `yaml:"routes"`
}

type AppConfig struct {
	Name                string `yaml:"name"`
	ReadTimeout         int    `yaml:"read_timeout"`
	WriteTimeout        int    `yaml:"write_timeout"`
	ConnectTimeout      int    `yaml:"connect_timeout"`
	DefaultCommentLimit int    `yaml:"default_comment_limit"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	UserName string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"sslmode"`
}

type DatabasesConfig struct {
	News     DBConfig `yaml:"news"`
	Comments DBConfig `yaml:"comments"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type Route struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
}

type KafkaTopics struct {
	CommentInput string `yaml:"comment_input"`
	AddComment   string `yaml:"add_comment"`

	Comments string `yaml:"comments"`
}

type KafkaConfig struct {
	Brokers       []string          `yaml:"brokers"`
	Topics        KafkaTopics       `yaml:"topics"`
	ConsumerGroup map[string]string `yaml:"consumer_group"`
}

// LoadConfig загружает конфиг из файла.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		log.Println("Config file path is empty")
		return nil, fmt.Errorf("config file path is empty")
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("failed to read config file")
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		log.Println("failed to parse confi yaml")
		return nil, fmt.Errorf("failed to parse config yaml: %w", err)
	}

	if err := cfg.Databases.Comments.Validate(); err != nil {
		return nil, fmt.Errorf("validation comments database config failed: %w", err)
	}
	if err := cfg.Databases.News.Validate(); err != nil {
		return nil, fmt.Errorf("validation news database config failed : %w", err)
	}

	log.Printf("config loaded successfully from %s", configPath)
	return &cfg, nil
}

// Метод для получения DSN строки подключения
func (db *DBConfig) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.UserName,
		db.Password,
		db.Host,
		db.Port,
		db.DBName,
		db.SSLMode,
	)
}

// Метод для проверки валидности конфигурации БД
func (db *DBConfig) Validate() error {
	if db.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if db.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if db.UserName == "" {
		return fmt.Errorf("database username is required")
	}
	if db.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	return nil
}

// Универсальный метод
func (c *Config) GetTopic(name string) (string, error) {
	switch name {
	case "comment_input":
		return c.Kafka.Topics.CommentInput, nil
	case "add_comment":
		return c.Kafka.Topics.AddComment, nil
	case "comments":
		return c.Kafka.Topics.Comments, nil
	default:
		return "", fmt.Errorf("topic %s not found", name)
	}
}

func (c *Config) GetNewsDBConfig() DBConfig {
	return c.Databases.News
}

func (c *Config) GetCommentsDBConfig() DBConfig {
	return c.Databases.Comments
}

func (c *Config) GetAppName() string {
	return c.App.Name
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

func (c *Config) GetCommentInputTopic() string {
	return c.Kafka.Topics.CommentInput
}

func (c *Config) GetAddCommentTopic() string {
	return c.Kafka.Topics.AddComment
}

func (c *Config) GetCommentsTopic() string {
	return c.Kafka.Topics.Comments
}
