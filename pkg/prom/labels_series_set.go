package prom

import (
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/util/annotations"
)

// SeriesSet contains a set of series.
type labelsSeriesSet struct {
	data    []labels.Labels
	current int
}

type labelsSeries struct {
	data labels.Labels
}

var _ storage.SeriesSet = &labelsSeriesSet{}

func (ms *labelsSeriesSet) At() storage.Series {
	return &labelsSeries{data: ms.data[ms.current]}
}

// Iterator returns a new iterator of the data of the series.
func (s *labelsSeries) Iterator(iterator chunkenc.Iterator) chunkenc.Iterator {
	return emptyIteratorValue
}

func (s *labelsSeries) Labels() labels.Labels {
	return s.data
}

// Err returns the current error.
func (ms *labelsSeriesSet) Err() error { return nil }

func (ms *labelsSeriesSet) Next() bool {
	if ms.current < 0 {
		ms.current = 0
	} else {
		ms.current++
	}

	return ms.current < len(ms.data)
}

func newLabelsSeriesSet(metrics []labels.Labels) storage.SeriesSet {
	return &labelsSeriesSet{data: metrics, current: -1}
}

// Warnings ...
func (s *labelsSeriesSet) Warnings() annotations.Annotations { return nil }
