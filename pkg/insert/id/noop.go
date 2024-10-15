package id

import (
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
