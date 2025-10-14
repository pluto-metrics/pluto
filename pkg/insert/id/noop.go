package id

import "github.com/lomik/rowbinary"

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

func (h *Noop) Update(labels []rowbinary.KV[[]byte, []byte]) {
}
