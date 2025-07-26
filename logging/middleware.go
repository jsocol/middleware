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

type Middleware struct {
	target         *http.ServeMux
	logger         *slog.Logger
	filteredRoutes map[string]struct{}
	extractors     []ContextExtractor
}

func Wrap(h *http.ServeMux, opts ...Option) http.Handler {
	m := &Middleware{
		target:         h,
		filteredRoutes: make(map[string]struct{}),
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
	defer func() {
		_, route := m.target.Handler(r)
		if m.filterRoute(route) {
			return
		}

		ctx := r.Context()
		attrs := []slog.Attr{
			slog.Int("http.status_code", ww.status),
			slog.String("http.route", route),
			slog.String("http.path", r.URL.Path),
			slog.String("http.method", r.Method),
			slog.Any("duration", time.Since(start)),
		}

		for _, fn := range m.extractors {
			attrs = append(attrs, fn(ctx)...)
		}

		m.logger.LogAttrs(
			ctx,
			slog.LevelInfo,
			fmt.Sprintf("%s %s [%d]", r.Method, r.URL.Path, ww.status),
			attrs...,
		)
	}()
	m.target.ServeHTTP(ww, r)
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
