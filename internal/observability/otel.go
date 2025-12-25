package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Config struct {
	Enabled         bool
	ServiceName     string
	ServiceVersion  string
	Environment     string
	OTLPEndpoint    string // full URL or host:port
	OTLPHeaders     map[string]string
	Insecure        bool
	SampleRatio     float64 // 0..1
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
}

func SetupFromEnv(ctx context.Context, serviceName, serviceVersion string) (shutdown func(context.Context) error, _ error) {
	cfg, err := configFromEnv(serviceName, serviceVersion)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}
	return Setup(ctx, cfg)
}

func Setup(parent context.Context, cfg Config) (shutdown func(context.Context) error, _ error) {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "webhookd"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "dev"
	}
	if cfg.StartupTimeout <= 0 {
		cfg.StartupTimeout = 5 * time.Second
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
	if cfg.SampleRatio <= 0 {
		cfg.SampleRatio = 1
	}
	if cfg.SampleRatio > 1 {
		cfg.SampleRatio = 1
	}

	if cfg.OTLPEndpoint == "" {
		// Safe local default if the user explicitly enables OTel.
		cfg.OTLPEndpoint = "http://localhost:4318"
	}
	endpointURL := normalizeOTLPEndpoint(cfg.OTLPEndpoint, cfg.Insecure)

	setupCtx, cancel := context.WithTimeout(context.Background(), cfg.StartupTimeout)
	defer cancel()

	exp, err := otlptracehttp.New(setupCtx,
		otlptracehttp.WithEndpointURL(endpointURL),
		otlptracehttp.WithHeaders(cfg.OTLPHeaders),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	res, err := resource.New(parent,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.ServiceVersion),
			attribute.String("deployment.environment.name", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		if ctx == nil {
			ctx = context.Background()
		}
		shutdownCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
		defer cancel()
		return tp.Shutdown(shutdownCtx)
	}, nil
}

func configFromEnv(serviceName, serviceVersion string) (Config, error) {
	enabled, err := getenvBool("WEBHOOKD_OTEL_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	// If the user set standard OTEL endpoints, implicitly enable.
	otelEndpoint := firstNonEmpty(
		os.Getenv("WEBHOOKD_OTEL_EXPORTER_OTLP_ENDPOINT"),
		os.Getenv("WEBHOOKD_OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"),
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"),
		os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	)
	if !enabled && otelEndpoint != "" {
		enabled = true
	}

	insecure, err := getenvBool("WEBHOOKD_OTEL_EXPORTER_OTLP_INSECURE", false)
	if err != nil {
		return Config{}, err
	}
	if !insecure {
		// Standard env name.
		insecure, _ = getenvBool("OTEL_EXPORTER_OTLP_INSECURE", false)
	}

	sampleRatio, err := getenvFloat("WEBHOOKD_OTEL_TRACES_SAMPLER_RATIO", 1)
	if err != nil {
		return Config{}, err
	}

	headers, err := parseHeaders(firstNonEmpty(
		os.Getenv("WEBHOOKD_OTEL_EXPORTER_OTLP_HEADERS"),
		os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"),
	))
	if err != nil {
		return Config{}, err
	}

	env := firstNonEmpty(os.Getenv("WEBHOOKD_ENV"), os.Getenv("ENV"), os.Getenv("OTEL_ENVIRONMENT"))

	return Config{
		Enabled:        enabled,
		ServiceName:    firstNonEmpty(os.Getenv("WEBHOOKD_OTEL_SERVICE_NAME"), os.Getenv("OTEL_SERVICE_NAME"), serviceName),
		ServiceVersion: serviceVersion,
		Environment:    env,
		OTLPEndpoint:   otelEndpoint,
		OTLPHeaders:    headers,
		Insecure:       insecure,
		SampleRatio:    sampleRatio,
	}, nil
}

func normalizeOTLPEndpoint(v string, insecure bool) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	if insecure {
		return "http://" + v
	}
	return "https://" + v
}

func parseHeaders(v string) (map[string]string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	parts := strings.Split(v, ",")
	out := make(map[string]string, 0)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		k, val, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header %q (expected key=value)", p)
		}
		k = strings.TrimSpace(k)
		val = strings.TrimSpace(val)
		if k == "" {
			return nil, errors.New("invalid header with empty key")
		}
		out[k] = val
	}
	return out, nil
}

func getenvBool(name string, def bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return b, nil
}

func getenvFloat(name string, def float64) (float64, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return f, nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
