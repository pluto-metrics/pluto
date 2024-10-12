package prom

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"maps"
	"mime/multipart"
	"slices"
	"time"

	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/pluto/pkg/scope"
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

	// don't fetch full ids, use hash
	unhash := NewHashSelector(maps.Keys(seriesMap))

	// fetch data by ids
	qq, err := sql.Template(`
		SELECT {{.id_hash}} as id_hash, min(timestamp), argMin(value, timestamp)
		FROM {{.table}}
		WHERE id IN ids
			AND timestamp >= {{.start|quote}}-{{.step|quote}}-{{.lookbackDelta}}
			AND timestamp <= {{.end|quote}}+{{.lookbackDelta}}
		GROUP BY id_hash, intDiv(timestamp-{{.start|quote}}, {{.step|quote}})
		FORMAT RowBinary
	`, map[string]interface{}{
		"id_hash":       unhash.SelectColumn("id"),
		"table":         q.config.Select.TableSamples,
		"start":         selectHints.Start,
		"end":           selectHints.End,
		"step":          step,
		"lookbackDelta": q.config.Prometheus.LookbackDelta.Milliseconds(),
	})

	ctx = scope.QueryBegin(ctx)
	scope.QueryWith(ctx, zap.String("query", qq))

	reqBuf := new(bytes.Buffer)
	reqWriter := multipart.NewWriter(reqBuf)

	createErr := func(err error) storage.SeriesSet {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		return nil
	}

	if err = reqWriter.WriteField("query", qq); err != nil {
		return createErr(err)
	}

	if err = reqWriter.WriteField("ids_format", "RowBinary"); err != nil {
		return createErr(err)
	}

	if err = reqWriter.WriteField("ids_structure", "id String"); err != nil {
		return createErr(err)
	}

	idsWriter, err := reqWriter.CreateFormFile("ids", "ids.bin")
	if err != nil {
		return createErr(err)
	}

	scope.QueryWith(ctx, zap.Int("ids", len(seriesMap)))

	for k := range seriesMap {
		if err = rowbinary.String.Write(idsWriter, k); err != nil {
			return createErr(err)
		}
	}

	if err = reqWriter.Close(); err != nil {
		return createErr(err)
	}

	chRequest, err := query.NewRequest(ctx, q.config.ClickHouse, query.Opts{
		Headers: map[string]string{
			"Content-Type": reqWriter.FormDataContentType(),
		},
	})
	if err != nil {
		return createErr(err)
	}

	_, err = io.Copy(chRequest, reqBuf)
	if err != nil {
		zap.L().Error("can't write query to clickhouse", zap.Error(err))
		return nil
	}
	defer chRequest.Close()

	chResponse, err := chRequest.Finish()
	if err != nil {
		zap.L().Error("can't finish request to clickhouse", zap.Error(err))
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
		Column(unhash.ColumnType()). // id
		Column(rowbinary.Int64).     // timestamp
		Column(rowbinary.Float64)    // value

	for r.Next() {
		id, _ := unhash.SchemaRead(r)
		timestamp, _ := schema.Read(r, rowbinary.Int64)
		value, _ := schema.Read(r, rowbinary.Float64)
		if r.Err() != nil {
			zap.L().Error("can't read row from clickhouse", zap.Error(r.Err()))
			return nil
		}

		dataMap[id].sampleAppend(timestamp, value)
	}

	if r.Err() != nil {
		zap.L().Error("can't read response from clickhouse", zap.Error(r.Err()))
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
		zap.L().Error("can't make series", zap.Error(err))
		return nil
	}

	return ss
}
