package prom

import (
	"bufio"
	"context"
	"sort"

	"github.com/jinzhu/copier"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/scope"
	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/zap"
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

	ctx = scope.QueryBegin(ctx)
	scope.QueryWith(ctx, zap.String("query", qq))
	defer scope.QueryFinish(ctx)

	chRequest, err := q.request(ctx, seriesCfg.ClickHouse, qq)
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
		sort.Slice(lb, func(i, j int) bool {
			return lb[i].Name < lb[j].Name
		})
		ret[id] = lb
	}
	if r.Err() != nil {
		return nil, err
	}

	return ret, nil
}
