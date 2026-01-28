package testing

// Option is a functional option for configuring round-trip testing.
type Option func(*roundTripConfig)

// roundTripConfig holds configuration for round-trip testing.
type roundTripConfig struct {
	// compareHeaderTags enables comparison of Header.Tags
	// By default, header tags are not compared because the encoder
	// reconstructs the header from Header fields.
	compareHeaderTags bool
}

// defaultConfig returns the default configuration.
func defaultConfig() *roundTripConfig {
	return &roundTripConfig{
		compareHeaderTags: false,
	}
}

// applyOptions applies functional options to the config.
func applyOptions(opts ...Option) *roundTripConfig {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithHeaderTagComparison enables comparison of Header.Tags.
// By default, header tags are not compared because the encoder
// reconstructs the header structure from Header fields, which may
// produce different tags than the original.
func WithHeaderTagComparison() Option {
	return func(cfg *roundTripConfig) {
		cfg.compareHeaderTags = true
	}
}
