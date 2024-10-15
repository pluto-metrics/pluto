package config

import (
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
)

type Config struct {
	ClickHouse ClickHouse `yaml:"clickhouse"`

	Insert struct {
		Enabled          bool        `yaml:"enabled" default:"true"`
		Listen           string      `yaml:"listen" default:"0.0.0.0:9095" validate:"hostname_port"`
		CloseConnections bool        `yaml:"close-connections" default:"false"`
		IDFunc           string      `yaml:"id_func" default:"name_with_sha256" validate:"oneof=name_with_sha256"`
		Table            string      `yaml:"table" default:"samples_null"`
		ClickHouse       *ClickHouse `yaml:"clickhouse"`
		TableOverride    []struct {
			Table      string      `yaml:"table"`
			IDFunc     string      `yaml:"id_func" default:"name_with_sha256" validate:"oneof=name_with_sha256"`
			When       string      `yaml:"when" default:"false"`
			WhenExpr   *vm.Program `yaml:"-"`
			ClickHouse *ClickHouse `yaml:"clickhouse"`
		} `yaml:"table_override"`
	} `yaml:"insert"`

	Select struct {
		TableSeries          string        `yaml:"table_series"  default:"series"`
		TableSamples         string        `yaml:"table_samples" default:"samples"`
		AutocompleteLookback time.Duration `yaml:"autocomplete_lookback" default:"168h"`
		SeriesPartitionMs    int64         `yaml:"series_partition_ms" default:"86400000"`
		SamplesClickhouse    *ClickHouse   `yaml:"samples_clickhouse"`
		SeriesClickhouse     *ClickHouse   `yaml:"series_clickhouse"`
		TableSeriesOverride  []struct {
			Table                string        `yaml:"table"`
			AutocompleteLookback time.Duration `yaml:"autocomplete_lookback" default:"168h"`
			SeriesPartitionMs    int64         `yaml:"series_partition_ms" default:"86400000"`
			When                 string        `yaml:"when" default:"false"`
			WhenExpr             *vm.Program   `yaml:"-"`
			ClickHouse           *ClickHouse   `yaml:"clickhouse"`
		} `yaml:"table_series_override"`
		TableSamplesOverride []struct {
			Table      string      `yaml:"table"`
			When       string      `yaml:"when" default:"false"`
			WhenExpr   *vm.Program `yaml:"-"`
			ClickHouse *ClickHouse `yaml:"clickhouse"`
		} `yaml:"table_override"`
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

	if err = cfg.ClickHouse.compile(); err != nil {
		return nil, err
	}
	if err = cfg.Insert.ClickHouse.compile(); err != nil {
		return nil, err
	}
	if err = cfg.Select.SeriesClickhouse.compile(); err != nil {
		return nil, err
	}
	if err = cfg.Select.SamplesClickhouse.compile(); err != nil {
		return nil, err
	}

	for i := 0; i < len(cfg.Insert.TableOverride); i++ {
		if cfg.Insert.TableOverride[i].WhenExpr, err = expr.Compile(cfg.Insert.TableOverride[i].When, expr.Env(InsertEnv{}), expr.AsBool()); err != nil {
			return nil, err
		}
		if err = cfg.Insert.TableOverride[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(cfg.Select.TableSeriesOverride); i++ {
		if cfg.Select.TableSeriesOverride[i].WhenExpr, err = expr.Compile(cfg.Select.TableSeriesOverride[i].When); err != nil {
			return nil, err
		}
		if err = cfg.Select.TableSeriesOverride[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
	}

	for i := 0; i < len(cfg.Select.TableSamplesOverride); i++ {
		if cfg.Select.TableSamplesOverride[i].WhenExpr, err = expr.Compile(cfg.Select.TableSamplesOverride[i].When); err != nil {
			return nil, err
		}
		if err = cfg.Select.TableSamplesOverride[i].ClickHouse.compile(); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
