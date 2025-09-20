package prom

import (
	"strings"

	"github.com/prometheus/prometheus/model/labels"
)

func labelsMapKey(lb labels.Labels) string {
	v := new(strings.Builder)

	lb.Range(func(l labels.Label) {
		if v.Len() > 0 {
			v.WriteByte('&')
		}
		v.WriteString(l.Name)
		v.WriteByte('=')
		v.WriteString(l.Value)
	})

	return v.String()
}
