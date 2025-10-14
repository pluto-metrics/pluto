package insert

import (
	"sync"

	"github.com/lomik/rowbinary"
)

var pbTimeseriesPool = sync.Pool{
	New: func() interface{} {
		return &pbTimeseries{
			Labels: rowbinary.NewKV(),
		}
	},
}

type pbSample struct {
	Value     float64
	Timestamp int64
}

type pbTimeseries struct {
	Labels         *rowbinary.KV[[]byte, []byte]
	Samples        []pbSample
	lastLabelName  []byte
	lastLabelValue []byte
}

func (p *pbTimeseries) begin() error {
	p.Labels.Reset()
	p.Samples = p.Samples[:0]
	return nil
}

func (p *pbTimeseries) labelBegin() error {
	p.lastLabelName = nil
	p.lastLabelValue = nil
	return nil
}

func (p *pbTimeseries) labelName(v []byte) error {
	p.lastLabelName = v
	return nil
}

func (p *pbTimeseries) labelValue(v []byte) error {
	p.lastLabelValue = v
	return nil
}
func (p *pbTimeseries) labelEnd() error {
	p.Labels.Append(p.lastLabelName, p.lastLabelValue)
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
