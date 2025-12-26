package config

import (
	"context"
	"errors"
	"os"

	"github.com/lesomnus/mkot"
	"github.com/lesomnus/otx"
	"github.com/lesomnus/z"
	"go.opentelemetry.io/otel/attribute"
	nooplog "go.opentelemetry.io/otel/log/noop"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	_ "github.com/lesomnus/mkot/exporters/otlp"
	_ "github.com/lesomnus/mkot/exporters/otlphttp"
	_ "github.com/lesomnus/mkot/exporters/pretty"
)

type OtelConfig struct {
	mkot.Config
}

func (c *OtelConfig) Build(ctx context.Context) (*otx.Otx, error) {
	otc := mkot.NewConfig()
	if c != nil {
		otc = &c.Config
	}
	if otc.Processors == nil {
		otc.Processors = map[mkot.Id]mkot.ProcessorConfig{}
	}

	const ServiceResourceId mkot.Id = "resource/internal"
	otc.Processors[ServiceResourceId] = &mkot.ResourceConfig{
		Attributes: []mkot.Attribute{
			{Key: "service.name", Value: attribute.StringValue("ora-get")},
			{Key: "service.version", Value: attribute.StringValue("dev")},
		},
	}
	for k := range otc.Providers {
		otc.Providers[k].Processors = append(otc.Providers[k].Processors, ServiceResourceId)
	}

	resolver := mkot.Make(ctx, otc)

	tracker_provider, err := resolver.Tracer(ctx, "")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, z.Err(err, "resolve tracer provider")
		}
		tracker_provider = sdktrace.NewTracerProvider()
	}

	meter_provider, err := resolver.Meter(ctx, "")
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, z.Err(err, "resolve meter provider")
		}
		meter_provider = noopmetric.NewMeterProvider()
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
		otx.WithMeterProvider(meter_provider),
		otx.WithLoggerProvider(logger_provider),
	), nil
}
