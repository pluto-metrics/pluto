package prom

import (
	"context"

	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/prometheus/prometheus/model/labels"
)

func (q *Querier) whereMatchLabels(_ context.Context, cfg config.ConfigSeries, where *sql.Where, matchers []*labels.Matcher) {
	mp := map[string]bool{}
	for _, v := range cfg.SeriesMaterializedLabels {
		mp[v] = true
	}

	for _, m := range matchers {
		column := sql.ArrayElement("labels", sql.Quote(m.Name))
		if m.Name == "__name__" {
			column = sql.Column("name")
		}
		if mp[m.Name] {
			column = sql.Column("label_" + m.Name)
		}

		switch m.Type {
		case labels.MatchEqual:
			where.And(sql.Eq(column, sql.Quote(m.Value)))
		case labels.MatchNotEqual:
			where.And(sql.Ne(column, sql.Quote(m.Value)))
		case labels.MatchRegexp:
			where.And(sql.Match(column, sql.Quote("^"+m.Value+"$")))
		case labels.MatchNotRegexp:
			where.And(sql.Not(sql.Match(column, sql.Quote("^"+m.Value+"$"))))
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
