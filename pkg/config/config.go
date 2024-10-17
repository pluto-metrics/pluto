package config

import (
	"time"

	"github.com/expr-lang/expr/vm"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
)

type ConfigWhen struct {
	WhenStr  string      `yaml:"when" default:""`
	WhenExpr *vm.Program `yaml:"-"`
}

type ClickHouse struct {
	DSN      string            `yaml:"dsn" validate:"uri" default:"http://127.0.0.1:8123/?async_insert=1&wait_for_async_insert=1"`
	Params   map[string]string `yaml:"params"`
	QueryLog struct {
		ConfigWhen `yaml:",inline"`
	} `yaml:"query_log"`
}

type ConfigInsert struct {
	Table      string      `yaml:"table"`
	IDFunc     string      `yaml:"id_func" default:"" validate:"oneof='' 'name_with_sha256'"`
	ClickHouse *ClickHouse `yaml:"clickhouse"`
}

type ConfigSeries struct {
	Table                    string        `yaml:"table"`
	AutocompleteLookback     time.Duration `yaml:"autocomplete_lookback"`
	SeriesPartitionMs        int64         `yaml:"series_partition_ms"`
	SeriesMaterializedLabels []string      `yaml:"series_materialized_labels"`
	ClickHouse               *ClickHouse   `yaml:"clickhouse"`
}

type ConfigSamples struct {
	Table      string      `yaml:"table"`
	ClickHouse *ClickHouse `yaml:"clickhouse"`
}

type Config struct {
	ClickHouse ClickHouse `yaml:"clickhouse"`

	Insert struct {
		Enabled          bool   `yaml:"enabled" default:"true"`
		Listen           string `yaml:"listen" default:"0.0.0.0:9095" validate:"hostname_port"`
		CloseConnections bool   `yaml:"close-connections" default:"false"`
		Table            string `yaml:"table" default:"samples_null"`
		IDFunc           string `yaml:"id_func" default:"name_with_sha256" validate:"oneof=name_with_sha256"`
	} `yaml:"insert"`

	Select struct {
		TableSeries          string        `yaml:"table_series"  default:"series"`
		TableSamples         string        `yaml:"table_samples" default:"samples"`
		AutocompleteLookback time.Duration `yaml:"autocomplete_lookback" default:"168h"`
		SeriesPartitionMs    int64         `yaml:"series_partition_ms" default:"86400000"`
		// https://clickhouse.com/docs/knowledgebase/improve-map-performance
		// column names should be label_<label_name>
		SeriesMaterializedLabels []string `yaml:"series_materialize_labels"`
	} `yaml:"select"`

	Prometheus struct {
		Enabled                    bool          `yaml:"enabled" default:"true"`
		Listen                     string        `yaml:"listen" default:"0.0.0.0:9096" comment:"listen addr for prometheus ui and api"`
		ExternalURL                string        `yaml:"external_url" default:"http://127.0.0.1:9096" comment:"allows to set URL for redirect manually"`
		PageTitle                  string        `yaml:"page_title" default:"Pluto"`
		LookbackDelta              time.Duration `yaml:"lookback_delta" default:"1h"`
		RemoteReadConcurrencyLimit int           `yaml:"remote_read_concurrency_limit" default:"10" comment:"concurrently handled remote read requests"`
	} `yaml:"prometheus"`

	Debug struct {
		Enabled bool   `yaml:"enabled" default:"true"`
		Listen  string `yaml:"listen" default:"0.0.0.0:9095"`
		Pprof   bool   `yaml:"pprof" default:"true"`
		Metrics bool   `yaml:"metrics" default:"true"`
	} `yaml:"debug"`

	Logging zap.Config `yaml:"logging"`

	OverrideInsert []struct {
		ConfigInsert `yaml:",inline"`
		ConfigWhen   `yaml:",inline"`
	} `yaml:"override_insert"`

	OverrideSeries []struct {
		ConfigSeries `yaml:",inline"`
		ConfigWhen   `yaml:",inline"`
	} `yaml:"override_series"`

	OverrideSamples []struct {
		ConfigSamples `yaml:",inline"`
		ConfigWhen    `yaml:",inline"`
	} `yaml:"override_samples"`
}

func LoadFromFile(filename string, development bool) (*Config, error) {
	cfg := Config{}

	var err error

	if development {
		cfg.Logging = zap.NewDevelopmentConfig()
	} else {
		cfg.Logging = zap.NewProductionConfig()
	}
	if err = configor.Load(&cfg, filename); err != nil {
		return nil, err
	}
	if err = validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return nil, err
	}

	for i := 0; i < len(cfg.OverrideInsert); i++ {

	}

	if err = cfg.ClickHouse.compile(); err != nil {
		return nil, err
	}

	for i := 0; i < len(cfg.OverrideInsert); i++ {
		if err = cfg.OverrideInsert[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
		if err = cfg.OverrideInsert[i].compileWhen(EnvInsert{}); err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(cfg.OverrideSeries); i++ {
		if err = cfg.OverrideSeries[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
		if err = cfg.OverrideSeries[i].compileWhen(EnvSeries{}); err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(cfg.OverrideSamples); i++ {
		if err = cfg.OverrideSamples[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
		if err = cfg.OverrideSamples[i].compileWhen(EnvSamples{}); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
