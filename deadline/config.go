package deadline

import "time"

const DefaultHeaderName = "Deadline"

type config struct {
	headerName     string
	defaultTimeout time.Duration
	maxTimeout     time.Duration
}

func newConfig() *config {
	return &config{
		headerName: DefaultHeaderName,
	}
}

type Option func(*config)

func WithMaxTimeout(t time.Duration) Option {
	return func(c *config) {
		c.maxTimeout = t
	}
}

func WithDefaultTimeout(t time.Duration) Option {
	return func(c *config) {
		c.defaultTimeout = t
	}
}

func WithHeaderName(name string) Option {
	return func(c *config) {
		c.headerName = name
	}
}
