package prom

import (
	"fmt"
	"iter"

	"github.com/OneOfOne/xxhash"
	"github.com/pluto-metrics/pluto/pkg/sql"
	"github.com/pluto-metrics/rowbinary"
	"github.com/pluto-metrics/rowbinary/schema"
)

type hashSelectorAlgo int

const (
	hashSelectorAlgoNone     hashSelectorAlgo = 0
	hashSelectorAlgoXxHash32                  = 1
	// hashSelectorUseXxHash64                 = 2
)

type hashSelector struct {
	algo hashSelectorAlgo
	mp32 map[uint32]string
}

// TODO: test in prod if hash32 is sufficient. Metric required
func NewHashSelector(values iter.Seq[string]) *hashSelector {
	mp32 := make(map[uint32]string)

	count := 0
	for v := range values {
		count += 1
		mp32[xxhash.ChecksumString32(v)] = v
	}

	if len(mp32) == count {
		return &hashSelector{
			algo: hashSelectorAlgoXxHash32,
			mp32: mp32,
		}
	}

	return &hashSelector{
		algo: hashSelectorAlgoNone,
	}
}

func (h *hashSelector) SelectColumn(column string) string {
	if h.algo == hashSelectorAlgoXxHash32 {
		return fmt.Sprintf("xxHash32(%s)", sql.Column(column))
	}

	return sql.Column(column)
}

func (h *hashSelector) ColumnType() rowbinary.Any {
	if h.algo == hashSelectorAlgoXxHash32 {
		return rowbinary.UInt32
	}
	return rowbinary.String
}

func (h *hashSelector) SchemaRead(r *schema.Reader) (string, error) {
	if h.algo == hashSelectorAlgoXxHash32 {
		v, err := schema.Read(r, rowbinary.UInt32)
		if err != nil {
			return "", err
		}
		return h.mp32[v], nil
	}

	return schema.Read(r, rowbinary.String)
}
