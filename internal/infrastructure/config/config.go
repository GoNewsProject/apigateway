package infrastructure

import (
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig - конфигурация приложения.
type AppConfig struct {
	Name               string    `yaml:"name"`
	ReadTimeout        int       `yaml:"read_timeout"`
	WriteTimeout       int       `yaml:"write_timeout"`
	ConnectTimeout     int       `yaml:"connect_timeout"`
	DefaultNewsLimit   int       `yaml:"default_news_limit"`
	ProcessingInterval int       `yaml:"processingInterval"`
	FeedURLs           []FeedURL `yaml:"feed_urls"`
}

type FeedURL struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// HTTPConfig - конфигурация HTTP сервера.
type HTTPConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LoggingConfig - конфигурация логирования.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Route - конфигурация сервисов для роутинга
type Route struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
}

type KafkaConfig struct {
	Brokers        []string          `yaml:"brokers"`
	Topics         KafkaTopics       `yaml:"topics"`
	ConsumerGroups map[string]string `yaml:"consumer_groups"`
}
type KafkaTopics struct {
	// Producers
	NewsInput     string `yaml:"news_input"`
	CommentsInput string `yaml:"comments_input"`
	AddComments   string `yaml:"add_comments"`

	// Consumers
	NewsDetail      string `yaml:"news_detail"`
	NewsList        string `yaml:"news_list"`
	FilteredContent string `yaml:"filtered_content"`
	FilterPublished string `yaml:"filter_published"`
	Comments        string `yaml:"comments"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"sslmode"`
}

// Config основная конфигурация.
type Config struct {
	App     AppConfig     `yaml:"app"`
	HTTP    HTTPConfig    `yaml:"http"`
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

// LoadConfig загружает конфиг из файла.
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		log.Println("Config path is empty")
		return nil, fmt.Errorf("config path is empty")
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		log.Println("Failed to read config file")
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(raw))

	var cfg Config
	if err = yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config yaml: %w", err)
	}

	return &cfg, nil
}
