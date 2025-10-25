package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

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
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	UserName   string `yaml:"user_name"`
	Password   string `yaml:"password"`
	DBName     string `yaml:"db_name"`
	Migrations string `yaml:"migrations"`
	SSLMode    string `yaml:"sslmode"`
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

type Config struct {
	App     AppConfig     `yaml:"app"`
	HTTP    HTTPConfig    `yaml:"http"`
	DB      DBConfig      `yaml:"database"`
	Logging LoggingConfig `yaml:"logging"`
	Kafka   KafkaConfig   `yaml:"kafka"`
	Routes  []Route       `yaml:"routes"`
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

// LoadConfig загружает конфиг из файла.
func (c *Config) LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		log.Println("Config file is empty")
		return nil, fmt.Errorf("config file is empty")
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Failed to read config file")
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		log.Println("failed to parse confi yaml")
		return nil, fmt.Errorf("failed to parse config yaml: %w", err)
	}

	return &cfg, nil
}
