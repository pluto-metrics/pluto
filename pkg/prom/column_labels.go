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
		return nil, err
	}

	ret := make(labels.Labels, int(n))
	for i := uint64(0); i < n; i++ {
		k, err := rb.String.Read(r)
		if err != nil {
			return nil, err
		}
		ret[i].Name = k

		v, err := rb.String.Read(r)
		if err != nil {
			return nil, err
		}
		ret[i].Value = v
	}

	return ret, nil
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
	err := rb.UVarint.Write(w, uint64(len(value)))
	if err != nil {
		return err
	}
	for i := 0; i < len(value); i++ {
		err = rb.String.Write(w, value[i].Name)
		if err != nil {
			return err
		}

		err = rb.String.Write(w, value[i].Value)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteAny implements rb.Type.
func (t *typeColumnLabels) WriteAny(w rb.Writer, v any) error {
	value, ok := v.(labels.Labels)
	if !ok {
		return errors.New("unexpected type")
	}
	return t.Write(w, value)
}
