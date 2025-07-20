package deadline_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"jsocol.io/middleware/deadline"
)

func TestMiddleware_Propagates(t *testing.T) {
	ctxDeadline := time.Now().Add(5 * time.Second)
	headerName := "X-Stop-At"
	hasDeadline := false

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if dl, ok := r.Context().Deadline(); ok {
			hasDeadline = true
			assert.Truef(t, ctxDeadline.Equal(dl), "got %v, want %v", dl, ctxDeadline)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add(headerName, ctxDeadline.Format(time.RFC3339Nano))

	wrapped := deadline.Wrap(mux, deadline.WithHeaderName(headerName))

	wrapped.ServeHTTP(w, r)

	assert.True(t, hasDeadline, "request context has deadline")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestMiddleware_WithDefaultTimeout(t *testing.T) {
	hasDeadline := false
	timeout := 3 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if dl, ok := r.Context().Deadline(); ok {
			hasDeadline = true
			to := time.Until(dl)
			assert.InDelta(t, timeout, to, float64(3*time.Millisecond))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	wrapped := deadline.Wrap(mux, deadline.WithDefaultTimeout(timeout))

	wrapped.ServeHTTP(w, r)

	assert.True(t, hasDeadline, "request context has deadline")
}

func TestMiddleware_WithMaxTimeout(t *testing.T) {
	hasDeadline := false
	reqDeadline := time.Now().Add(10 * time.Second)
	maxTimeout := 3 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if dl, ok := r.Context().Deadline(); ok {
			hasDeadline = true
			to := time.Until(dl)
			assert.InDelta(t, maxTimeout, to, float64(3*time.Millisecond))
		}
		w.WriteHeader(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Add(deadline.DefaultHeaderName, reqDeadline.Format(time.RFC3339))

	wrapped := deadline.Wrap(mux, deadline.WithMaxTimeout(maxTimeout))

	wrapped.ServeHTTP(w, r)

	assert.True(t, hasDeadline, "request context has deadline")
}
