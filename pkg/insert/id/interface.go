package id

import (
	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/prometheus/prometheus/prompb"
)

type Provider interface {
	Update(labels []prompb.Label)
	UpdateBytes(labels []labels.Bytes)
	ID() string
	Name() string
}
