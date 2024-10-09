package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
)

type ClickHouse struct {
	DSN      string `default:"http://127.0.0.1:8123" validate:"uri"`
	Database string `default:"default"`
	Table    string `default:"samples_null"`
}

type Config struct {
	Insert struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9095" validate:"hostname_port"`
		IDFunc  string `default:"name_with_sha256" validate:"oneof=name_with_sha256"`
		Target  ClickHouse
	}

	Select struct {
	}

	Debug struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9095" validate:"hostname_port"`
		Pprof   bool   `default:"true"`
		Metrics bool   `default:"true"`
	}

	Logging zap.Config
}

func LoadFromFile(filename string, development bool) (*Config, error) {
	cfg := Config{}

	if development {
		cfg.Logging = zap.NewDevelopmentConfig()
	} else {
		cfg.Logging = zap.NewProductionConfig()
	}
	if err := configor.Load(&cfg, filename); err != nil {
		return nil, err
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
