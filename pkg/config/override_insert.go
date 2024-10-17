package config

import (
	"net/http"

	"github.com/expr-lang/expr"
	"github.com/spf13/cast"
)

type EnvInsert struct {
	GetParams map[string]string `expr:"GET"`
	Headers   map[string]string `expr:"GET"`
}

func NewEnvInsert() *EnvInsert {
	return &EnvInsert{
		GetParams: map[string]string{},
		Headers:   map[string]string{},
	}
}

func (cfg *Config) GetInsert(values *EnvInsert) (ConfigInsert, error) {
	ret := ConfigInsert{
		Table:      cfg.Insert.Table,
		IDFunc:     cfg.Insert.IDFunc,
		ClickHouse: &cfg.ClickHouse,
	}

	for _, o := range cfg.OverrideInsert {
		if o.WhenExpr == nil {
			continue
		}

		result, err := expr.Run(o.WhenExpr, values)
		if err != nil {
			return ret, err
		}

		if cast.ToBool(result) {
			ret.Table = mergeAny(ret.Table, o.Table)
			ret.IDFunc = mergeAny(ret.IDFunc, o.IDFunc)
			ret.ClickHouse = mergeClickHouse(ret.ClickHouse, o.ClickHouse)
			return ret, nil
		}
	}

	return ret, nil
}

func (env *EnvInsert) WithRequest(r *http.Request) *EnvInsert {
	for k := range r.URL.Query() {
		env.GetParams[k] = r.URL.Query().Get(k)
	}

	for k := range r.Header {
		env.Headers[k] = r.Header.Get(k)
	}

	return env
}
