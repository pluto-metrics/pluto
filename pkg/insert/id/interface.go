package id

import (
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
)

type Provider interface {
	Update(labels []labels.Bytes)
	ID() []byte
	Name() []byte
}
