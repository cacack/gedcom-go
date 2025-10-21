package encoder

// EncodeOptions provides configuration for encoding GEDCOM files.
type EncodeOptions struct {
	// LineEnding specifies the line ending to use ("\r\n" or "\n")
	LineEnding string
}

// DefaultOptions returns the default encoding options.
func DefaultOptions() *EncodeOptions {
	return &EncodeOptions{
		LineEnding: "\n",
	}
}
