package prom

import (
	"errors"

	rb "github.com/pluto-metrics/rowbinary"
	"github.com/prometheus/prometheus/model/labels"
)

var ColumnLabels rb.Type[labels.Labels] = &typeColumnLabels{}

type typeColumnLabels struct {
}

// Read implements rb.Type.
func (t *typeColumnLabels) Read(r rb.Reader) (labels.Labels, error) {
	n, err := rb.UVarint.Read(r)
	if err != nil {
		return labels.New(), err
	}

	ss := make([]string, 0, int(n)*2)

	for i := uint64(0); i < n; i++ {
		k, err := rb.String.Read(r)
		if err != nil {
			return labels.New(), err
		}
		ss = append(ss, k)

		v, err := rb.String.Read(r)
		if err != nil {
			return labels.New(), err
		}
		ss = append(ss, v)
	}

	return labels.FromStrings(ss...), nil
}

// ReadAny implements rb.Type.
func (t *typeColumnLabels) ReadAny(r rb.Reader) (any, error) {
	return t.Read(r)
}

// String implements rb.Type.
func (t *typeColumnLabels) String() string {
	return "Map(String, String)"
}

// Write implements rb.Type.
func (t *typeColumnLabels) Write(w rb.Writer, value labels.Labels) error {
	err := rb.UVarint.Write(w, uint64(value.Len()))
	if err != nil {
		return err
	}

	value.Range(func(l labels.Label) {
		if err != nil {
			return
		}
		err = rb.String.Write(w, l.Name)
		if err != nil {
			return
		}
		err = rb.String.Write(w, l.Value)
	})

	return err
}

// WriteAny implements rb.Type.
func (t *typeColumnLabels) WriteAny(w rb.Writer, v any) error {
	value, ok := v.(labels.Labels)
	if !ok {
		return errors.New("unexpected type")
	}
	return t.Write(w, value)
}
