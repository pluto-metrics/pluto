package config

import (
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/spf13/cast"
)

type QueryLogEnv struct {
	Kind      string `expr:"kind"`
	Query     string `expr:"query"`
	ElapsedMs int64  `expr:"elapsed_ms"`
}

type ClickHouse struct {
	DSN          string            `yaml:"dsn" validate:"uri" default:"http://127.0.0.1:8123/?async_insert=1&wait_for_async_insert=1"`
	Params       map[string]string `yaml:"params"`
	QueryLog     string            `yaml:"query_log" default:"kind != 'insert'"`
	QueryLogExpr *vm.Program       `yaml:"-"`
}

func (ch *ClickHouse) compile() error {
	if ch == nil {
		return nil
	}
	if ch.QueryLog == "" {
		ch.QueryLogExpr = nil
		return nil
	}

	var err error
	ch.QueryLogExpr, err = expr.Compile(ch.QueryLog, expr.Env(QueryLogEnv{}), expr.AsBool())
	return err
}

func (ch *ClickHouse) GetDSN() string {
	if ch == nil {
		return ""
	}
	return ch.DSN
}

func (ch *ClickHouse) GetParams() map[string]string {
	if ch == nil {
		return nil
	}
	return ch.Params
}

func (ch *ClickHouse) GetQueryLogExpr() *vm.Program {
	if ch == nil {
		return nil
	}
	return ch.QueryLogExpr
}

func (ch *ClickHouse) GetQueryLog() string {
	if ch == nil {
		return ""
	}
	return ch.QueryLog
}

func (ch *ClickHouse) ShouldQueryLog(v QueryLogEnv) (bool, error) {
	if ch == nil {
		return false, nil
	}
	if ch.QueryLogExpr == nil {
		return false, nil
	}
	output, err := expr.Run(ch.QueryLogExpr, v)
	if err != nil {
		return false, err
	}
	return cast.ToBool(output), nil
}
