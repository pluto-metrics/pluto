package prom

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/pluto-metrics/rowbinary"
	"github.com/stretchr/testify/assert"
)

// requests clickhouse, caching locally to disk
// re-running the test can already work without CH. including in CI if you commit fixtures/*
func execLocal(query string) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(query))
	key := fmt.Sprintf("%x", h.Sum(nil))
	filename := fmt.Sprintf("fixtures/ch_%s.bin", key)

	// fmt.Println(filename, query)

	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		body, err := exec.Command("clickhouse", "local", "--query", query).Output()
		if err != nil {
			return nil, err
		}

		err = os.WriteFile(filename, body, 0600)
		return body, err
	}
	// #nosec G304
	return os.ReadFile(filename)
}

func TestClickhouseHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tp    rowbinary.Any
		want  interface{}
		query string
	}{
		{rowbinary.UInt32, xxhash.ChecksumString32("aaaa"), "SELECT xxHash32('aaaa')"},
		{rowbinary.UInt64, xxhash.ChecksumString64("aaaa"), "SELECT xxHash64('aaaa')"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s/%#v", tt.tp, tt.want), func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)

			body, err := execLocal(tt.query + " AS value FORMAT RowBinary SETTINGS session_timezone='UTC'")
			assert.NoError(err)

			r := bytes.NewReader(body)
			value, err := tt.tp.ReadAny(r)
			if assert.NoError(err) {
				assert.Equal(tt.want, value)
			}

		})
	}
}
