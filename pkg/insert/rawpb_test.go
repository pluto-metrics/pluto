package insert

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/pluto-metrics/pluto/pkg/insert/id"
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

func BenchmarkPayloadToRowBinaryWithSHA256(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNameWithSha256()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := payloadToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}

func BenchmarkPayloadToRowBinaryWithNoop(b *testing.B) {
	raw := readFixture("34dd878af9d34cae46373dffa8df973ed94ab45be0ffa2fa0830bb1bb497ad90.gz")
	w := bufio.NewWriter(io.Discard)
	h := id.NewNoop()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := payloadToRowBinary(raw, w, h); err != nil {
			panic(err)
		}
	}
}
