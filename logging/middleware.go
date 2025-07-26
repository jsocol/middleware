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

type ContextExtractor func(context.Context) []slog.Attr

type Leveler func(status int) slog.Level

var defaultLeveler Leveler = func(status int) slog.Level {
	if status >= 500 {
		return slog.LevelError
	}
	return slog.LevelInfo
}

var _ http.Handler = &Middleware{}

type Middleware struct {
	target         http.Handler
	logger         *slog.Logger
	leveler        Leveler
	filteredPaths  map[string]struct{}
	filteredRoutes map[string]struct{}
	extractors     []ContextExtractor
}

func Wrap(h http.Handler, opts ...Option) http.Handler {
	m := &Middleware{
		target:         h,
		filteredPaths:  make(map[string]struct{}),
		filteredRoutes: make(map[string]struct{}),
		leveler:        defaultLeveler,
	}

	for _, o := range opts {
		o(m)
	}

	if m.logger == nil {
		m.logger = slog.Default()
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

type Option func(mw *Middleware)

func WithLogger(l *slog.Logger) Option {
	return func(mw *Middleware) {
		mw.logger = l
	}
}

func WithPathFilter(paths ...string) Option {
	return func(mw *Middleware) {
		for _, path := range paths {
			mw.filteredPaths[path] = struct{}{}
		}
	}
}

func WithRouteFilter(routes ...string) Option {
	return func(mw *Middleware) {
		for _, route := range routes {
			mw.filteredRoutes[route] = struct{}{}
		}
	}
}

func WithContextExtractors(fns ...ContextExtractor) Option {
	return func(mw *Middleware) {
		mw.extractors = append(mw.extractors, fns...)
	}
}

func WithLeveler(fn Leveler) Option {
	return func(mw *Middleware) {
		mw.leveler = fn
	}
}
