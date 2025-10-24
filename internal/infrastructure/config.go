package infrastructure

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// AppConfig - конфигурация приложения.
type AppConfig struct {
	Name           string `yaml:"name"`
	Env            string `yaml:"env"`
	Version        string `yaml:"version"`
	ReadTimeout    int    `yaml:"read_timeout"`
	WriteTimeout   int    `yaml:"write_timeout"`
	ConnectTimeout int    `yaml:"connect_timeout"`
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

// Config основная конфигурация.
type Config struct {
	App     AppConfig     `yaml:"app"`
	HTTP    HTTPConfig    `yaml:"http"`
	Logging LoggingConfig `yaml:"logging"`
	Routes  []Route       `yaml:"routes"`
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "dev"
}

func (c *Config) GetAppName() string {
	return c.App.Name
}

func (c *Config) GetVersion() string {
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

// LoadConfig загружает конфиг из файла.
func LoadConfig(configPath string) (*Config, error) {
	appEnv := os.Getenv("APP_ENV")

	if appEnv != "prod" && appEnv != "production" {
		if err := godotenv.Load(); err != nil {
			if err = godotenv.Load("./apigateway/.env"); err != nil {
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
		return nil, fmt.Errorf("failed to parse config yaml: %w", err)
	}

	cfg.overrideRoutesForDev()

	return &cfg, nil
}

// overrideRoutesForDev переписывает пути для локальной разработки.
func (c *Config) overrideRoutesForDev() {
	if c.App.Env != "dev" {
		return
	}

	for i, route := range c.Routes {
		u, err := url.Parse(route.BaseURL)
		if err != nil {
			continue
		}

		port := u.Port()
		if port == "" {
			port = "80"
		}

		c.Routes[i].BaseURL = fmt.Sprintf("http://localhost:%s", port)
	}
}
