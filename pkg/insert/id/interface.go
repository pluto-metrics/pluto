package id

import (
	"github.com/lomik/rowbinary"
)

type Provider interface {
	Update(labels *rowbinary.KV[[]byte, []byte])
	ID() []byte
	Name() []byte
}
