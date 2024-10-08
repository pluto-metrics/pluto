package labels

import (
	"errors"

	rb "github.com/pluto-metrics/rowbinary"
	"github.com/prometheus/prometheus/prompb"
)

var ColumnPrompb rb.Type[[]prompb.Label] = &typeColumnPrompb{}

type typeColumnPrompb struct {
}

// Read implements rb.Type.
func (t *typeColumnPrompb) Read(r rb.Reader) ([]prompb.Label, error) {
	n, err := rb.UVarint.Read(r)
	if err != nil {
		return nil, err
	}

	ret := make([]prompb.Label, int(n))
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
func (t *typeColumnPrompb) ReadAny(r rb.Reader) (any, error) {
	return t.Read(r)
}

// String implements rb.Type.
func (t *typeColumnPrompb) String() string {
	return "Map(String, String)"
}

// Write implements rb.Type.
func (t *typeColumnPrompb) Write(w rb.Writer, value []prompb.Label) error {
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
func (t *typeColumnPrompb) WriteAny(w rb.Writer, v any) error {
	value, ok := v.([]prompb.Label)
	if !ok {
		return errors.New("unexpected type")
	}
	return t.Write(w, value)
}
