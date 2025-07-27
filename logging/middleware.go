// Package logging implements structured access logging for [http.Handler]
// servers. The logs are written using a [slog.Logger] and so can be formatted
// by a [slog.Handler]. Additional information can be added to logs via
// [ContextExtractor] functions.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var _ http.ResponseWriter = &wrappedWriter{}

type wrappedWriter struct {
	http.ResponseWriter
	status int
}

func (w *wrappedWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *wrappedWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

// ContextExtractor functions are used to pull additional attributes out of a
// [context.Context] instance. See
// [jsocol.io/middleware/logging/pkg/otelextractor.New] for an example.
type ContextExtractor func(context.Context) []slog.Attr

// A Leveler deteremines slog.Level based on an HTTP status code. The default
// Leveler returns [slog.LevelInfo] for all statuses below 500, and
// [slog.LevelError] for statuses >= 500.
type Leveler func(status int) slog.Level

var defaultLeveler Leveler = func(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	return slog.LevelInfo
}

var _ http.Handler = &Middleware{}

// Middleware is an [http.Handler] that records access logs for every request
// handled by the wrapped [http.Handler]. See the [Option] functions for more
// configuration options.
type Middleware struct {
	target         http.Handler
	logger         *slog.Logger
	leveler        Leveler
	filteredPaths  map[string]struct{}
	filteredRoutes map[string]struct{}
	extractors     []ContextExtractor
}

// Wrap returns a new [http.Handler] that is wrapped in a loggin [Middleware]
// struct and will record access logs automatically. If Wrap is given an
// [*http.ServeMux], it will attempt to extract the matching route as well as
// the request path.
func Wrap(h http.Handler, opts ...Option) http.Handler {
	m := &Middleware{
		target:         h,
		logger:         slog.Default(),
		filteredPaths:  make(map[string]struct{}),
		filteredRoutes: make(map[string]struct{}),
	}

	for _, o := range opts {
		o(m)
	}

	if m.leveler == nil {
		m.leveler = defaultLeveler
	}

	return m
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ww := &wrappedWriter{
		ResponseWriter: w,
	}
	start := time.Now()
	var route string

	defer func() {
		if m.filterPath(r.URL.Path) || (route != "" && m.filterRoute(route)) {
			return
		}

		ctx := r.Context()
		attrs := []slog.Attr{
			slog.Int("http.status_code", ww.status),
			slog.String("http.path", r.URL.Path),
			slog.String("http.method", r.Method),
			slog.Any("duration", time.Since(start)),
		}

		if route != "" {
			attrs = append(attrs, slog.String("http.route", route))
		}

		for _, fn := range m.extractors {
			attrs = append(attrs, fn(ctx)...)
		}

		m.logger.LogAttrs(
			ctx,
			m.leveler(ww.status),
			fmt.Sprintf("%s %s [%d]", r.Method, r.URL.Path, ww.status),
			attrs...,
		)
	}()

	if h, ok := m.target.(*http.ServeMux); ok {
		handler, pattern := h.Handler(r)
		route = pattern
		handler.ServeHTTP(ww, r)
	} else {
		m.target.ServeHTTP(ww, r)
	}
}

func (m *Middleware) filterPath(path string) bool {
	_, ok := m.filteredPaths[path]
	return ok
}

func (m *Middleware) filterRoute(route string) bool {
	_, ok := m.filteredRoutes[route]
	return ok
}

// Options configure a [Middleware] instance.
type Option func(mw *Middleware)

// WithLogger specifies a particular [*slog.Logger] for the [Middleware] to
// use. Otherwise, [slog.Default] is used.
func WithLogger(l *slog.Logger) Option {
	return func(mw *Middleware) {
		mw.logger = l
	}
}

// WithPathFilter excludes certain paths from access logging, e.g. to avoid
// logging internal health checks or favicon requests.
func WithPathFilter(paths ...string) Option {
	return func(mw *Middleware) {
		for _, path := range paths {
			mw.filteredPaths[path] = struct{}{}
		}
	}
}

// WithRouteFilter excludes certain route patterns from an [http.ServeMux] from
// access logging. Uses [http.ServeMux.Handler] to determine the pattern, so
// the ignored routes should match those patterns.
func WithRouteFilter(routes ...string) Option {
	return func(mw *Middleware) {
		for _, route := range routes {
			mw.filteredRoutes[route] = struct{}{}
		}
	}
}

// WithContextExtractors adds [ContextExtractor] functions that attempt to
// gather additional information from the [http.Request.Context] to add to logs
// as [slog.Attr].
func WithContextExtractors(fns ...ContextExtractor) Option {
	return func(mw *Middleware) {
		mw.extractors = append(mw.extractors, fns...)
	}
}

// WithLeveler specifies an alternative [Leveler].
func WithLeveler(fn Leveler) Option {
	return func(mw *Middleware) {
		mw.leveler = fn
	}
}
