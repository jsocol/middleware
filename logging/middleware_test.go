package logging_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	logging "jsocol.io/middleware/logging"
)

type testHandler struct {
	group   string
	attrs   []slog.Attr
	records []slog.Record
}

func (t *testHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (t *testHandler) Handle(_ context.Context, rec slog.Record) error {
	t.records = append(t.records, rec)
	return nil
}

func (t *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testHandler{
		group: t.group,
		attrs: append(t.attrs, attrs...),
	}
}

func (t *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		group: name,
		attrs: slices.Clone(t.attrs),
	}
}

var _ slog.Handler = &testHandler{}

func TestMiddleware_Logs(t *testing.T) {
	th := &testHandler{}
	logger := slog.New(th)

	route := "GET /foo/{id}"
	mux := http.NewServeMux()
	mux.HandleFunc(route, http.NotFound)

	rr := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/foo/1234", nil)

	mw := logging.Wrap(mux, logging.WithLogger(logger))
	mw.ServeHTTP(rr, r)

	assert.Len(t, th.records, 1)

	attrs := make(map[string]slog.Attr)
	th.records[0].Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a
		return true
	})
	assert.Equal(t, int64(http.StatusNotFound), attrs["http.status_code"].Value.Int64())
	assert.Equal(t, http.MethodGet, attrs["http.method"].Value.String())
	assert.Equal(t, "/foo/1234", attrs["http.path"].Value.String())
	assert.Equal(t, route, attrs["http.route"].Value.String())
	assert.NotEqual(t, time.Duration(0), attrs["duration"].Value.Duration())
}

func TestMiddleware_WithContextExtractors(t *testing.T) {
	th := &testHandler{}
	logger := slog.New(th)
	mux := http.NewServeMux()

	ctxKey := "key"
	ctxVal := "foo"
	ctx := context.WithValue(context.Background(), ctxKey, ctxVal)

	rr := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

	mw := logging.Wrap(mux, logging.WithLogger(logger), logging.WithContextExtractors(
		func(ctx context.Context) []slog.Attr {
			val := ctx.Value(ctxKey).(string)
			return []slog.Attr{
				slog.String("key", val),
			}
		},
	))

	mw.ServeHTTP(rr, r)

	assert.Len(t, th.records, 1)
	attrs := make(map[string]slog.Attr)
	th.records[0].Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a
		return true
	})
	assert.Equal(t, ctxVal, attrs["key"].Value.String())
}

func TestMiddleware_WithPathFilter(t *testing.T) {
	f := func(filter []string, path string, shouldLog bool) func(*testing.T) {
		return func(t *testing.T) {
			th := &testHandler{}
			logger := slog.New(th)
			mux := http.NewServeMux()

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, path, nil)

			mw := logging.Wrap(mux, logging.WithLogger(logger), logging.WithPathFilter(filter...))

			mw.ServeHTTP(rr, r)

			if shouldLog {
				assert.NotEmpty(t, th.records)
			} else {
				assert.Empty(t, th.records)
			}
		}
	}

	testCases := []struct {
		name      string
		filter    []string
		path      string
		shouldLog bool
	}{
		{
			name:      "ignores by path",
			filter:    []string{"/healthz"},
			path:      "/healthz",
			shouldLog: false,
		},
		{
			name:      "does not ignore by prefix",
			filter:    []string{"/foo"},
			path:      "/foo/bar",
			shouldLog: true,
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, f(tc.filter, tc.path, tc.shouldLog))
	}
}

func TestMiddleware_WithRouteFilter(t *testing.T) {
	f := func(filter []string, path string, shouldLog bool) func(*testing.T) {
		return func(t *testing.T) {
			th := &testHandler{}
			logger := slog.New(th)
			mux := http.NewServeMux()
			for _, f := range filter {
				mux.HandleFunc(f, http.NotFound)
			}

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, path, nil)

			mw := logging.Wrap(mux, logging.WithLogger(logger), logging.WithRouteFilter(filter...))

			mw.ServeHTTP(rr, r)

			if shouldLog {
				assert.NotEmpty(t, th.records)
			} else {
				assert.Empty(t, th.records)
			}
		}
	}

	testCases := []struct {
		name      string
		filter    []string
		path      string
		shouldLog bool
	}{
		{
			name:      "ignores by old style route",
			filter:    []string{"/healthz"},
			path:      "/healthz",
			shouldLog: false,
		},
		{
			name:      "ignores by route with verb",
			filter:    []string{"GET /foo/{thing}"},
			path:      "/foo/bar",
			shouldLog: false,
		},
		{
			name:      "does not ignore non-matching verb",
			filter:    []string{"POST /foo/{thing}"},
			path:      "/foo/bar",
			shouldLog: true,
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, f(tc.filter, tc.path, tc.shouldLog))
	}
}

func TestMiddleware_WithLeveler(t *testing.T) {
	f := func(leveler logging.Leveler, statusCode int, expectedLevel slog.Level) func(*testing.T) {
		return func(t *testing.T) {
			th := &testHandler{}
			logger := slog.New(th)
			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(statusCode)
			})

			rr := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)

			mw := logging.Wrap(mux, logging.WithLogger(logger), logging.WithLeveler(leveler))

			mw.ServeHTTP(rr, r)
		}
	}

	testCases := []struct {
		name      string
		filter    []string
		path      string
		shouldLog bool
	}{
		{
			name:      "ignores by old style route",
			filter:    []string{"/healthz"},
			path:      "/healthz",
			shouldLog: false,
		},
		{
			name:      "ignores by route with verb",
			filter:    []string{"GET /foo/{thing}"},
			path:      "/foo/bar",
			shouldLog: false,
		},
		{
			name:      "does not ignore non-matching verb",
			filter:    []string{"POST /foo/{thing}"},
			path:      "/foo/bar",
			shouldLog: true,
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, f(tc.filter, tc.path, tc.shouldLog))
	}
}
