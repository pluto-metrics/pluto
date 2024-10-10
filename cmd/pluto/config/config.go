package config

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mcuadros/go-defaults"
	"go.uber.org/zap"
)

type Config struct {
	ClickHouse struct {
		DSN      string `default:"http://127.0.0.1:8123" validate:"uri"`
		Database string `default:"default"`
	}

	Insert struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9095" validate:"hostname_port"`
		IDFunc  string `default:"name_with_sha256" validate:"oneof=name_with_sha256"`
		Table   string `default:"samples_null"`
	}

	Select struct {
		TableSeries          string        `default:"series"`
		TableSamples         string        `default:"samples"`
		AutocompleteLookback time.Duration `default:"168h"`
		SeriesPartitionMs    int64         `default:"86400000"`
	}

	Prometheus struct {
		Enabled                    bool          `default:"true"`
		Listen                     string        `default:"0.0.0.0:9096" comment:"listen addr for prometheus ui and api"`
		ExternalURL                string        `default:"http://127.0.0.1:9096" comment:"allows to set URL for redirect manually"`
		PageTitle                  string        `default:"Pluto"`
		LookbackDelta              time.Duration `default:"5m"`
		RemoteReadConcurrencyLimit int           `default:"10" comment:"concurrently handled remote read requests"`
	}

	Debug struct {
		Enabled bool   `default:"true"`
		Listen  string `default:"0.0.0.0:9095"`
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
	defaults.SetDefaults(&cfg)
	// if err := configor.Load(&cfg, filename); err != nil {
	// 	return nil, err
	// }
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
