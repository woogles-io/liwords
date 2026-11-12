package main

import (
	"os"
	"testing"

	"github.com/matryer/is"
)

// Spans keep the service name and version from the environment, and say
// which host they came from.
func TestTraceResource(t *testing.T) {
	is := is.New(t)
	t.Setenv("OTEL_SERVICE_NAME", "liwords")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.version=0.1.0")
	res, err := traceResource()
	is.NoErr(err)

	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.Emit()
	}
	hostname, err := os.Hostname()
	is.NoErr(err)
	is.Equal(got["service.name"], "liwords")
	is.Equal(got["service.version"], "0.1.0")
	is.Equal(got["host.name"], hostname)
}
