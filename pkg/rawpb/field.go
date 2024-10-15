package rawpb

import "math"

func FieldBytes(num int, f func([]byte) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			bytes: f,
		})
	}
}

func FieldNested(num int, n *RawPB) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			bytes: n.Parse,
		})
	}
}

func FieldString(num int, f func(string) error) Option {
	return FieldBytes(num, func(b []byte) error {
		return f(string(b))
	})
}

func FieldInt64(num int, f func(int64) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			varint: func(v uint64) error {
				return f(int64(v))
			},
		})
	}
}

func FieldFloat64(num int, f func(float64) error) Option {
	return func(p *RawPB) {
		p.setField(num, field{
			fixed64: func(p []byte) error {
				u := uint64(p[0]) | (uint64(p[1]) << 8) | (uint64(p[2]) << 16) | (uint64(p[3]) << 24) |
					(uint64(p[4]) << 32) | (uint64(p[5]) << 40) | (uint64(p[6]) << 48) | (uint64(p[7]) << 56)

				return f(math.Float64frombits(u))
			},
		})
	}
}
