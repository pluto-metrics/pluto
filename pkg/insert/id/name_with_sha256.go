package id

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"unsafe"

	"github.com/lomik/rowbinary"
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

func (h *NameWithSha256) Update(labels *rowbinary.KV[[]byte, []byte]) {
	h.name = nil
	h.id = nil

	labels.Sort(func(a, b []byte) bool { return bytes.Compare(a, b) < 0 })

	h.hh.Reset()
	h.hb.Reset(h.hh)

	i := 0
	labels.Each(func(key, value []byte) error {
		if i > 0 {
			h.hb.WriteByte('&')
		}
		h.hb.Write(key)
		h.hb.WriteByte('=')
		h.hb.Write(value)

		if h.name == nil && bytes.Equal(key, labelName) {
			h.name = value
		}

		i++
		return nil
	})

	h.hb.Flush()

	if len(h.id) < len(h.name)+65 {
		h.id = make([]byte, len(h.name)+65)
	}

	h.hh.Sum(h.hash[:0])

	copy(h.id, h.name)
	h.id[len(h.name)] = '?'
	hex.Encode(h.id[len(h.name)+1:], h.hash[:])
}
