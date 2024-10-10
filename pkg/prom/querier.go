package prom

import (
	"github.com/pluto-metrics/pluto/pkg/config"
)

// Querier provides reading access to time series data.
type Querier struct {
	config *config.Config
	mint   int64
	maxt   int64
}

// Close releases the resources of the Querier.
func (q *Querier) Close() error {
	return nil
}
