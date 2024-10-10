package insert

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
)

type Opts struct {
	Clickhouse      config.ClickHouse
	ClickhouseTable string
	IDFunc          string
}

type PrometheusRemoteWrite struct {
	opts Opts
}

func NewPrometheusRemoteWrite(opts Opts) *PrometheusRemoteWrite {
	return &PrometheusRemoteWrite{opts: opts}
}

func (rcv *PrometheusRemoteWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqCompressed, err := io.ReadAll(r.Body)
	if err != nil {
		zap.L().Error("can't read prometheus request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqRaw, err := snappy.Decode(nil, reqCompressed)
	if err != nil {
		zap.L().Error("can't decode prometheus request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest

	if err := proto.Unmarshal(reqRaw, &req); err != nil {
		zap.L().Error("can't unmarshal prometheus request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chRequest, err := query.NewRequest(r.Context(), rcv.opts.Clickhouse.DSN, rcv.opts.Clickhouse.Params, query.Opts{})
	if err != nil {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer chRequest.Close()

	_, err = fmt.Fprintf(chRequest, "INSERT INTO %s FORMAT RowBinaryWithNamesAndTypes\n", rcv.opts.ClickhouseTable)
	if err != nil {
		zap.L().Error("can't write query to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	chRequestWriter := schema.NewWriter(chRequest).
		Format(schema.RowBinaryWithNamesAndTypes).
		Column("id", rowbinary.String).
		Column("name", rowbinary.String).
		Column("labels", labels.ColumnPrompb).
		Column("timestamp", rowbinary.Int64).
		Column("value", rowbinary.Float64)

	if err := chRequestWriter.WriteHeader(); err != nil {
		zap.L().Error("can't write rowbinary header to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	series := req.GetTimeseries()

	for i := 0; i < len(series); i++ {
		s := series[i]

		name, id := id.NameWithSha256(s.Labels)

		// @TODO: if has many samples - precalc labels column once
		for j := 0; j < len(s.Samples); j++ {
			if err := chRequestWriter.WriteValues(
				id,
				name,
				s.Labels,
				s.Samples[j].Timestamp,
				s.Samples[j].Value,
			); err != nil {
				zap.L().Error("can't write sample to clickhouse", zap.Error(err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	chResponse, err := chRequest.Finish()
	if err != nil {
		zap.L().Error("can't finish request to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	err = chResponse.Close()
	if err != nil {
		zap.L().Error("can't close response from clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
}
