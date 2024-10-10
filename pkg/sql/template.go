package sql

import (
	"strings"
	"text/template"

	"go.uber.org/zap"
)

func Template(t string, args interface{}) (string, error) {
	// @TODO: cache template
	funcMap := template.FuncMap{
		"column": Column,
		"quote":  Quote,
	}
	tmpl, err := template.New(t).Funcs(funcMap).Parse(t)
	if err != nil {
		zap.L().Error("can't parse template", zap.Error(err))
		return "", err
	}

	out := new(strings.Builder)

	err = tmpl.Execute(out, args)
	if err != nil {
		zap.L().Error("can't execute template", zap.Error(err))
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
