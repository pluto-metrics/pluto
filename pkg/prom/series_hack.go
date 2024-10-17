package prom

import (
	"math"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
)

func labelsMapKeyLE(lb labels.Labels) (string, string) {
	v := new(strings.Builder)
	le := ""

	for _, l := range lb {
		if l.Name == "le" {
			le = l.Value
			continue
		}
		v.WriteString(l.Name)
		v.WriteByte('=')
		v.WriteString(l.Value)
	}

	return v.String(), le
}

func quantileGroupCleanup(group []*series) {
	// slow variant for test
	// @TODO
	tsMap := make(map[int64]int)
	for _, s := range group {
		for i := 0; i < len(s.samples); i++ {
			tsMap[s.samples[i].timestamp]++
		}
	}

	// keep only in many series
	for _, s := range group {
		n := make([]sample, 0, len(s.samples))
		for i := 0; i < len(s.samples); i++ {
			if tsMap[s.samples[i].timestamp] == len(group) {
				n = append(n, s.samples[i])
			}
		}
		s.samples = n
	}
}

func hackSeriesNanPoint(s *series, hints hints) {
	if hints.step == 0 {
		return
	}
	if hints.function != "rate" {
		return
	}
	if len(s.samples) < 1 {
		return
	}
	s.samples = append(s.samples, sample{timestamp: s.samples[len(s.samples)-1].timestamp + hints.step, value: math.NaN()})
}

func hackSeries(data []series, hints hints) []series {
	groups := make(map[string][]*series)

	for i := 0; i < len(data); i++ {
		k, le := labelsMapKeyLE(data[i].labels)
		if le == "" {
			hackSeriesNanPoint(&data[i], hints)
			continue
		}
		groups[k] = append(groups[k], &data[i])
	}

	// cleanup each group
	for _, g := range groups {
		if len(g) < 2 {
			if len(g) == 1 {
				hackSeriesNanPoint(g[0], hints)
			}
			continue
		}
		quantileGroupCleanup(g)
	}
	return data
}
