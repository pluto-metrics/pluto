package prom

import (
	"context"
	"log/slog"

	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/pluto/pkg/sql"
	"github.com/lomik/rowbinary"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/util/annotations"

	"github.com/prometheus/prometheus/model/labels"
)

// LabelValues returns all potential values for a label name.
func (q *Querier) LabelValues(ctx context.Context, label string, hints *storage.LabelHints, matchers ...*labels.Matcher) ([]string, annotations.Annotations, error) {
	seriesCfg, err := q.config.GetSeries(&config.EnvSeries{Limit: hints.Limit})
	if err != nil {
		return nil, nil, err
	}

	now := timeNow()
	start := now.Add(-seriesCfg.AutocompleteLookback).UnixMilli()
	end := now.UnixMilli()

	where := sql.NewWhere()
	q.whereSeriesTimeRange(ctx, where, start, end)
	q.whereMatchLabels(ctx, seriesCfg, where, matchers)

	qq, err := sql.Template(`
		SELECT
			{{if (eq .label "__name__")}}
				name
			{{else}}
				arrayElement(labels, {{.label|quote}})
			{{end}}
			AS value
		FROM {{.table}}
		{{.where.SQL}}
		GROUP BY value
		ORDER BY value
		FORMAT RowBinary
	`, map[string]interface{}{
		"label": label,
		"table": seriesCfg.Table,
		"where": where,
	})
	if err != nil {
		return nil, nil, err
	}

	chClient := rowbinary.NewClient(
		rowbinary.WithDiscovery(q.config.Extension.ClickHouseDiscovery),
		rowbinary.WithHTTPClient(q.config.Extension.HTTPClient),
		seriesCfg.ClickHouse.RowBinaryOptions()...,
	)
	defer chClient.Close()

	rows := []string{}

	err = chClient.Select(ctx, qq,
		rowbinary.WithFormatReader(func(r *rowbinary.FormatReader) error {
			var s string
			for r.Next() {
				rowbinary.Scan(r, rowbinary.String, &s)
				rows = append(rows, s)
			}
			return r.Err()
		}),
	)
	if err != nil {
		slog.ErrorContext(ctx, "can't finish request to clickhouse", lg.Error(err))
		return nil, nil, err
	}

	return rows, nil, nil
}
