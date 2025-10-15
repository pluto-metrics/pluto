package insert

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/golang/snappy"
	"github.com/lomik/pluto/pkg/cli"
	"github.com/lomik/pluto/pkg/config"
	"github.com/lomik/pluto/pkg/insert/id"
	"github.com/lomik/pluto/pkg/insert/labels"
	"github.com/lomik/pluto/pkg/lg"
	"github.com/lomik/rawpb"
	"github.com/lomik/rowbinary"
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

	ch := cli.New(r.Context(), rcv.opts.Config, insertCfg.ClickHouse)
	defer ch.Close()

	typeLabels := rowbinary.MapKV(rowbinary.StringBytes, rowbinary.StringBytes)

	err = ch.Insert(r.Context(),
		insertCfg.Table,
		rowbinary.RowBinaryWithNamesAndTypes,
		rowbinary.WithColumn("id", rowbinary.StringBytes),
		rowbinary.WithColumn("name", rowbinary.StringBytes),
		rowbinary.WithColumn("labels", typeLabels),
		rowbinary.WithColumn("timestamp", rowbinary.Int64),
		rowbinary.WithColumn("value", rowbinary.Float64),
		rowbinary.WithFormatWriter(func(rbw *rowbinary.FormatWriter) error {
			// parse protobuf and write to clickhouse
			ts := pbTimeseriesPool.Get().(*pbTimeseries)
			defer pbTimeseriesPool.Put(ts)

			h := id.NewNameWithSha256()

			parser := rawpb.New(
				rawpb.Message(1, rawpb.New(
					rawpb.Begin(ts.begin),
					rawpb.Message(1, rawpb.New(
						rawpb.Begin(ts.labelBegin),
						rawpb.Bytes(1, ts.labelName),
						rawpb.Bytes(2, ts.labelValue),
					)),
					rawpb.Message(2, rawpb.New(
						rawpb.Begin(ts.sampleBegin),
						rawpb.Double(1, ts.sampleValue),
						rawpb.Int64(2, ts.sampleTimestamp),
					)),
					rawpb.End(func() error {
						if len(ts.Labels) == 0 || len(ts.Samples) == 0 {
							return nil
						}
						h.Update(ts.Labels)

						for j := 0; j < len(ts.Samples); j++ {
							rowbinary.Write(rbw, rowbinary.StringBytes, h.ID())
							rowbinary.Write(rbw, rowbinary.StringBytes, h.Name())
							rowbinary.Write(rbw, labels.ColumnLabels, ts.Labels)
							rowbinary.Write(rbw, rowbinary.Int64, ts.Samples[j].Timestamp)
							rowbinary.Write(rbw, rowbinary.Float64, ts.Samples[j].Value)

							if rbw.Err() != nil {
								return rbw.Err()
							}
						}

						return nil
					}),
				)),
			)

			return parser.Parse(reqRaw)
		}),
	)

	if err != nil {
		slog.ErrorContext(r.Context(), "can't insert data to clickhouse", lg.Error(err))
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
