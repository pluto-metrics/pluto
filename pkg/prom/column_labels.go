package prom

import (
	"errors"

	"github.com/lomik/rowbinary"
	"github.com/prometheus/prometheus/model/labels"
)

var ColumnLabels rowbinary.Type[labels.Labels] = rowbinary.MakeTypeWrapAny(&typeColumnLabels{})

type typeColumnLabels struct {
}

// String implements rb.Type.
func (t typeColumnLabels) String() string {
	return rowbinary.Map(rowbinary.String, rowbinary.String).String()
}

func (t typeColumnLabels) Binary() []byte {
	return rowbinary.Map(rowbinary.String, rowbinary.String).Binary()
}

func (t typeColumnLabels) Scan(r rowbinary.Reader, value *labels.Labels) error {
	n, err := rowbinary.VarintRead(r)
	if err != nil {
		return labels.New(), err
	}

	ss := make([]string, 0, int(n)*2)

	for i := uint64(0); i < n; i++ {
		k, err := rowbinary.String.Read(r)
		if err != nil {
			return labels.New(), err
		}
		ss = append(ss, k)

		v, err := rowbinary.String.Read(r)
		if err != nil {
			return labels.New(), err
		}
		ss = append(ss, v)
	}

	*value = labels.FromStrings(ss...)

	return nil
}

// Write implements rb.Type.
func (t *typeColumnLabels) Write(w rowbinary.Writer, value labels.Labels) error {
	return errors.ErrUnsupported
}
