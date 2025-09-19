package config

import (
	"log/slog"

	"github.com/expr-lang/expr"
	"github.com/pluto-metrics/pluto/pkg/lg"
	"github.com/spf13/cast"
)

func (w *ConfigWhen) compileWhen(env any) error {
	if w == nil {
		return nil
	}
	if w.WhenStr == "" {
		w.WhenExpr = nil
		return nil
	}
	var err error
	w.WhenExpr, err = expr.Compile(w.WhenStr, expr.Env(env), expr.AsBool())
	return err
}

func (w *ConfigWhen) When(env any) (bool, error) {
	if w == nil {
		return false, nil
	}
	if w.WhenExpr == nil {
		return false, nil
	}
	output, err := expr.Run(w.WhenExpr, env)
	if err != nil {
		slog.Error("can't evaluate expression", slog.String("expr", w.WhenStr), lg.Error(err))
		return false, err
	}
	return cast.ToBool(output), nil
}

func (w *ConfigWhen) Pass() bool {
	if w == nil {
		return true
	}
	if w.WhenExpr == nil {
		return true
	}
	return false
}
