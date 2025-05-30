package prom

import (
	"log"
	"sort"

	"github.com/prometheus/prometheus/util/annotations"

	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
)

type sample struct {
	timestamp int64
	value     float64
}

// SeriesIterator iterates over the data of a time series.
type seriesIterator struct {
	series  *series
	current int
}

// Series represents a single time series.
type series struct {
	labels     labels.Labels
	samples    []sample
	isQuantile bool // presumably the series is used in calculating the quantile
}

// SeriesSet contains a set of series.
type seriesSet struct {
	data    []series
	current int
	ann     annotations.Annotations
	err     error
}

var _ storage.SeriesSet = &seriesSet{}

func makeSeriesSet(data []series, hints *storage.SelectHints) (storage.SeriesSet, error) {
	ss := &seriesSet{data: data, current: -1}
	if len(ss.data) == 0 {
		return ss, nil
	}

	if len(data) == 0 {
		return ss, nil
	}

	// sort all samples
	for index := 0; index < len(ss.data); index++ {
		sort.Slice(ss.data[index].samples, func(i, j int) bool {
			return ss.data[index].samples[i].timestamp < ss.data[index].samples[j].timestamp
		})
	}

	// some points may not be saved in the storage yet and this breaks the histogram_quantile function. incomplete data needs to be removed
	ss.data = hackSeries(ss.data, hints)

	return ss, nil
}

func emptySeriesSet() storage.SeriesSet {
	return &seriesSet{data: make([]series, 0), current: -1}
}

func errorSeriesSet(err error) storage.SeriesSet {
	return &seriesSet{err: err, current: -1}
}

func newLabelsSeriesSet(metrics []labels.Labels) storage.SeriesSet {
	data := make([]series, len(metrics))
	for i := 0; i < len(metrics); i++ {
		data[i].labels = metrics[i]
	}
	return &seriesSet{data: data, current: -1}
}

// Seek advances the iterator forward to the value at or after
// the given timestamp.
func (sit *seriesIterator) Seek(t int64) chunkenc.ValueType {
	for ; sit.current < len(sit.series.samples); sit.current++ {
		if sit.series.samples[sit.current].timestamp >= t {
			// sit.logger().Debug("seriesIterator.Seek", zap.Int64("t", t), zap.Bool("ret", true))
			return chunkenc.ValFloat
		}
	}

	// sit.logger().Debug("seriesIterator.Seek", zap.Int64("t", t), zap.Bool("ret", false))
	return chunkenc.ValNone
}

// At returns the current timestamp/value pair.
func (sit *seriesIterator) At() (t int64, v float64) {
	index := sit.current
	if index < 0 || index >= len(sit.series.samples) {
		index = 0
	}
	p := sit.series.samples[index]
	// sit.logger().Debug("seriesIterator.At", zap.Int64("t", int64(p.Time)*1000), zap.Float64("v", p.Value))
	return p.timestamp, p.value
}

// AtHistogram returns the current timestamp/value pair if the value is
// a histogram with integer counts. Before the iterator has advanced,
// the behaviour is unspecified.
func (sit *seriesIterator) AtHistogram(histogram *histogram.Histogram) (int64, *histogram.Histogram) {
	log.Fatal("seriesIterator.AtHistogram not implemented")
	return 0, nil // @TODO
}

// AtFloatHistogram returns the current timestamp/value pair if the
// value is a histogram with floating-sample counts. It also works if the
// value is a histogram with integer counts, in which case a
// FloatHistogram copy of the histogram is returned. Before the iterator
// has advanced, the behaviour is unspecified.
func (sit *seriesIterator) AtFloatHistogram(histogram *histogram.FloatHistogram) (int64, *histogram.FloatHistogram) {
	log.Fatal("seriesIterator.AtFloatHistogram not implemented")
	return 0, nil // @TODO
}

// AtT returns the current timestamp.
// Before the iterator has advanced, the behaviour is unspecified.
func (sit *seriesIterator) AtT() int64 {
	t, _ := sit.At()
	return t
}

// Next advances the iterator by one.
func (sit *seriesIterator) Next() chunkenc.ValueType {
	if sit.current < len(sit.series.samples)-1 {
		sit.current++
		// sit.logger().Debug("seriesIterator.Next", zap.Bool("ret", true))
		return chunkenc.ValFloat
	}
	// sit.logger().Debug("seriesIterator.Next", zap.Bool("ret", false))
	return chunkenc.ValNone
}

// Err returns the current error.
func (sit *seriesIterator) Err() error { return nil }

// Err returns the current error.
func (ss *seriesSet) Err() error { return ss.err }

func (ss *seriesSet) At() storage.Series {
	if ss == nil || ss.current < 0 || ss.current >= len(ss.data) {
		// zap.L().Debug("seriesSet.At", zap.String("metricName", "nil"))
		return nil
	}
	s := &ss.data[ss.current]
	// zap.L().Debug("seriesSet.At", zap.String("metricName", s.name()))
	return s
}

func (ss *seriesSet) Next() bool {
	if ss == nil || ss.current+1 >= len(ss.data) {
		// zap.L().Debug("seriesSet.Next", zap.Bool("ret", false))
		return false
	}

	ss.current++
	// zap.L().Debug("seriesSet.Next", zap.Bool("ret", true))
	return true
}

// Warnings ...
func (ss *seriesSet) Warnings() annotations.Annotations {
	return ss.ann
}

// Iterator returns a new iterator of the data of the series.
func (s *series) Iterator(iterator chunkenc.Iterator) chunkenc.Iterator {
	return &seriesIterator{series: s, current: -1}
}

func (s *series) Labels() labels.Labels {
	return s.labels
}

func (s *series) sampleAppend(timestamp int64, value float64) {
	if s == nil {
		return
	}
	s.samples = append(s.samples, sample{timestamp: timestamp, value: value})
}
