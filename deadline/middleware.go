package deadline

import (
	"context"
	"net/http"
	"time"
)

var _ http.Handler = &Middleware{}

type Middleware struct {
	*config

	target http.Handler
}

func Wrap(target http.Handler, opts ...Option) http.Handler {
	mw := &Middleware{
		config: newConfig(),
		target: target,
	}

	for _, o := range opts {
		o(mw.config)
	}

	return mw
}

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		var deadline time.Time
		now := time.Now()

		if incomingDeadline := r.Header.Get(m.headerName); incomingDeadline != "" {
			if dl, err := time.Parse(time.RFC3339Nano, incomingDeadline); err == nil {
				deadline = dl
			}
		}

		if deadline.IsZero() && m.defaultTimeout != 0 {
			deadline = now.Add(m.defaultTimeout)
		}

		if !deadline.IsZero() {
			if m.maxTimeout != 0 {
				maxDeadline := now.Add(m.maxTimeout)
				if deadline.After(maxDeadline) {
					deadline = maxDeadline
				}
			}
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()

			r = r.WithContext(ctx)
		}
	}
	m.target.ServeHTTP(w, r)
}
