package parser

// Option configures a Parser.
type Option func(*options)

type options struct {
	trace bool
}

// Trace enables token tracing to standard error while parsing.
func Trace() Option {
	return func(o *options) { o.trace = true }
}
