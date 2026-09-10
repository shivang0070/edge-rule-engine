package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Actions  ActionsConfig  `yaml:"actions"`
	Logging  LoggingConfig  `yaml:"logging"`
	Engine   EngineConfig   `yaml:"engine"`
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type ActionsConfig struct {
	Directory string `yaml:"directory"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type EngineConfig struct {
	MaxActiveRules         int `yaml:"maxActiveRules"`
	DefaultCooldownSeconds int `yaml:"defaultCooldownSeconds"`
	StateRetentionHours    int `yaml:"stateRetentionHours"`
	EventRetentionHours    int `yaml:"eventRetentionHours"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Address: ":8080",
		},
		Database: DatabaseConfig{
			Path: "./engine.db",
		},
		Actions: ActionsConfig{
			Directory: "./actions",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Engine: EngineConfig{
			MaxActiveRules:         50,
			DefaultCooldownSeconds: 300,
			StateRetentionHours:    24,
			EventRetentionHours:    24,
		},
	}
}

func Load(path string) (*Config, error) {
	config := Default()

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Return defaults if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}
