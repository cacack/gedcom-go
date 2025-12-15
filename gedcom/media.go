package gedcom

// CropRegion defines a subregion of an image to display (GEDCOM 7.0 CROP).
// Used to specify which portion of an image should be displayed when referenced.
type CropRegion struct {
	// Height is the region height in pixels (default: image height - Top)
	Height int

	// Left is the horizontal position of top-left corner (default 0)
	Left int

	// Top is the vertical position of top-left corner (default 0)
	Top int

	// Width is the region width in pixels (default: image width - Left)
	Width int
}

// MediaFile represents a single file within a multimedia record (GEDCOM 7.0 FILE structure).
// A multimedia object can have multiple files (e.g., high-res and thumbnail versions).
type MediaFile struct {
	// FileRef is the path or URL to the file
	FileRef string

	// Form is the MIME type (required in GEDCOM 7.0, e.g., "image/jpeg", "video/mp4")
	Form string

	// MediaType is the category (MEDI tag): AUDIO, BOOK, CARD, ELECTRONIC, PHOTO, VIDEO, etc.
	MediaType string

	// Title is a descriptive title for this file
	Title string

	// Translations contains alternate versions (transcripts, thumbnails, different formats)
	Translations []*MediaTranslation
}

// MediaLink represents a reference to a multimedia object (GEDCOM 7.0 MULTIMEDIA_LINK).
// Used when entities (individuals, families, events) reference media objects.
type MediaLink struct {
	// Crop is an optional crop region for images
	Crop *CropRegion

	// MediaXRef is the pointer to the OBJE record (e.g., "@O1@")
	MediaXRef string

	// Title is an optional title that overrides the FILE's TITL
	Title string
}

// MediaObject represents a multimedia record (GEDCOM 7.0 MULTIMEDIA_RECORD).
// This is a top-level record (0 @Xn@ OBJE) that stores the actual media files.
type MediaObject struct {
	// ChangeDate is when the record was last modified (CHAN tag)
	ChangeDate *ChangeDate

	// CreationDate is when the record was created (CREA tag, GEDCOM 7.0)
	CreationDate *ChangeDate

	// Files contains 1:M file references (required, at least one)
	Files []*MediaFile

	// Notes are references to note records
	Notes []string

	// RefNumbers are user reference numbers (REFN tag, can have multiple)
	RefNumbers []string

	// Restriction is the access restriction level (RESN tag)
	Restriction string

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Tags contains all raw tags for this media object (for unknown/custom tags)
	Tags []*Tag

	// UIDs are unique identifiers (UID tag, can have multiple in GEDCOM 7.0)
	UIDs []string

	// XRef is the cross-reference identifier for this media object
	XRef string
}

// MediaTranslation represents an alternate version of a file (GEDCOM 7.0 FILE-TRAN).
// Examples: transcripts for audio, thumbnails for images, different format conversions.
type MediaTranslation struct {
	// FileRef is the path or URL to the translation/alternate file
	FileRef string

	// Form is the MIME type of the translation file
	Form string
}
