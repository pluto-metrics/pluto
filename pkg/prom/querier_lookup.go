package prom

import (
	"bufio"
	"context"

	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/model/labels"
	"go.uber.org/zap"
)

func (q *Querier) lookup(ctx context.Context, start, end int64, matchers []*labels.Matcher) (map[string]labels.Labels, error) {
	where := sql.NewWhere()
	q.whereSeriesTimeRange(ctx, where, start, end)
	q.whereMatchLabels(ctx, where, matchers)

	qq, err := sql.Template(`
		SELECT id, any(labels)
		FROM {{.table}}
		{{.where.SQL}}
		GROUP BY id
		FORMAT RowBinary
	`, map[string]interface{}{
		"table": q.config.Select.TableSeries,
		"where": where,
	})
	if err != nil {
		return nil, err
	}
	chRequest, err := q.request(ctx, qq)
	if err != nil {
		return nil, err
	}
	defer chRequest.Close()

	chResponse, err := chRequest.Finish()
	if err != nil {
		zap.L().Error("can't finish request to clickhouse", zap.Error(err))
		return nil, err
	}
	defer chResponse.Close()

	r := schema.NewReader(bufio.NewReader(chResponse)).
		Format(schema.RowBinary).
		Column(rowbinary.String). // id
		Column(ColumnLabels)      // labels

	ret := make(map[string]labels.Labels)

	for r.Next() {
		id, err := schema.Read(r, rowbinary.String)
		if err != nil {
			return nil, err
		}
		lb, err := schema.Read(r, ColumnLabels)
		if err != nil {
			return nil, err
		}
		ret[id] = lb
	}
	if r.Err() != nil {
		return nil, err
	}

	return ret, nil
}
