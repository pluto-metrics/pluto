package prom

import (
	"context"
	"log/slog"

	"github.com/jinzhu/copier"
	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/pluto/pkg/sql"
	"github.com/lomik/rowbinary"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

func (q *Querier) selectSeries(ctx context.Context, selectHints *storage.SelectHints, matchers []*labels.Matcher) (map[string]labels.Labels, error) {
	envSeries := config.EnvSeries{}
	if err := copier.Copy(&envSeries, selectHints); err != nil {
		return nil, err
	}

	seriesCfg, err := q.config.GetSeries(&envSeries)
	if err != nil {
		return nil, err
	}

	where := sql.NewWhere()
	q.whereSeriesTimeRange(ctx, where, selectHints.Start, selectHints.End)
	q.whereMatchLabels(ctx, seriesCfg, where, matchers)

	qq, err := sql.Template(`
		SELECT id, any(labels)
		FROM {{.table}}
		{{.where.SQL}}
		GROUP BY id
		FORMAT RowBinary
	`, map[string]interface{}{
		"table": seriesCfg.Table,
		"where": where,
	})
	if err != nil {
		return nil, err
	}

	chClient := rowbinary.NewClient(
		rowbinary.WithDiscovery(q.config.Extension.ClickHouseDiscovery),
		rowbinary.WithHTTPClient(q.config.Extension.HTTPClient),
		seriesCfg.ClickHouse.RowBinaryOptions()...,
	)
	defer chClient.Close()

	ret := make(map[string]labels.Labels)

	err = chClient.Select(ctx, qq,
		rowbinary.WithFormatReader(func(r *rowbinary.FormatReader) error {
			for r.Next() {
				var id string
				var lb labels.Labels
				rowbinary.Scan(r, rowbinary.String, &id)
				rowbinary.Scan(r, ColumnLabels, &lb)
				ret[id] = lb
			}
			return r.Err()
		}),
	)

	if err != nil {
		slog.ErrorContext(ctx, "can't finish request to clickhouse", lg.Error(err))
		return nil, err
	}

	return ret, nil
}
