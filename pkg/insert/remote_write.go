package insert

import (
	"fmt"
	"io"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pluto-metrics/pluto/pkg/config"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/query"
	"github.com/pluto-metrics/pluto/pkg/scope"
	"go.uber.org/zap"

	_ "github.com/gogo/protobuf/gogoproto"
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

	insertCfg, err := rcv.opts.Config.InsertConfig(
		config.NewInsertEnv().WithRequest(r),
	)
	if err != nil {
		zap.L().Error("can't get insert config", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	qq := fmt.Sprintf("INSERT INTO %s FORMAT RowBinaryWithNamesAndTypes\n", insertCfg.Table)

	ctx := scope.QueryBegin(r.Context())
	scope.QueryWith(ctx, zap.String("query", qq))
	defer scope.QueryFinish(ctx)

	chRequest, err := query.NewRequest(ctx, insertCfg.ClickHouse, query.Opts{})
	if err != nil {
		zap.L().Error("can't create request to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer chRequest.Close()

	_, err = fmt.Fprint(chRequest, qq)
	if err != nil {
		zap.L().Error("can't write query to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if err := rawpbPromPbToRowBinary(reqRaw, chRequest, id.NewNameWithSha256()); err != nil {
		zap.L().Error("can't write request to clickhouse", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
