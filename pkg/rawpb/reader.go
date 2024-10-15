package rawpb

type reader struct {
	body   []byte
	offset int
}

func newReader(body []byte) *reader {
	return &reader{body: body}
}

func (r *reader) varint() (uint64, error) {
	var ret uint64
	i := 0
	for r.next() {
		ret += uint64(r.body[r.offset]&0x7f) << (7 * uint64(i))
		if r.body[r.offset]&0x80 == 0 { // last byte of varint
			r.offset++
			return ret, nil
		}
		r.offset++
		i++
	}
	return ret, ErrorTruncated
}

func (r *reader) next() bool {
	return r.offset < len(r.body)
}

func (r *reader) bytes(n int) ([]byte, error) {
	if r.offset+n > len(r.body) {
		return nil, ErrorTruncated
	}
	v := r.body[r.offset : r.offset+n]
	r.offset += n
	return v, nil
}

func (r *reader) lengthDelimited() ([]byte, error) {
	l, err := r.varint()
	if err != nil {
		return nil, err
	}
	return r.bytes(int(l))
}
