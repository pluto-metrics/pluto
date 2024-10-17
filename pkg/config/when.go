package config

import (
	"github.com/expr-lang/expr"
	"github.com/spf13/cast"
	"go.uber.org/zap"
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
		zap.L().Error("can't evaluate expression", zap.String("expr", w.WhenStr), zap.Error(err))
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
