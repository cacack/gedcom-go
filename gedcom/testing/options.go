package testing

// Option is a functional option for configuring round-trip testing.
type Option func(*roundTripConfig)

// roundTripConfig holds configuration for round-trip testing.
//
// Empty for now: header-tag comparison, the only setting it ever held, became
// unconditional when the encoder stopped discarding the header (issue #429),
// and its opt-in was removed in v3. The type and the Option plumbing stay
// because the variadic opts parameter is part of the AssertRoundTrip and
// CheckRoundTrip signatures.
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
