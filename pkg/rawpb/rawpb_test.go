package rawpb

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
)

func readFixture(name string) []byte {
	gz, err := os.ReadFile("fixtures/" + name)
	if err != nil {
		panic(err)
	}
	gzReader, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		panic(err)
	}
	raw, err := io.ReadAll(gzReader)
	if err != nil {
		panic(err)
	}
	return raw
}
func TestRawPB(t *testing.T) {
	assert := assert.New(t)
	body := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")

	r := New(
		Begin(func() error { return nil }),
		End(func() error { return nil }),
		FieldNested(1, New(
			Begin(func() error { return nil }),
			End(func() error { return nil }),
			FieldNested(1, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				FieldString(1, func(v string) error {
					return nil
				}),
				FieldString(2, func(v string) error {
					return nil
				}),
			)),
			FieldNested(2, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				FieldFloat64(1, func(v float64) error {
					return nil
				}),
				FieldInt64(2, func(v int64) error {
					return nil
				}),
			)),
		)),
	)

	err := r.Parse(body)
	assert.NoError(err)
}

func BenchmarkParse(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	r := New(
		Begin(func() error { return nil }),
		End(func() error { return nil }),
		FieldNested(1, New(
			Begin(func() error { return nil }),
			End(func() error { return nil }),
			FieldNested(1, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				FieldBytes(1, func(v []byte) error {
					return nil
				}),
				FieldBytes(2, func(v []byte) error {
					return nil
				}),
			)),
			FieldNested(2, New(
				Begin(func() error { return nil }),
				End(func() error { return nil }),
				FieldFloat64(1, func(v float64) error {
					return nil
				}),
				FieldInt64(2, func(v int64) error {
					return nil
				}),
			)),
		)),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Parse(raw); err != nil {
			panic(err)
		}
	}
}

func BenchmarkProtoUnmarshal(b *testing.B) {
	var req prompb.WriteRequest
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := proto.Unmarshal(raw, &req); err != nil {
			panic(err)
		}
	}
}
