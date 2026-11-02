package config

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/lesomnus/mkot/pretty"
	"github.com/lesomnus/otx"
	"github.com/lesomnus/otx/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	collectorlog "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

func TestOtelDefaults(t *testing.T) {
	for _, text := range []string{"", "enabled: false", "enabled: true"} {
		t.Run(text, func(t *testing.T) {
			var c OtelConfig
			require.NoError(t, yaml.Unmarshal([]byte(text), &c))
			require.NoError(t, c.Evaluate())
			if text == "enabled: true" {
				e, ok := c.Exporters["pretty"].(*pretty.ExporterConfig)
				require.True(t, ok)
				require.Equal(t, []string{"stdout"}, e.OutputPaths)
			} else {
				require.Empty(t, c.Exporters)
			}
			o, err := c.Build(context.Background())
			require.NoError(t, err)
			require.NoError(t, o.Start(context.Background()))
			require.NoError(t, o.Shutdown(context.Background()))
		})
	}
}

func TestExampleOtelConfig(t *testing.T) {
	b, err := os.ReadFile("../../oras-get.yaml")
	require.NoError(t, err)
	var c Config
	require.NoError(t, yaml.Unmarshal(b, &c))
	require.NoError(t, c.Otel.Evaluate())
	o, err := c.Otel.Build(context.Background())
	require.NoError(t, err)
	require.NoError(t, o.Start(context.Background()))
	require.NoError(t, o.Shutdown(context.Background()))
}

func TestLegacyOTLPHTTPExportsStructuredLog(t *testing.T) {
	type received struct {
		path string
		body []byte
		err  error
	}
	records := make(chan received, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		records <- received{r.URL.Path, b, err}
		w.Header().Set("Content-Type", "application/x-protobuf")
	}))
	defer srv.Close()
	var c OtelConfig
	require.NoError(t, yaml.Unmarshal([]byte(fmt.Sprintf(`
enabled: true
exporters:
  otlphttp:
    endpoint: %s
providers:
  logger:
    exporters: [otlphttp]
`, srv.URL)), &c))
	require.NoError(t, c.Evaluate())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	o, err := c.Build(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, o.Shutdown(context.Background())) })
	require.NoError(t, o.Start(ctx))
	log.From(otx.Into(ctx, o)).Info("export regression", slog.String("registry", "example.test"), slog.Int("count", 3))
	require.NoError(t, o.ForceFlush(ctx))
	select {
	case got := <-records:
		require.NoError(t, got.err)
		require.Equal(t, "/v1/logs", got.path)
		var request collectorlog.ExportLogsServiceRequest
		require.NoError(t, proto.Unmarshal(got.body, &request))
		require.Len(t, request.ResourceLogs, 1)
		r := request.ResourceLogs[0]
		resources := map[string]string{}
		for _, a := range r.Resource.Attributes {
			resources[a.Key] = a.Value.GetStringValue()
		}
		require.Equal(t, "oras-get", resources["service.name"])
		require.Len(t, r.ScopeLogs, 1)
		require.Len(t, r.ScopeLogs[0].LogRecords, 1)
		record := r.ScopeLogs[0].LogRecords[0]
		require.Equal(t, "export regression", record.Body.GetStringValue())
		require.Len(t, record.Attributes, 2)
		require.Equal(t, "registry", record.Attributes[0].Key)
		require.Equal(t, "example.test", record.Attributes[0].Value.GetStringValue())
		require.Equal(t, "count", record.Attributes[1].Key)
		require.EqualValues(t, 3, record.Attributes[1].Value.GetIntValue())
	case <-ctx.Done():
		t.Fatal("log did not reach the configured HTTP collector")
	}
}

func TestRegistryUsesApplicationTrace(t *testing.T) {
	var parent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parent = r.Header.Get("traceparent")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"repositories":[]}`)
	}))
	defer srv.Close()
	p := trace.NewTracerProvider()
	o := otx.New(otx.WithTracerProvider(p), otx.WithPropagator(propagation.TraceContext{}))
	t.Cleanup(func() { require.NoError(t, o.Shutdown(context.Background())) })
	ctx, span := o.Tracer().Start(otx.Into(context.Background(), o), "pull")
	defer span.End()
	var c RegistryConfig
	require.NoError(t, yaml.Unmarshal([]byte("addr: "+srv.URL), &c))
	require.NoError(t, c.Evaluate())
	upstream, err := c.Build(ctx)
	require.NoError(t, err)
	require.NoError(t, upstream.Registry.Repositories(ctx, "", func([]string) error { return nil }))
	require.Contains(t, parent, span.SpanContext().TraceID().String())
}
