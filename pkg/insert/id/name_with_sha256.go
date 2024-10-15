package id

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"sort"
	"unsafe"

	"github.com/pluto-metrics/pluto/pkg/insert/labels"
)

func unsafeBytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

var labelName = []byte("__name__")

type NameWithSha256 struct {
	name  []byte
	id    []byte
	idLen int
	hash  [32]byte
	hh    hash.Hash
	hb    *bufio.Writer
}

func NewNameWithSha256() *NameWithSha256 {
	hh := sha256.New()
	return &NameWithSha256{
		hh: hh,
		hb: bufio.NewWriter(hh),
	}
}

func (h *NameWithSha256) ID() []byte {
	return h.id
}

func (h *NameWithSha256) Name() []byte {
	return h.name
}

func (h *NameWithSha256) Update(labels []labels.Bytes) {
	h.name = nil
	h.id = nil

	sort.Slice(labels, func(i, j int) bool {
		return bytes.Compare(labels[i].Name, labels[j].Name) < 0
	})

	h.hh.Reset()
	h.hb.Reset(h.hh)

	for i := 0; i < len(labels); i++ {
		if i > 0 {
			h.hb.WriteByte('&')
		}
		h.hb.Write(labels[i].Name)
		h.hb.WriteByte('=')
		h.hb.Write(labels[i].Value)

		if h.name == nil && bytes.Equal(labels[i].Name, labelName) {
			h.name = labels[i].Value
		}
	}
	h.hb.Flush()

	if len(h.id) < len(h.name)+65 {
		h.id = make([]byte, len(h.name)+65)
	}

	h.hh.Sum(h.hash[:0])

	copy(h.id, h.name)
	h.id[len(h.name)] = '?'
	hex.Encode(h.id[len(h.name)+1:], h.hash[:])
}
