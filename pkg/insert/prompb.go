package insert

import (
	proto "github.com/gogo/protobuf/proto"
	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
	"github.com/prometheus/prometheus/prompb"
)

func naivePromPbToRowBinary(raw []byte, w rowbinary.Writer, h id.Provider) error {
	ws := schema.NewWriter(w).
		Format(schema.RowBinaryWithNamesAndTypes).
		Column("id", rowbinary.String).
		Column("name", rowbinary.String).
		Column("labels", labels.ColumnPrompb).
		Column("timestamp", rowbinary.Int64).
		Column("value", rowbinary.Float64)

	if err := ws.WriteHeader(); err != nil {
		return err
	}

	var req prompb.WriteRequest

	if err := proto.Unmarshal(raw, &req); err != nil {
		return err
	}

	series := req.GetTimeseries()

	for i := 0; i < len(series); i++ {
		s := series[i]

		h.Update(s.Labels)

		// @TODO: if has many samples - precalc labels column once
		for j := 0; j < len(s.Samples); j++ {
			if err := ws.WriteValues(
				h.ID(),
				h.Name(),
				s.Labels,
				s.Samples[j].Timestamp,
				s.Samples[j].Value,
			); err != nil {
				return err
			}
		}
	}

	return nil
}
