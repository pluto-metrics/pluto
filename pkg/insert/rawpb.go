package insert

import (
	"sync"

	"github.com/lomik/pluto/pkg/insert/labels"
)

var pbTimeseriesPool = sync.Pool{
	New: func() any {
		return &pbTimeseries{
			Labels: make([]labels.Label, 0),
		}
	},
}

type pbSample struct {
	Value     float64
	Timestamp int64
}

type pbTimeseries struct {
	Labels         []labels.Label
	Samples        []pbSample
	lastLabelName  []byte
	lastLabelValue []byte
}

func (p *pbTimeseries) begin() error {
	p.Labels = p.Labels[:0]
	p.Samples = p.Samples[:0]
	return nil
}

func (p *pbTimeseries) labelBegin() error {
	p.Labels = append(p.Labels, labels.Label{})
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
