package id

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/prometheus/prometheus/prompb"
)

// returns metric name and id
func NameWithSha256(labels []prompb.Label) (string, string) {
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	var name string
	var b strings.Builder
	h := sha256.New()

	for i := 0; i < len(labels); i++ {
		if i > 0 {
			h.Write([]byte{'&'})
		}
		h.Write([]byte(url.QueryEscape(labels[i].Name)))
		h.Write([]byte{'='})
		h.Write([]byte(url.QueryEscape(labels[i].Value)))

		if labels[i].Name == "__name__" {
			name = labels[i].Value
		}
	}

	b.WriteString(url.QueryEscape(name))
	b.WriteByte('?')
	fmt.Fprintf(&b, "%x", h.Sum(nil))

	return name, b.String()
}
