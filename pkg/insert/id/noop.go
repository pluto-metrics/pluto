package id

import (
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/prometheus/prometheus/prompb"
)

type Noop struct {
}

func NewNoop() *Noop {
	return &Noop{}
}

func (h *Noop) Update(labels []prompb.Label) {
}

func (h *Noop) ID() string {
	return "noop"
}

func (h *Noop) Name() string {
	return "noop"
}

func (h *Noop) UpdateBytes(labels []labels.Bytes) {
}
