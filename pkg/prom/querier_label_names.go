package prom

import (
	"context"
	"log/slog"

	"github.com/lomik/pluto/pkg/cli"
	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/pluto/pkg/sql"
	"github.com/lomik/rowbinary"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/util/annotations"

	"github.com/prometheus/prometheus/model/labels"
)

// LabelNames returns all the unique label names present in the block in sorted order.
func (q *Querier) LabelNames(ctx context.Context, hints *storage.LabelHints, matchers ...*labels.Matcher) ([]string, annotations.Annotations, error) {
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
		SELECT arrayJoin(mapKeys(labels)) AS value
		FROM {{.table}}
		{{.where.SQL}}
		GROUP BY value
		ORDER BY value
	`, map[string]any{
		"table": seriesCfg.Table,
		"where": where,
	})
	if err != nil {
		return nil, nil, err
	}

	ch := cli.New(ctx, q.config, seriesCfg.ClickHouse)
	defer ch.Close()

	rows := []string{}

	err = ch.Select(ctx, qq,
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
