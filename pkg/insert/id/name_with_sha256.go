package id

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"unsafe"

	"github.com/pluto-metrics/pluto/pkg/insert/labels"
	"github.com/prometheus/prometheus/prompb"
)

func unsafeBytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

type NameWithSha256 struct {
	name string
	id   string
}

func NewNameWithSha256() *NameWithSha256 {
	return &NameWithSha256{}
}

func (h *NameWithSha256) Update(labels []prompb.Label) {
	h.name = ""
	h.id = ""

	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	var b strings.Builder
	hh := sha256.New()

	for i := 0; i < len(labels); i++ {
		if i > 0 {
			hh.Write([]byte{'&'})
		}
		hh.Write([]byte(url.QueryEscape(labels[i].Name)))
		hh.Write([]byte{'='})
		hh.Write([]byte(url.QueryEscape(labels[i].Value)))

		if labels[i].Name == "__name__" {
			h.name = labels[i].Value
		}
	}

	b.WriteString(url.PathEscape(h.name))
	b.WriteByte('?')
	fmt.Fprintf(&b, "%x", hh.Sum(nil))

	h.id = b.String()
}

func (h *NameWithSha256) ID() string {
	return h.id
}

func (h *NameWithSha256) Name() string {
	return h.name
}

func (h *NameWithSha256) UpdateBytes(labels []labels.Bytes) {
	h.name = ""
	h.id = ""

	sort.Slice(labels, func(i, j int) bool {
		return bytes.Compare(labels[i].Name, labels[j].Name) < 0
	})

	var b strings.Builder
	hh := sha256.New()

	for i := 0; i < len(labels); i++ {
		if i > 0 {
			hh.Write([]byte{'&'})
		}
		hh.Write([]byte(url.QueryEscape(unsafeBytesToString(labels[i].Name))))
		hh.Write([]byte{'='})
		hh.Write([]byte(url.QueryEscape(unsafeBytesToString(labels[i].Value))))

		if unsafeBytesToString(labels[i].Name) == "__name__" {
			h.name = unsafeBytesToString(labels[i].Value)
		}
	}

	b.WriteString(url.PathEscape(h.name))
	b.WriteByte('?')
	fmt.Fprintf(&b, "%x", hh.Sum(nil))

	h.id = b.String()
}
