package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/configor"
)

type ClickHouse struct {
	DSN      string `default:"http://127.0.0.1:8123" validator:"uri"`
	Database string `default:"default"`
	Table    string `default:"samples"`
}

type Config struct {
	Insert struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9095" validator:"hostname_port"`
		IDFunc  string `default:"name_with_sha256" validator:"oneof name_with_sha256"`
		Target  ClickHouse
	}

	Select struct {
	}

	Debug struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9096" validator:"hostname_port"`
		Pprof   bool   `default:"true"`
		Metrics bool   `default:"true"`
	}
}

func LoadFromFile(filename string) (*Config, error) {
	cfg := Config{}
	if err := configor.Load(&cfg, filename); err != nil {
		return nil, err
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
