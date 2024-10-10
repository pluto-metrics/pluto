package prom

import (
	"context"

	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/prometheus/prometheus/model/labels"
)

func (q *Querier) whereMatchLabels(_ context.Context, where *sql.Where, matchers []*labels.Matcher) {
	for _, m := range matchers {
		switch m.Type {
		case labels.MatchEqual:
			where.And(sql.Eq(sql.ArrayElement("labels", sql.Quote(m.Name)), sql.Quote(m.Value)))
		case labels.MatchNotEqual:
			where.And(sql.Ne(sql.ArrayElement("labels", sql.Quote(m.Name)), sql.Quote(m.Value)))
		case labels.MatchRegexp:
			// @TODO
		case labels.MatchNotRegexp:
			// @TODO
		}
	}
}

func (q *Querier) whereSeriesTimeRange(_ context.Context, where *sql.Where, start int64, end int64) {
	where.And(
		sql.Gte(sql.Column("timestamp_min"), sql.Quote(start-q.config.Select.SeriesPartitionMs)),
	)
	where.And(
		sql.Lte(sql.Column("timestamp_min"), sql.Quote(end)),
	)
	where.And(
		sql.Gte(sql.Column("timestamp_max"), sql.Quote(start)),
	)
}
