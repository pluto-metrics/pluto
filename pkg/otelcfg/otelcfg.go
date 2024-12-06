package otelcfg

import (
	"context"
	"errors"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

type ConfigTraceOtlpHttp struct {
	Endpoint string            `yaml:"endpoint" validate:"omitempty,uri" default:""`
	Headers  map[string]string `yaml:"headers"`
}

type ConfigLogOtlpHttp struct {
	Endpoint string            `yaml:"endpoint" validate:"omitempty,uri" default:""`
	Headers  map[string]string `yaml:"headers"`
}

type ConfigTrace struct {
	Stdout   bool                `yaml:"stdout" default:"false"`
	OtlpHttp ConfigTraceOtlpHttp `yaml:"otlp_http"`
}

type ConfigLog struct {
	Stdout   bool              `yaml:"stdout" default:"false"`
	OtlpHttp ConfigLogOtlpHttp `yaml:"otlp_http"`
}

type Config struct {
	Trace ConfigTrace `yaml:"trace"`
	Log   ConfigLog   `yaml:"log"`
}

type Manager interface {
	Shutdown(ctx context.Context) error
}

type manager struct {
	cfg            Config
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	loggerProvider *log.LoggerProvider
	otelZapLogger  *otelzap.Logger
	zapLogger      *zap.Logger
	prop           propagation.TextMapPropagator
}

func New(ctx context.Context, cfg Config, zapCfg zap.Config) (Manager, error) {
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

	tm.zapLogger = zap.Must(zapCfg.Build())
	tm.otelZapLogger = otelzap.New(tm.zapLogger)
	otelzap.ReplaceGlobals(tm.otelZapLogger)

	defer zap.RedirectStdLog(tm.zapLogger)()
	defer zap.ReplaceGlobals(tm.zapLogger)()
	return tm, nil
}

func (tm *manager) Shutdown(ctx context.Context) error {
	var err error

	if tm.zapLogger != nil {
		tm.zapLogger.Sync()
	}

	if tm.otelZapLogger != nil {
		tm.otelZapLogger.Sync()
	}

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
		opts = append(opts, trace.WithBatcher(exporter, trace.WithBatchTimeout(time.Second)))
	}

	if tm.cfg.Trace.OtlpHttp.Endpoint != "" {
		exporter, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(tm.cfg.Trace.OtlpHttp.Endpoint),
			otlptracehttp.WithHeaders(tm.cfg.Trace.OtlpHttp.Headers),
		)
		if err != nil {
			return err
		}
		opts = append(opts, trace.WithBatcher(exporter))
	}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("pluto"),
		),
	)

	if err != nil {
		return err
	}

	opts = append(opts, trace.WithResource(r))

	tm.tracerProvider = trace.NewTracerProvider(opts...)
	return nil
}

func (tm *manager) newMeterProvider(_ context.Context) error {
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

func (tm *manager) newLoggerProvider(ctx context.Context) error {
	opts := make([]log.LoggerProviderOption, 0)

	if tm.cfg.Log.Stdout {
		exporter, err := stdoutlog.New()
		if err != nil {
			return err
		}
		opts = append(opts, log.WithProcessor(log.NewBatchProcessor(exporter)))
	}

	if tm.cfg.Log.OtlpHttp.Endpoint != "" {
		exporter, err := otlploghttp.New(ctx,
			otlploghttp.WithEndpointURL(tm.cfg.Log.OtlpHttp.Endpoint),
			otlploghttp.WithHeaders(tm.cfg.Log.OtlpHttp.Headers),
		)
		if err != nil {
			return err
		}
		opts = append(opts, log.WithProcessor(log.NewBatchProcessor(exporter)))
	}

	tm.loggerProvider = log.NewLoggerProvider(opts...)

	return nil
}
