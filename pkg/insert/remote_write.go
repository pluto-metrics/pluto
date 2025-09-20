package insert

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/lg"
	"github.com/pluto-metrics/pluto/pkg/query"
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
	if rcv.opts.Config.Insert.CloseConnections {
		w.Header().Add("Connection", "close")
	}

	reqCompressed, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(r.Context(), "can't read prometheus request", lg.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqRaw, err := snappy.Decode(nil, reqCompressed)
	if err != nil {
		slog.ErrorContext(r.Context(), "can't decode prometheus request", lg.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	insertCfg, err := rcv.opts.Config.GetInsert(
		config.NewEnvInsert().WithRequest(r),
	)
	if err != nil {
		slog.ErrorContext(r.Context(), "can't get insert config", lg.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	qq := fmt.Sprintf("INSERT INTO %s FORMAT RowBinaryWithNamesAndTypes\n", insertCfg.Table)

	chRequest, err := query.NewRequest(r.Context(), *insertCfg.ClickHouse, query.Opts{})
	if err != nil {
		slog.ErrorContext(r.Context(), "can't create request to clickhouse", lg.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer chRequest.Close()

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		slog.ErrorContext(r.Context(), "can't write query to clickhouse", lg.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := payloadToRowBinary(reqRaw, chRequest, id.NewNameWithSha256()); err != nil {
		slog.ErrorContext(r.Context(), "can't write request to clickhouse", lg.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	chResponse, err := chRequest.Finish()
	if err != nil {
		slog.ErrorContext(r.Context(), "can't finish request to clickhouse", lg.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	err = chResponse.Close()
	if err != nil {
		slog.ErrorContext(r.Context(), "can't close response from clickhouse", lg.Error(err))
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
