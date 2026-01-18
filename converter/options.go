package converter

// ConvertOptions configures the conversion behavior.
type ConvertOptions struct {
	// Validate runs validation on the converted document.
	// Default: true
	Validate bool

	// StrictDataLoss fails the conversion if any data would be lost.
	// Default: false
	StrictDataLoss bool

	// PreserveUnknownTags keeps vendor extensions and unknown tags.
	// Default: true
	PreserveUnknownTags bool
}

// DefaultOptions returns the default conversion options.
func DefaultOptions() *ConvertOptions {
	return &ConvertOptions{
		Validate:            true,
		StrictDataLoss:      false,
		PreserveUnknownTags: true,
	}
}
