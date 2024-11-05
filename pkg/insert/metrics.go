package insert

import (
	"log"

	"go.opentelemetry.io/otel"
)

var meter = otel.Meter("github.com/pluto-metrics/pluto/pkg/insert")

var metricSamplesReceived = must(meter.Int64Counter("pluto_samples_received_total"))

var metricRemoteWriteRequests = must(meter.Int64Counter("pluto_remote_write_requests_total"))

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}
