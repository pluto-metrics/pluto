package prom

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/jinzhu/copier"
	"github.com/lomik/pluto/pkg/cli"
	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/pluto/pkg/sql"
	"github.com/lomik/rowbinary"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

// override in unit tests for stable results
var timeNow = time.Now

// Select returns a set of series that matches the given label matchers.
func (q *Querier) Select(ctx context.Context, sortSeries bool, selectHints *storage.SelectHints, labelsMatcher ...*labels.Matcher) storage.SeriesSet {
	seriesMap, err := q.selectSeries(ctx, selectHints, labelsMatcher)
	if err != nil {
		slog.ErrorContext(ctx, "can't find series", lg.Error(err))
		return errorSeriesSet(err)
	}

	if len(seriesMap) == 0 {
		return emptySeriesSet()
	}

	if selectHints != nil && selectHints.Func == "series" {
		// /api/v1/series?match[]=...
		return newLabelsSeriesSet(slices.Collect(maps.Values(seriesMap)))
	}

	// select config with overrides
	envSamples := config.EnvSamples{}
	if err := copier.Copy(&envSamples, selectHints); err != nil {
		return errorSeriesSet(err)
	}

	samplesCfg, err := q.config.GetSamples(&envSamples)
	if err != nil {
		return errorSeriesSet(err)
	}

	var step int64 = 1000 // 1 second
	if selectHints.Step != 0 {
		step = selectHints.Step
	}

	// don't fetch full ids, use hash
	unhash := NewHashSelector(maps.Keys(seriesMap))

	timestampDiv := int64(1)
	if samplesCfg.SamplesTimestampUInt32 {
		timestampDiv = 1000
	}

	// fetch data by ids
	qq, err := sql.Template(`
		SELECT {{.id_hash}} as id_hash, min(timestamp), maxArgMin(value, timestamp)
		FROM {{.table}}
		WHERE id IN ids
			AND timestamp >= {{.start|quote}}-{{.step|quote}}
			AND timestamp <= {{.end|quote}}
		GROUP BY id_hash, intDiv(timestamp-{{.start|quote}}, {{.step|quote}})
	`, map[string]any{
		"id_hash": unhash.SelectColumn("id"),
		"table":   samplesCfg.Table,
		"start":   selectHints.Start / timestampDiv,
		"end":     selectHints.End / timestampDiv,
		"step":    step / timestampDiv,
	})
	if err != nil {
		slog.ErrorContext(ctx, "can't create request to clickhouse", lg.Error(err))
		return errorSeriesSet(err)
	}

	ch := cli.New(ctx, q.config, samplesCfg.ClickHouse)
	defer ch.Close()

	dataMap := make(map[string]*series, len(seriesMap))
	uniqDataMap := make(map[string]*series, len(seriesMap)) // key = labelsMapKey(labels)
	for k, v := range seriesMap {
		uniqKey := labelsMapKey(v)
		d := uniqDataMap[uniqKey]
		if d == nil {
			d = &series{
				labels:  v,
				samples: make([]sample, 0),
			}
		}
		uniqDataMap[uniqKey] = d
		dataMap[k] = d
	}

	err = ch.Select(ctx, qq,
		rowbinary.WithExternalData("ids",
			rowbinary.WithColumn("id", rowbinary.String),
			rowbinary.WithFormatWriter(func(w *rowbinary.FormatWriter) error {
				for k := range seriesMap {
					rowbinary.Write(w, rowbinary.String, k)
				}
				return w.Err()
			}),
		),
		rowbinary.WithFormatReader(func(r *rowbinary.FormatReader) error {
			var id string
			var timestamp int64
			var timestamp32 uint32
			var value float64

			for r.Next() {
				id, _ = unhash.SchemaRead(r)
				if samplesCfg.SamplesTimestampUInt32 {
					rowbinary.Scan(r, rowbinary.UInt32, &timestamp32)
					timestamp = int64(timestamp32) * 1000
				} else {
					rowbinary.Scan(r, rowbinary.Int64, &timestamp)
				}
				rowbinary.Scan(r, rowbinary.Float64, &value)
				if r.Err() != nil {
					slog.ErrorContext(ctx, "can't read row from clickhouse", lg.Error(r.Err()))
					return r.Err()
				}

				dataMap[id].sampleAppend(timestamp, value)
			}

			// @TODO: read
			return r.Err()
		}),
	)
	if err != nil {
		slog.ErrorContext(ctx, "can't finish request to clickhouse", lg.Error(err))
		return errorSeriesSet(err)
	}

	data := make([]series, 0, len(uniqDataMap))
	for _, v := range uniqDataMap {
		if len(v.samples) == 0 {
			continue
		}
		data = append(data, *v)
	}

	ss, err := makeSeriesSet(data, selectHints)
	if err != nil {
		slog.ErrorContext(ctx, "can't make series", lg.Error(err))
		return errorSeriesSet(err)
	}

	return ss
}
