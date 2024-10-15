package rawpb

import (
	"github.com/pkg/errors"
)

var ErrorTruncated = errors.New("Message truncated")
var ErrorUnknownWireType = errors.New("Unknown wire type")
var ErrorInvalidMessage = errors.New("Invalid message")
var ErrorWrongWireType = errors.New("Wrong wire type")

type field struct {
	varint  func(v uint64) error
	fixed64 func(v []byte) error
	bytes   func(v []byte) error
	fixed32 func(v []byte) error
}

type RawPB struct {
	// @TODO: may use array
	beginFunc func()
	schema    map[uint64]*field
	endFunc   func()
}

func New(opts ...Option) *RawPB {
	r := &RawPB{
		schema: make(map[uint64]*field),
	}

	for _, o := range opts {
		o(r)
	}

	return r
}

func (pb *RawPB) Parse(body []byte) error {
	if pb.beginFunc != nil {
		pb.beginFunc()
	}

	r := newReader(body)

	for r.next() {
		// read wire type
		tag, err := r.varint()
		if err != nil {
			return err
		}
		wt := tag % 8
		num := tag >> 3
		switch wt {
		case 0: // varint
			v, err := r.varint()
			if err != nil {
				return err
			}
			if f, ok := pb.schema[num]; ok {
				if f.varint == nil {
					return errors.WithStack(ErrorWrongWireType)
				}
				if err := f.varint(v); err != nil {
					return err
				}
			}
		case 1: // 64-bit
			v, err := r.bytes(8)
			if err != nil {
				return err
			}
			if f, ok := pb.schema[num]; ok {
				if f.fixed64 == nil {
					return errors.WithStack(ErrorWrongWireType)
				}
				if err := f.fixed64(v); err != nil {
					return err
				}
			}
		case 2: // Length-delimited
			v, err := r.lengthDelimited()
			if err != nil {
				return err
			}
			if f, ok := pb.schema[num]; ok {
				if f.bytes != nil {
					err = f.bytes(v)
					if err != nil {
						return err
					}
				}
				// @TODO: unpacked
			}
		case 5: // 32-bit
			v, err := r.bytes(4)
			if err != nil {
				return err
			}
			if f, ok := pb.schema[num]; ok {
				if f.fixed32 == nil {
					return errors.WithStack(ErrorWrongWireType)
				}
				if err := f.fixed32(v); err != nil {
					return err
				}
			}
		default:
			return errors.WithStack(ErrorUnknownWireType)
		}
	}

	if pb.endFunc != nil {
		pb.endFunc()
	}
	return nil
}
