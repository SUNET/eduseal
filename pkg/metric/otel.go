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

func (m *Metric) newExporter(ctx context.Context, cfg *model.Cfg) error {
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(cfg.Common.Metric.Addr),
		otlpmetricgrpc.WithTimeout(time.Duration(cfg.Common.Metric.Timeout)*time.Second),
	)
	if err != nil {
		return err
	}
	m.exporter = exp

	return nil
}

func (m *Metric) newProvider(serviceName string) {
	if m.exporter == nil {
		panic("exporter is nil")
	}

	m.Provider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(m.exporter)))
	sdkmetric.WithResource(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	))

	otel.SetMeterProvider(m.Provider)
}

// New return a new metric
func New(ctx context.Context, cfg *model.Cfg, log *logger.Log, serviceName string) (*Metric, error) {
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

func NewSimple(ctx context.Context, serviceName string) (*Metric, error) {
	m := &Metric{}

	// Provider
	m.Provider = sdkmetric.NewMeterProvider()

	m.Meter = otel.Meter(serviceName)

	return m, nil
}
