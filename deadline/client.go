package deadline

import (
	"net/http"
	"time"
)

var _ http.RoundTripper = &Transport{}

type Transport struct {
	http.RoundTripper

	*config
}

func (t *Transport) RoundTrip(r *http.Request) (*http.Response, error) {
	var deadline time.Time
	now := time.Now()

	if dl, ok := r.Context().Deadline(); ok {
		deadline = dl
	} else if t.defaultTimeout != 0 {
		deadline = now.Add(t.defaultTimeout)
	}

	if !deadline.IsZero() {
		if t.maxTimeout != 0 {
			maxDeadline := now.Add(t.maxTimeout)
			if deadline.After(maxDeadline) {
				deadline = maxDeadline
			}
		}

		r.Header.Add(t.headerName, deadline.Format(time.RFC3339Nano))
	}
	return t.RoundTripper.RoundTrip(r)
}

func WrapClient(c *http.Client, opts ...Option) *http.Client {
	t := &Transport{
		RoundTripper: c.Transport,
		config:       newConfig(),
	}

	for _, o := range opts {
		o(t.config)
	}

	c.Transport = t

	return c
}
