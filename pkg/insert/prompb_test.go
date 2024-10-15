package insert

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/pluto-metrics/pluto/pkg/insert/id"
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

	buf1 := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	h := id.NewNameWithSha256()

	w1 := bufio.NewWriter(buf1)
	w2 := bufio.NewWriter(buf2)

	assert.NoError(naivePromPbToRowBinary(raw, w1, h))
	assert.NoError(rawpbPromPbToRowBinary(raw, w2, h))

	w1.Flush()
	w2.Flush()

	assert.Equal(buf1.Bytes(), buf2.Bytes())
}

func BenchmarkNaivePromPbToRowBinaryWithSHA256(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNameWithSha256()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := naivePromPbToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}

func BenchmarkNaivePromPbToRowBinaryWithNoop(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNoop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := naivePromPbToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}

func BenchmarkRawpbPromPbToRowBinaryWithSHA256(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNameWithSha256()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rawpbPromPbToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}

func BenchmarkRawpbPromPbToRowBinaryWithNoop(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNoop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := rawpbPromPbToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}
