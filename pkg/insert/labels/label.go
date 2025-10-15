package labels

import (
	"errors"

	"github.com/lomik/rowbinary"
)

type Label struct {
	Name  []byte
	Value []byte
}

var ColumnLabels rowbinary.Type[[]Label] = rowbinary.MakeTypeWrapAny[[]Label](typeColumnLabels{})

type typeColumnLabels struct {
}

func (t typeColumnLabels) String() string {
	return rowbinary.Map(rowbinary.String, rowbinary.String).String()
}

func (t typeColumnLabels) Binary() []byte {
	return rowbinary.Map(rowbinary.String, rowbinary.String).Binary()
}

func (t typeColumnLabels) Scan(r rowbinary.Reader, value *[]Label) error {
	return errors.ErrUnsupported
}

func (t typeColumnLabels) Write(w rowbinary.Writer, value []Label) error {
	if err := rowbinary.VarintWrite(w, uint64(len(value))); err != nil {
		return err
	}
	for _, l := range value {
		if err := rowbinary.StringBytes.Write(w, l.Name); err != nil {
			return err
		}
		if err := rowbinary.StringBytes.Write(w, l.Value); err != nil {
			return err
		}
	}
	return nil
}
