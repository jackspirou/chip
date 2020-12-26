package scanner

import "time"

// Option takes an option and returns an error.
type Option func(*options) error

type options struct {
	ttl   time.Duration
	trace bool
}

// Trace sets the trace option.
func Trace() Option {
	return func(o *options) error {
		o.trace = true
		return nil
	}
}
