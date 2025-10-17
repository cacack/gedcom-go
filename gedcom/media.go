package gedcom

// MediaObject represents a multimedia file (photo, document, video, etc.).
type MediaObject struct {
	// XRef is the cross-reference identifier for this media object
	XRef string

	// FileRef is the file path or URL to the media file
	FileRef string

	// Format is the media format (e.g., "jpeg", "png", "pdf")
	Format string

	// Title is a descriptive title for the media
	Title string

	// Type is the media type (e.g., "photo", "document")
	Type string

	// Notes are references to note records
	Notes []string

	// Tags contains all raw tags for this media object (for unknown/custom tags)
	Tags []*Tag
}
