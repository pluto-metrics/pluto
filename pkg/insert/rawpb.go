package insert

import (
	"sync"
	"unsafe"

	"github.com/pluto-metrics/pluto/pkg/insert/id"
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/pluto-metrics/pluto/pkg/rawpb"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
)

var pbTimeseriesPool = sync.Pool{
	New: func() interface{} { return &pbTimeseries{} },
}

func unsafeBytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

type pbSample struct {
	Value     float64
	Timestamp int64
}

type pbTimeseries struct {
	Labels  []labels.Bytes
	Samples []pbSample
}

func (p *pbTimeseries) begin() error {
	p.Labels = p.Labels[:0]
	p.Samples = p.Samples[:0]
	return nil
}

func (p *pbTimeseries) labelBegin() error {
	p.Labels = append(p.Labels, labels.Bytes{})
	return nil
}

func (p *pbTimeseries) labelName(v []byte) error {
	p.Labels[len(p.Labels)-1].Name = v
	return nil
}

func (p *pbTimeseries) labelValue(v []byte) error {
	p.Labels[len(p.Labels)-1].Value = v
	return nil
}

func (p *pbTimeseries) sampleBegin() error {
	p.Samples = append(p.Samples, pbSample{})
	return nil
}
func (p *pbTimeseries) sampleValue(v float64) error {
	p.Samples[len(p.Samples)-1].Value = v
	return nil
}

func (p *pbTimeseries) sampleTimestamp(v int64) error {
	p.Samples[len(p.Samples)-1].Timestamp = v
	return nil
}

func payloadToRowBinary(raw []byte, w rowbinary.OriginWriter, h id.Provider) error {
	ws := schema.NewWriter(w).
		Format(schema.RowBinaryWithNamesAndTypes).
		Column("id", rowbinary.String).
		Column("name", rowbinary.String).
		Column("labels", labels.ColumnBytes).
		Column("timestamp", rowbinary.Int64).
		Column("value", rowbinary.Float64)

	if err := ws.WriteHeader(); err != nil {
		return err
	}

	ts := pbTimeseriesPool.Get().(*pbTimeseries)
	defer pbTimeseriesPool.Put(ts)

	parser := rawpb.New(
		rawpb.FieldNested(1, rawpb.New(
			rawpb.Begin(ts.begin),
			rawpb.FieldNested(1, rawpb.New(
				rawpb.Begin(ts.labelBegin),
				rawpb.FieldBytes(1, ts.labelName),
				rawpb.FieldBytes(2, ts.labelValue),
			)),
			rawpb.FieldNested(2, rawpb.New(
				rawpb.Begin(ts.sampleBegin),
				rawpb.FieldFloat64(1, ts.sampleValue),
				rawpb.FieldInt64(2, ts.sampleTimestamp),
			)),
			rawpb.End(func() error {
				if len(ts.Labels) == 0 || len(ts.Samples) == 0 {
					return nil
				}
				h.Update(ts.Labels)

				for j := 0; j < len(ts.Samples); j++ {
					if err := ws.WriteValues(
						unsafeBytesToString(h.ID()),
						unsafeBytesToString(h.Name()),
						ts.Labels,
						ts.Samples[j].Timestamp,
						ts.Samples[j].Value,
					); err != nil {
						return err
					}
				}

				return nil
			}),
		)),
	)

	if err := parser.Parse(raw); err != nil {
		return err
	}

	return nil
}
