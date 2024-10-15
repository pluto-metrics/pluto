package id

import "github.com/prometheus/prometheus/prompb"

type Provider interface {
	Update(labels []prompb.Label)
	ID() string
	Name() string
}
