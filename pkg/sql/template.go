package sql

import (
	"log/slog"
	"strings"
	"text/template"

	"github.com/pluto-metrics/pluto/pkg/lg"
)

func Template(t string, args interface{}) (string, error) {
	// @TODO: cache template
	funcMap := template.FuncMap{
		"column": Column,
		"quote":  Quote,
	}
	tmpl, err := template.New(t).Funcs(funcMap).Parse(t)
	if err != nil {
		slog.Error("can't parse template", lg.Error(err))
		return "", err
	}

	out := new(strings.Builder)

	err = tmpl.Execute(out, args)
	if err != nil {
		slog.Error("can't execute template", lg.Error(err))
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
