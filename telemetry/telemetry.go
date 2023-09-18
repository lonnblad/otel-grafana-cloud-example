package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	otel_trace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/lonnblad/basic-otel-grafana-cloud-example/config"
)

func NewResource(ctx context.Context) (_ *resource.Resource, err error) {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(config.GetServiceName()),
			semconv.ServiceVersion(config.GetServiceVersion()),
			semconv.DeploymentEnvironment(config.GetEnvironment()),
		),
	)
	if err != nil {
		err = fmt.Errorf("couldn't create the resource: %w", err)
		return
	}

	return res, nil
}

func InitMeter(ctx context.Context, res *resource.Resource) (_ *metric.MeterProvider, err error) {
	metricExp, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(config.GetOTELExporterEndpointUrl()),
		otlpmetricgrpc.WithDialOption(grpc.WithBlock()),
	)
	if err != nil {
		err = fmt.Errorf("couldn't create a otlpmetricgrpc exporter: %w", err)
		return
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(
				metricExp,
				metric.WithInterval(2*time.Second),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		err = fmt.Errorf("couldn't start runtime meter: %w", err)
		return
	}

	return meterProvider, nil
}

func InitTracer(ctx context.Context, res *resource.Resource) (_ *trace.TracerProvider, err error) {
	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(config.GetOTELExporterEndpointUrl()),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))

	exporter, err := otlptrace.New(ctx, traceClient)
	if err != nil {
		err = fmt.Errorf("couldn't create a otlptrace exporter: %w", err)
		return
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return traceProvider, nil
}

type OtelHandler struct {
	slog.Handler
}

func NewOtelHandler(h slog.Handler) *OtelHandler {
	return &OtelHandler{Handler: h}
}

var _ slog.Handler = &OtelHandler{}

func (otel OtelHandler) Handle(ctx context.Context, rec slog.Record) error {
	const (
		traceIDKey = "traceId"
		spanIDKey  = "spanId"
	)

	spanCtx := otel_trace.SpanContextFromContext(ctx)

	if spanCtx.HasTraceID() {
		rec.AddAttrs(slog.String(traceIDKey, spanCtx.TraceID().String()))
	}

	if spanCtx.HasSpanID() {
		rec.AddAttrs(slog.String(spanIDKey, spanCtx.SpanID().String()))
	}

	return otel.Handler.Handle(ctx, rec)
}
