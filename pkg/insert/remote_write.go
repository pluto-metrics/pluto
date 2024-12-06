package insert

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/pluto/pkg/trace"
	"go.opentelemetry.io/otel/attribute"
)

type Opts struct {
	Config *config.Config
}

type PrometheusRemoteWrite struct {
	opts Opts
}

func NewPrometheusRemoteWrite(opts Opts) *PrometheusRemoteWrite {
	return &PrometheusRemoteWrite{opts: opts}
}

func (rcv *PrometheusRemoteWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := trace.Start(r.Context(), "PrometheusRemoteWrite.ServeHTTP")
	defer span.End()

	metricRemoteWriteRequests.Add(r.Context(), 1)

	if rcv.opts.Config.Insert.CloseConnections {
		w.Header().Add("Connection", "close")
	}

	reqCompressed, err := io.ReadAll(r.Body)
	if err != nil {
		trace.Log(ctx).Error("can't read prometheus request", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqRaw, err := snappy.Decode(nil, reqCompressed)
	if err != nil {
		trace.Log(ctx).Error("can't decode prometheus request", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	insertCfg, err := rcv.opts.Config.GetInsert(
		ctx,
		config.NewEnvInsert().WithRequest(r),
	)
	if err != nil {
		trace.Log(ctx).Error("can't get insert config", trace.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	qq := fmt.Sprintf("INSERT INTO %s FORMAT RowBinaryWithNamesAndTypes\n", insertCfg.Table)

	span.SetAttributes(attribute.String("query", qq))

	chRequest, err := query.NewRequest(ctx, *insertCfg.ClickHouse, query.Opts{})
	if err != nil {
		trace.Log(ctx).Error("can't create request to clickhouse", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer chRequest.Close()

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		trace.Log(ctx).Error("can't write query to clickhouse", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := payloadToRowBinary(ctx, reqRaw, chRequest, id.NewNameWithSha256()); err != nil {
		trace.Log(ctx).Error("can't write request to clickhouse", trace.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chResponse, err := chRequest.Finish()
	if err != nil {
		trace.Log(ctx).Error("can't finish request to clickhouse", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	err = chResponse.Close()
	if err != nil {
		trace.Log(ctx).Error("can't close response from clickhouse", trace.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if rcv.opts.Config.Insert.CloseConnections {
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			return
		}

		conn.Close()
	}
}
