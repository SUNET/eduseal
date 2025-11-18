package metric

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Metric is a wrapper for opentelemetry metric
type Metric struct {
	Provider *sdkmetric.MeterProvider
	exporter *otlpmetricgrpc.Exporter
	log      *logger.Log
	metric.Meter
}

func (c *Metric) newExporter(ctx context.Context, cfg *model.Cfg) error {
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(cfg.Common.Metric.Addr),
		otlpmetricgrpc.WithTimeout(time.Duration(cfg.Common.Metric.Timeout)*time.Second),
	)
	if err != nil {
		return err
	}
	c.exporter = exp

	return nil
}

func (c *Metric) newProvider(serviceName string) {
	if c.exporter == nil {
		panic("exporter is nil")
	}

	c.Provider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(c.exporter),
		),
	)
	sdkmetric.WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	))

	otel.SetMeterProvider(c.Provider)
}

// New return a new metric
func New(ctx context.Context, cfg *model.Cfg, serviceName string, log *logger.Log) (*Metric, error) {
	m := &Metric{
		log: log,
	}

	// Exporter
	err := m.newExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Provider
	m.newProvider(serviceName)

	m.Meter = otel.Meter(serviceName)

	log.Info("Started")

	return m, nil
}

func NewForTesting(ctx context.Context, serviceName string) (*Metric, error) {
	c := &Metric{}

	// Provider
	c.Provider = sdkmetric.NewMeterProvider()

	c.Meter = otel.Meter(serviceName)

	return c, nil
}
