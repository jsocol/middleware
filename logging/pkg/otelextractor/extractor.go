package otelextractor

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Options allows changing the attribute keys used by the extractor.
type Options struct {
	// Group defaults to empty. If it is set to a non-empty value, the
	// attributes will be returned wrapped in a slog.Group with the
	// given name.
	Group string

	// TraceID defaults to "trace_id". If a non-empty value is given, it
	// is used instead.
	TraceID string

	// SpanID defaults to "span_id". If a non-empty value is given, it
	// is used instead.
	SpanID string
}

// New returns a new [jsocol.io/middleware/logging.ContextExtractor]
// that adds information from any active OpenTelemetry SpanContext into
// the server log.
func New(opts *Options) func(context.Context) []slog.Attr {
	if opts == nil {
		opts = &Options{}
	}

	groupName := ""
	if opts.Group != "" {
		groupName = opts.Group
	}

	traceIDName := "trace_id"
	if opts.TraceID != "" {
		traceIDName = opts.TraceID
	}

	spanIDName := "span_id"
	if opts.SpanID != "" {
		spanIDName = opts.SpanID
	}

	return func(ctx context.Context) []slog.Attr {
		sc := trace.SpanContextFromContext(ctx)
		var attrs []slog.Attr

		if sc.IsValid() {
			traceID := slog.String(traceIDName, sc.TraceID().String())
			spanID := slog.String(spanIDName, sc.SpanID().String())
			if groupName != "" {
				attrs = append(attrs, slog.Group(
					groupName,
					traceID,
					spanID,
				))
			} else {
				attrs = append(attrs, traceID, spanID)
			}
		}

		return attrs
	}
}
