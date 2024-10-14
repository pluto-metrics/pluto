package config

import (
	"github.com/expr-lang/expr"
	"github.com/spf13/cast"
)

type InsertEnv struct {
	GetParams map[string]string `expr:"GET"`
}

type InsertConfig struct {
	Table      string
	IDFunc     string
	ClickHouse ClickHouse `yaml:"clickhouse"`
}

func (cfg *Config) InsertConfig(values InsertEnv) (InsertConfig, error) {
	ret := InsertConfig{}

	for _, o := range cfg.Insert.TableOverride {
		if o.WhenExpr == nil {
			continue
		}

		result, err := expr.Run(o.WhenExpr, values)
		if err != nil {
			return ret, err
		}
		if cast.ToBool(result) {
			ret.Table = mergeAny(cfg.Insert.Table, o.Table)
			ret.IDFunc = mergeAny(cfg.Insert.IDFunc, o.IDFunc)
			ret.ClickHouse = *mergeClickHouse(&cfg.ClickHouse, cfg.Insert.ClickHouse, o.ClickHouse)
			return ret, nil
		}
	}

	ret.Table = mergeAny(cfg.Insert.Table)
	ret.IDFunc = mergeAny(cfg.Insert.IDFunc)
	ret.ClickHouse = *mergeClickHouse(&cfg.ClickHouse, cfg.Insert.ClickHouse)
	return ret, nil
}
