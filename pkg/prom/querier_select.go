package prom

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"mime/multipart"
	"slices"
	"time"

	"github.com/jinzhu/copier"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/query"
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
	seriesMap, err := q.selectSeries(ctx, selectHints, labelsMatcher)
	if err != nil {
		zap.L().Error("can't find series", zap.Error(err))
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
		FORMAT RowBinary
	`, map[string]interface{}{
		"id_hash": unhash.SelectColumn("id"),
		"table":   samplesCfg.Table,
		"start":   selectHints.Start / timestampDiv,
		"end":     selectHints.End / timestampDiv,
		"step":    step / timestampDiv,
	})
	if err != nil {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		return errorSeriesSet(err)
	}

	reqBuf := new(bytes.Buffer)
	reqWriter := multipart.NewWriter(reqBuf)

	createErr := func(err error) storage.SeriesSet {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		return errorSeriesSet(err)
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

	idsWriterBuf := bufio.NewWriter(idsWriter)

	schemaWriter := schema.NewWriter(idsWriterBuf).
		Format(schema.RowBinary).
		Column("id", rowbinary.String)

	for k := range seriesMap {
		if err = schemaWriter.WriteValues(k); err != nil {
			return createErr(err)
		}
	}

	if err = idsWriterBuf.Flush(); err != nil {
		return createErr(err)
	}

	if err = reqWriter.Close(); err != nil {
		return createErr(err)
	}

	chRequest, err := query.NewRequest(ctx, *samplesCfg.ClickHouse, query.Opts{
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
		return errorSeriesSet(err)
	}
	defer chRequest.Close()

	chResponse, err := chRequest.Finish()
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.L().Error("can't finish request to clickhouse", zap.Error(err))
		}
		return errorSeriesSet(err)
	}
	defer chResponse.Close()

	// fetch results
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

	r := schema.NewReader(bufio.NewReader(chResponse)).
		Format(schema.RowBinary).
		Column(unhash.ColumnType()) // id

	if samplesCfg.SamplesTimestampUInt32 {
		r = r.Column(rowbinary.UInt32) // timestamp uint32
	} else {
		r = r.Column(rowbinary.Int64) // timestamp int64 with ms
	}
	r = r.Column(rowbinary.Float64) // value

	var id string
	var timestamp int64
	var timestamp32 uint32
	var value float64

	for r.Next() {
		id, _ = unhash.SchemaRead(r)
		if samplesCfg.SamplesTimestampUInt32 {
			timestamp32, _ = schema.Read(r, rowbinary.UInt32)
			timestamp = int64(timestamp32) * 1000
		} else {
			timestamp, _ = schema.Read(r, rowbinary.Int64)
		}
		value, _ = schema.Read(r, rowbinary.Float64)
		if r.Err() != nil {
			zap.L().Error("can't read row from clickhouse", zap.Error(r.Err()))
			return errorSeriesSet(r.Err())
		}

		dataMap[id].sampleAppend(timestamp, value)
	}

	if r.Err() != nil {
		zap.L().Error("can't read response from clickhouse", zap.Error(r.Err()))
		return errorSeriesSet(r.Err())
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
		zap.L().Error("can't make series", zap.Error(err))
		return errorSeriesSet(err)
	}

	return ss
}
