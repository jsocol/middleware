package deadline_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"jsocol.io/middleware/deadline"
)

type testRoundTripper struct {
	req *http.Request
}

func (trt *testRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	trt.req = r
	return nil, errors.New("no response")
}

func TestTransport_PropagatesFromContext(t *testing.T) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	trt := &testRoundTripper{}
	client := &http.Client{
		Transport: trt,
	}
	client = deadline.WrapClient(client)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	_, _ = client.Do(req)

	dl := trt.req.Header.Get(deadline.DefaultHeaderName)
	assert.NotEmpty(t, dl)

	dlTime, err := time.Parse(time.RFC3339Nano, dl)
	assert.NoError(t, err)
	assert.InDelta(t, timeout, time.Until(dlTime), float64(5*time.Millisecond))
}

func TestTransport_WithHeaderName(t *testing.T) {
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	header := "X-Test"
	trt := &testRoundTripper{}
	client := &http.Client{
		Transport: trt,
	}
	client = deadline.WrapClient(client, deadline.WithHeaderName(header))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	_, _ = client.Do(req)

	dl := trt.req.Header.Get(header)
	assert.NotEmpty(t, dl)

	dlTime, err := time.Parse(time.RFC3339Nano, dl)
	assert.NoError(t, err)
	assert.InDelta(t, timeout, time.Until(dlTime), float64(5*time.Millisecond))
}

func TestTransport_WithDefaultTimeout(t *testing.T) {
	trt := &testRoundTripper{}
	client := &http.Client{
		Transport: trt,
	}
	timeout := 3 * time.Second
	client = deadline.WrapClient(client, deadline.WithDefaultTimeout(timeout))
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	_, _ = client.Do(req)

	dl := trt.req.Header.Get(deadline.DefaultHeaderName)
	assert.NotEmpty(t, dl)

	dlTime, err := time.Parse(time.RFC3339Nano, dl)
	assert.NoError(t, err)
	assert.InDelta(t, timeout, time.Until(dlTime), float64(5*time.Millisecond))
}

func TestTransport_WithMaxTimeout(t *testing.T) {
	maxTimeout := 2 * time.Second
	timeout := 5 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	trt := &testRoundTripper{}
	client := &http.Client{
		Transport: trt,
	}
	client = deadline.WrapClient(client, deadline.WithMaxTimeout(maxTimeout))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	_, _ = client.Do(req)

	dl := trt.req.Header.Get(deadline.DefaultHeaderName)
	assert.NotEmpty(t, dl)

	dlTime, err := time.Parse(time.RFC3339Nano, dl)
	assert.NoError(t, err)
	assert.InDelta(t, maxTimeout, time.Until(dlTime), float64(5*time.Millisecond))
}
