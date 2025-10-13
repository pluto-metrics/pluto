package id

import (
	"github.com/lomik/pluto/pkg/insert/labels"
)

type Provider interface {
	Update(labels []labels.Bytes)
	ID() []byte
	Name() []byte
}
