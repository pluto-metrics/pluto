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
	beginFunc func() error
	schema    []field // @TODO unlimited schema
	endFunc   func() error
}

func New(opts ...Option) *RawPB {
	r := &RawPB{
		schema: make([]field, 0),
	}

	for _, o := range opts {
		o(r)
	}

	return r
}

func (pb *RawPB) setField(num int, f field) {
	for len(pb.schema) <= num {
		pb.schema = append(pb.schema, field{})
	}

	pb.schema[num] = f
}

func (pb *RawPB) Parse(body []byte) error {
	if pb.beginFunc != nil {
		if err := pb.beginFunc(); err != nil {
			return err
		}
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
			if pb.schema[num].varint != nil {
				if err := pb.schema[num].varint(v); err != nil {
					return err
				}
			}
		case 1: // 64-bit
			v, err := r.bytes(8)
			if err != nil {
				return err
			}
			if pb.schema[num].fixed64 != nil {
				if err := pb.schema[num].fixed64(v); err != nil {
					return err
				}
			}
		case 2: // Length-delimited
			v, err := r.lengthDelimited()
			if err != nil {
				return err
			}
			if pb.schema[num].bytes != nil {
				err = pb.schema[num].bytes(v)
				if err != nil {
					return err
				}
				// @TODO: unpacked
			}
		case 5: // 32-bit
			v, err := r.bytes(4)
			if err != nil {
				return err
			}
			if pb.schema[num].fixed32 != nil {
				if err := pb.schema[num].fixed32(v); err != nil {
					return err
				}
			}
		default:
			return errors.WithStack(ErrorUnknownWireType)
		}
	}

	if pb.endFunc != nil {
		if err := pb.endFunc(); err != nil {
			return err
		}
	}
	return nil
}
