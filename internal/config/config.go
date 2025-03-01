package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel string         `yaml:"log_level"`
	Database DatabaseConfig `yaml:"database"`
	HTTP     HTTPConfig     `yaml:"http"`
}

type DatabaseConfig struct {
	DSN   string `yaml:"dsn"`
	Table string `yaml:"table"`
}

type HTTPConfig struct {
	Address string `yaml:"address"`
}

func Load(filename string) (Config, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer fh.Close()

	var config Config
	if err := yaml.NewDecoder(fh).Decode(&config); err != nil {
		return Config{}, err
	}

	return config, nil
}
