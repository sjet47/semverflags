package semverflags

type options struct {
	ignorePrerelease bool
}

// Option controls Registry behavior.
type Option func(*options)

func applyOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// WithIgnorePrerelease makes Resolve ignore the prerelease part of the input
// version when comparing feature ranges. For example, "1.2.3-rc.1" is treated
// as "1.2.3".
func WithIgnorePrerelease() Option {
	return func(o *options) {
		o.ignorePrerelease = true
	}
}
