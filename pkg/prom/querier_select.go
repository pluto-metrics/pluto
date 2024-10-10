package prom

import (
	"bufio"
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"go.uber.org/zap"
)

// override in unit tests for stable results
var timeNow = time.Now

// Select returns a set of series that matches the given label matchers.
func (q *Querier) Select(ctx context.Context, sortSeries bool, selectHints *storage.SelectHints, labelsMatcher ...*labels.Matcher) storage.SeriesSet {
	seriesMap, err := q.lookup(ctx, selectHints.Start, selectHints.End, labelsMatcher)

	if len(seriesMap) == 0 {
		return emptySeriesSet()
	}

	if selectHints != nil && selectHints.Func == "series" {
		// /api/v1/series?match[]=...
		return newLabelsSeriesSet(slices.Collect(maps.Values(seriesMap)))
	}

	var step int64 = 1000 // 1 second
	if selectHints.Step != 0 {
		step = selectHints.Step
	}

	ids := new(strings.Builder)
	for k := range seriesMap {
		if ids.Len() > 0 {
			ids.WriteByte(',')
		}
		ids.WriteString(sql.Quote(k))
	}
	// fetch data by ids
	// @TODO use external table
	// @TODO use hashed id
	qq, err := sql.Template(`
		SELECT id, min(timestamp), argMin(value, timestamp)
		FROM {{.table}}
		WHERE id IN ({{.ids}})
			AND timestamp >= {{.start|quote}}-{{.step|quote}}-{{.lookbackDelta}}
			AND timestamp <= {{.end|quote}}+{{.lookbackDelta}}
		GROUP BY id, intDiv(timestamp-{{.start|quote}}, {{.step|quote}})
		FORMAT RowBinary
	`, map[string]interface{}{
		"table":         q.config.Select.TableSamples,
		"start":         selectHints.Start,
		"end":           selectHints.End,
		"ids":           ids.String(),
		"step":          step,
		"lookbackDelta": q.config.Prometheus.LookbackDelta.Milliseconds(),
	})

	chRequest, err := q.request(ctx, qq)
	if err != nil {
		// @TODO: log error
		return nil
	}
	defer chRequest.Close()

	chResponse, err := chRequest.Finish()
	if err != nil {
		zap.L().Error("can't finish request to clickhouse", zap.Error(err))
		// @TODO: log error
		return nil
	}
	defer chResponse.Close()

	// fetch results
	dataMap := make(map[string]*series, len(seriesMap))
	for k, v := range seriesMap {
		dataMap[k] = &series{
			labels:  v,
			samples: make([]sample, 0),
		}
	}

	r := schema.NewReader(bufio.NewReader(chResponse)).
		Format(schema.RowBinary).
		Column(rowbinary.String). // id
		Column(rowbinary.Int64).  // timestamp
		Column(rowbinary.Float64) // value

	for r.Next() {
		id, _ := schema.Read(r, rowbinary.String)
		timestamp, _ := schema.Read(r, rowbinary.Int64)
		value, _ := schema.Read(r, rowbinary.Float64)
		if r.Err() != nil {
			// @TODO
			return nil
		}

		dataMap[id].sampleAppend(timestamp, value)
	}

	if r.Err() != nil {
		// @TODO
		return nil
	}

	data := make([]series, 0, len(dataMap))
	for _, v := range dataMap {
		if len(v.samples) == 0 {
			continue
		}
		data = append(data, *v)
	}

	ss, err := makeSeriesSet(data, hints{step: step})
	if err != nil {
		return nil // , nil, err @TODO
	}

	return ss //, nil, nil
}
