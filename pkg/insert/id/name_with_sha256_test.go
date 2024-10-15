package id

import (
	"testing"

	"github.com/pluto-metrics/pluto/pkg/insert/labels"
)

func BenchmarkNameWithSHA256(b *testing.B) {

	ll := []labels.Bytes{
		{Name: []byte("__name__"), Value: []byte("go_gc_duration_seconds")},
		{Name: []byte("instance"), Value: []byte("localhost:9090")},
		{Name: []byte("job"), Value: []byte("prometheus")},
		{Name: []byte("quantile"), Value: []byte("0.25")},
	}

	h := NewNameWithSha256()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Update(ll)
	}
}
