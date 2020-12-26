package ast

import "time"

// Option takes an option and returns and error
type Option func(*options) error

type options struct {
	ttl   time.Duration
	trace bool
}
