package insert

import (
	"bytes"
	"io"
	"net/http"

	"github.com/golang/snappy"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/prompb"

	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
)

type PrometheusRemoteWrite struct {
	IDFunc string
}

func (rcv *PrometheusRemoteWrite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqCompressed, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reqRaw, err := snappy.Decode(nil, reqCompressed)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest

	if err := proto.Unmarshal(reqRaw, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chBuf := new(bytes.Buffer)

	rbw := schema.NewWriter(chBuf).
		Format(schema.RowBinaryWithNamesAndTypes).
		Column("id", rowbinary.String).
		Column("name", rowbinary.String).
		Column("labels", labels.ColumnPrompb).
		Column("timestamp", rowbinary.Int64).
		Column("value", rowbinary.Float64)

	if err := rbw.WriteHeader(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	series := req.GetTimeseries()

	for i := 0; i < len(series); i++ {
		s := series[i]

		name, id := id.NameWithSha256(s.Labels)

		// @TODO: if has many samples - precalc labels column once
		for j := 0; j < len(s.Samples); j++ {
			if err := rbw.WriteValues(
				id,
				name,
				s.Labels,
				s.Samples[j].Timestamp,
				s.Samples[j].Value,
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// err = rcv.unpackFast(r.Context(), reqBuf)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
}
