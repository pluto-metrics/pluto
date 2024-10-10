package config

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
)

type ClickHouse struct {
	DSN    string            `yaml:"dsn" validate:"uri" default:"http://127.0.0.1:8123/?async_insert=1&wait_for_async_insert=1"`
	Params map[string]string `yaml:"params"`
}

type Config struct {
	ClickHouse ClickHouse `yaml:"clickhouse"`

	Insert struct {
		Enabled bool   `yaml:"enabled" default:"true"`
		Listen  string `yaml:"listen" default:"0.0.0.0:9095" validate:"hostname_port"`
		IDFunc  string `yaml:"id_func" default:"name_with_sha256" validate:"oneof=name_with_sha256"`
		Table   string `yaml:"table" default:"samples_null"`
	} `yaml:"insert"`

	Select struct {
		TableSeries          string        `yaml:"table_series"  default:"series"`
		TableSamples         string        `yaml:"table_samples" default:"samples"`
		AutocompleteLookback time.Duration `yaml:"autocomplete_lookback" default:"168h"`
		SeriesPartitionMs    int64         `yaml:"series_partition_ms" default:"86400000"`
	} `yaml:"select"`

	Prometheus struct {
		Enabled                    bool          `yaml:"enabled" default:"true"`
		Listen                     string        `yaml:"listen" default:"0.0.0.0:9096" comment:"listen addr for prometheus ui and api"`
		ExternalURL                string        `yaml:"external_url" default:"http://127.0.0.1:9096" comment:"allows to set URL for redirect manually"`
		PageTitle                  string        `yaml:"page_title" default:"Pluto"`
		LookbackDelta              time.Duration `yaml:"lookback_delta" default:"5m"`
		RemoteReadConcurrencyLimit int           `yaml:"remote_read_concurrency_limit" default:"10" comment:"concurrently handled remote read requests"`
	} `yaml:"prometheus"`

	Debug struct {
		Enabled bool   `yaml:"enabled" default:"true"`
		Listen  string `yaml:"listen" default:"0.0.0.0:9095"`
		Pprof   bool   `yaml:"pprof" default:"true"`
		Metrics bool   `yaml:"metrics" default:"true"`
	} `yaml:"debug"`

	Logging zap.Config `yaml:"logging"`
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
