package prom

import (
	"math"
	"strings"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
)

func isHistogram(data []series) bool {
	if len(data) < 2 {
		return false
	}

	for i := 0; i < len(data); i++ {
		isBucket := false
		if data[i].labels.Get("le") != "" {
			isBucket = true
		}
		if !isBucket {
			return false
		}
	}

	return true
}

func keyHistogram(lb labels.Labels) (string, string) {
	v := new(strings.Builder)
	le := ""

	lb.Range(func(l labels.Label) {
		if l.Name == "le" {
			le = l.Value
			return
		}
		v.WriteString(l.Name)
		v.WriteByte('=')
		v.WriteString(l.Value)
	})

	return v.String(), le
}

func hackSingleHistogram(h []*series, hints *storage.SelectHints) {
	// slow variant for test
	// @TODO
	tsMap := make(map[int64]int)
	for _, s := range h {
		for i := 0; i < len(s.samples); i++ {
			tsMap[s.samples[i].timestamp]++
		}
	}

	// keep only in many series
	for _, s := range h {
		n := make([]sample, 0, len(s.samples))
		for i := 0; i < len(s.samples); i++ {
			if tsMap[s.samples[i].timestamp] == len(h) {
				n = append(n, s.samples[i])
			}
		}
		s.samples = n
	}
}

func hackHistogram(data []series, hints *storage.SelectHints) []series {
	groups := make(map[string][]*series)

	for i := 0; i < len(data); i++ {
		k, le := keyHistogram(data[i].labels)
		if le == "" {
			continue
		}
		groups[k] = append(groups[k], &data[i])
	}

	// cleanup each group
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		hackSingleHistogram(g, hints)
	}
	return data
}

func hackRate(data []series, hints *storage.SelectHints) []series {
	for i := 0; i < len(data); i++ {
		if len(data[i].samples) == 0 {
			continue
		}
		data[i].samples = append(data[i].samples, sample{
			timestamp: data[i].samples[len(data[i].samples)-1].timestamp + hints.Step,
			value:     math.NaN(),
		})
	}
	return data
}

func hackSeries(data []series, hints *storage.SelectHints) []series {
	if isHistogram(data) {
		return hackHistogram(data, hints)
	}

	if hints.Step > 0 && hints.Func == "rate" {
		return hackRate(data, hints)
	}

	return data
}
