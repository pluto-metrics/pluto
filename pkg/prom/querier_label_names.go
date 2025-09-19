package prom

import (
	"bufio"
	"context"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/util/annotations"
	"go.uber.org/zap"

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
		FORMAT RowBinary
	`, map[string]interface{}{
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
		zap.L().Error("can't finish request to clickhouse", zap.Error(err))
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
		return nil, nil, err
	}

	return rows, nil, nil
}
