package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Logger     LoggerConfig     `mapstructure:"logger"`
	HTTPServer HTTPServerConfig `mapstructure:"httpServer"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Timezone   string           `mapstructure:"timezone"`
	Auth       AuthConfig       `mapstructure:"auth"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

type HTTPServerConfig struct {
	Host        string        `mapstructure:"host"`
	Port        string        `mapstructure:"port"`
	ReadTimeout time.Duration `mapstructure:"readTimeout"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	URI    string `mapstructure:"uri"`
}

type AuthConfig struct {
	PrivateKey string `mapstructure:"privateKey"`
}

func NewConfig(path string) (*Config, error) {
	parser := viper.New()
	parser.SetConfigFile(path)

	err := parser.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	for _, key := range parser.AllKeys() {
		value := parser.GetString(key)
		parser.Set(key, os.ExpandEnv(value))
	}

	var config Config
	err = parser.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("cannot parse config file: %w", err)
	}

	return &config, err
}
