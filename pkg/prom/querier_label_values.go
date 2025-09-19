package prom

import (
	"bufio"
	"context"
	"log/slog"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/lg"
	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
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

	chRequest, err := q.request(ctx, seriesCfg.ClickHouse, qq)
	if err != nil {
		return nil, nil, err
	}
	defer chRequest.Close()

	chResponse, err := chRequest.Finish()
	if err != nil {
		slog.ErrorContext(ctx, "can't finish request to clickhouse", lg.Error(err))
		return nil, nil, err
	}
	defer chResponse.Close()

	r := schema.NewReader(bufio.NewReader(chResponse)).
		Format(schema.RowBinary).
		Column(rowbinary.String)

	rows := []string{}
	for r.Next() {
		row, err := schema.Read(r, rowbinary.String)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, row)
	}

	if r.Err() != nil {
		return nil, nil, r.Err()
	}

	return rows, nil, nil
}
