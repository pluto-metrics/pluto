package prom

import (
	"strings"

	"github.com/prometheus/prometheus/model/labels"
)

func labelsMapKey(lb labels.Labels) string {
	v := new(strings.Builder)

	for i, l := range lb {
		if i > 0 {
			v.WriteByte('&')
		}
		v.WriteString(l.Name)
		v.WriteByte('=')
		v.WriteString(l.Value)
	}

	return v.String()
}
