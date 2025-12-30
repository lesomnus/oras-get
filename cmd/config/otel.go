package config

import (
	"context"
	"errors"
	"os"

	"github.com/lesomnus/mkot"
	"github.com/lesomnus/oras-get/cmd/version"
	"github.com/lesomnus/otx"
	"github.com/lesomnus/z"
	"go.opentelemetry.io/otel/attribute"
	nooplog "go.opentelemetry.io/otel/log/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	_ "github.com/lesomnus/mkot/otlp"
	_ "github.com/lesomnus/mkot/otlphttp"
	"github.com/lesomnus/mkot/pretty"
	_ "github.com/lesomnus/mkot/pretty"
)

type OtelConfig struct {
	mkot.Config `yaml:",inline"`
}

func (c *OtelConfig) Evaluate() error {
	if !c.Enabled {
		*c = OtelConfig{}
		return nil
	}
	if !(c.Processors == nil && c.Exporters == nil && c.Providers == nil) {
		return nil
	}

	c.Exporters = map[mkot.Id]mkot.ExporterConfig{
		"pretty": &pretty.Config{
			Output: os.Stdout,
		},
	}
	c.Providers = map[mkot.Id]*mkot.ProviderConfig{
		"logger": {
			Exporters: []mkot.Id{"pretty"},
		},
	}

	return nil
}

func (c *OtelConfig) Build(ctx context.Context) (*otx.Otx, error) {
	if !c.Enabled {
		return otx.New(), nil
	}
	if c.Processors == nil {
		c.Processors = map[mkot.Id]mkot.ProcessorConfig{}
	}

	const ServiceResourceId mkot.Id = "resource/oras-get"
	c.Processors[ServiceResourceId] = &mkot.ResourceProcessor{
		Attributes: []mkot.Attribute{
			{Key: "service.name", Value: attribute.StringValue("oras-get")},
			{Key: "service.version", Value: attribute.StringValue(version.Get().Version)},
		},
	}
	for k := range c.Providers {
		c.Providers[k].Processors = append(c.Providers[k].Processors, ServiceResourceId)
	}

	resolver := mkot.Make(ctx, &c.Config)

	tracker_provider, err := resolver.Tracer(ctx, "")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, z.Err(err, "resolve tracer provider")
		}
		tracker_provider = sdktrace.NewTracerProvider()
	}

	logger_provider, err := resolver.Logger(ctx, "")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, z.Err(err, "resolve logger provider")
		}
		logger_provider = nooplog.NewLoggerProvider()
	}

	return otx.New(
		otx.WithController(resolver),
		otx.WithTracerProvider(tracker_provider),
		otx.WithLoggerProvider(logger_provider),
	), nil
}
