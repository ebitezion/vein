package main

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdouttrace "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func setupTracing(cfg config) (oteltrace.Tracer, func(context.Context) error, error) {
	if !cfg.tracing.enabled {
		provider := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(provider)
		return provider.Tracer(cfg.appName), provider.Shutdown, nil
	}

	ctx := context.Background()
	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	if cfg.tracing.otlpEndpoint != "" {
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.tracing.otlpEndpoint),
			otlptracegrpc.WithInsecure(),
		)
	} else {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.tracing.sampleRatio))
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)

	otel.SetTracerProvider(provider)
	return provider.Tracer(cfg.appName), provider.Shutdown, nil
}

func (app *application) tracingMiddleware(next http.Handler) http.Handler {
	if app.tracer == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := app.tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
