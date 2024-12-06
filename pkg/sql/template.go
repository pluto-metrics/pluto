package sql

import (
	"context"

	"strings"
	"text/template"

	"github.com/pluto-metrics/pluto/pkg/trace"
)

func Template(ctx context.Context, t string, args interface{}) (string, error) {
	// @TODO: cache template
	funcMap := template.FuncMap{
		"column": Column,
		"quote":  Quote,
	}
	tmpl, err := template.New(t).Funcs(funcMap).Parse(t)
	if err != nil {
		trace.Log(ctx).Error("can't parse template", trace.Error(err))
		return "", err
	}

	out := new(strings.Builder)

	err = tmpl.Execute(out, args)
	if err != nil {
		trace.Log(ctx).Error("can't execute template", trace.Error(err))
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
