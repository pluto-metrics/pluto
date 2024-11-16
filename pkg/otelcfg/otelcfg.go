package otelcfg

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

type ConfigTraceOtlpHttp struct {
	Endpoint string            `yaml:"endpoint" validate:"uri" default:""`
	Headers  map[string]string `yaml:"headers"`
}

type ConfigTrace struct {
	Stdout   bool                `yaml:"stdout" default:"false"`
	OtlpHttp ConfigTraceOtlpHttp `yaml:"otlp_http"`
}

type Config struct {
	Trace ConfigTrace `yaml:"trace"`
}

type Manager interface {
	Shutdown(ctx context.Context) error
}

type manager struct {
	cfg            Config
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	loggerProvider *log.LoggerProvider
	prop           propagation.TextMapPropagator
}

func New(ctx context.Context, cfg Config) (Manager, error) {
	tm := &manager{
		cfg: cfg,
	}

	// Set up propagator.
	tm.newPropagator()

	// Set up trace provider.
	if err := tm.newTraceProvider(ctx); err != nil {
		return nil, err
	}

	// Set up meter provider.
	if err := tm.newMeterProvider(ctx); err != nil {
		return nil, err
	}

	// Set up logger provider.
	if err := tm.newLoggerProvider(ctx); err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(tm.prop)
	otel.SetMeterProvider(tm.meterProvider)
	otel.SetTracerProvider(tm.tracerProvider)
	global.SetLoggerProvider(tm.loggerProvider)

	return tm, nil
}

func (tm *manager) Shutdown(ctx context.Context) error {
	var err error

	if tm.tracerProvider != nil {
		err = errors.Join(err, tm.tracerProvider.Shutdown(ctx))
	}

	if tm.loggerProvider != nil {
		err = errors.Join(err, tm.loggerProvider.Shutdown(ctx))
	}

	if tm.meterProvider != nil {
		err = errors.Join(err, tm.meterProvider.Shutdown(ctx))
	}

	return err
}

func (tm *manager) newPropagator() {
	tm.prop = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func (tm *manager) newTraceProvider(ctx context.Context) error {
	opts := make([]trace.TracerProviderOption, 0)

	if tm.cfg.Trace.Stdout {
		exporter, err := stdouttrace.New()
		if err != nil {
			return err
		}
		opts = append(opts, trace.WithBatcher(exporter))
	}

	if tm.cfg.Trace.OtlpHttp.Endpoint != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(tm.cfg.Trace.OtlpHttp.Endpoint))
		if err != nil {
			return err
		}
		opts = append(opts, trace.WithBatcher(exporter))
	}

	tm.tracerProvider = trace.NewTracerProvider(opts...)
	return nil
}

func (tm *manager) newMeterProvider(ctx context.Context) error {
	// metricExporter, err := stdoutmetric.New()
	// if err != nil {
	// 	return nil, err
	// }
	prometheusExporter, err := prometheus.New(
		prometheus.WithoutScopeInfo(),
		prometheus.WithoutTargetInfo(),
	)
	if err != nil {
		return err
	}

	tm.meterProvider = metric.NewMeterProvider(
		// metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(time.Minute))),
		metric.WithReader(prometheusExporter),
	)
	return nil
}

func (tm *manager) newLoggerProvider(_ context.Context) error {
	logExporter, err := stdoutlog.New()
	if err != nil {
		return err
	}

	tm.loggerProvider = log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)

	return nil
}
