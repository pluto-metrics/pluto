package id

import (
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
)

type Noop struct {
}

var noopBytes = []byte("noop")

func NewNoop() *Noop {
	return &Noop{}
}

func (h *Noop) ID() []byte {
	return noopBytes
}

func (h *Noop) Name() []byte {
	return noopBytes
}

func (h *Noop) Update(labels []labels.Bytes) {
}
