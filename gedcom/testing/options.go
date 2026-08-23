package testing

// Option is a functional option for configuring round-trip testing.
type Option func(*roundTripConfig)

// roundTripConfig holds configuration for round-trip testing.
//
// Empty for now: header-tag comparison, the only setting it ever held, is
// unconditional since the encoder stopped discarding the header. The type and
// the Option plumbing stay because the variadic opts parameter is part of the
// AssertRoundTrip and CheckRoundTrip signatures.
type roundTripConfig struct{}

// defaultConfig returns the default configuration.
func defaultConfig() *roundTripConfig {
	return &roundTripConfig{}
}

// applyOptions applies functional options to the config.
func applyOptions(opts ...Option) *roundTripConfig {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithHeaderTagComparison is a no-op retained for compatibility.
//
// Header tags are now compared on every round-trip. The option existed because
// the encoder rebuilt HEAD from four scalar fields and discarded everything
// else, so comparing them failed nearly every fixture; issue #429 fixed the
// encoder, and the comparison no longer has anything to opt into.
//
// Deprecated: header tags are always compared. Remove the call; it does
// nothing. This function will be removed in the next major version.
func WithHeaderTagComparison() Option {
	return func(*roundTripConfig) {}
}
