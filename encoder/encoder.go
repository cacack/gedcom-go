package encoder

import (
	"fmt"
	"io"
	"strings"

	"github.com/cacack/gedcom-go/v2/gedcom"
)

// Encode writes a GEDCOM document to a writer.
func Encode(w io.Writer, doc *gedcom.Document) error {
	return EncodeWithOptions(w, doc, DefaultOptions())
}

// EncodeWithOptions writes a GEDCOM document with custom options.
func EncodeWithOptions(w io.Writer, doc *gedcom.Document, opts *EncodeOptions) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Write header
	if err := writeHeader(w, doc.Header, opts); err != nil {
		return err
	}

	// Write records
	for _, record := range doc.Records {
		if err := writeRecord(w, record, opts); err != nil {
			return err
		}
	}

	// Write trailer
	if err := writeTrailer(w, opts); err != nil {
		return err
	}

	return nil
}

func writeHeader(w io.Writer, header *gedcom.Header, opts *EncodeOptions) error {
	if _, err := fmt.Fprintf(w, "0 HEAD%s", opts.LineEnding); err != nil {
		return err
	}

	// Use TargetVersion if set, otherwise use header.Version
	version := header.Version
	if opts.TargetVersion != "" {
		version = opts.TargetVersion
	}

	if version != "" {
		if _, err := fmt.Fprintf(w, "1 GEDC%s", opts.LineEnding); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "2 VERS %s%s", version, opts.LineEnding); err != nil {
			return err
		}
	}

	if header.Encoding != "" {
		if _, err := fmt.Fprintf(w, "1 CHAR %s%s", header.Encoding, opts.LineEnding); err != nil {
			return err
		}
	}

	if header.SourceSystem != "" {
		if _, err := fmt.Fprintf(w, "1 SOUR %s%s", header.SourceSystem, opts.LineEnding); err != nil {
			return err
		}
	}

	if header.Language != "" {
		if _, err := fmt.Fprintf(w, "1 LANG %s%s", header.Language, opts.LineEnding); err != nil {
			return err
		}
	}

	return nil
}

func writeRecord(w io.Writer, record *gedcom.Record, opts *EncodeOptions) error {
	// Determine the tags to write and the level-0 line value together:
	// - If record.Tags has content, use those (preserves lossless behavior) and the
	//   stored record.Value.
	// - If record.Tags is empty/nil but Entity is set, convert the entity to tags.
	//   Some records (SNOTE) carry text on the level-0 line; when record.Value is
	//   empty, derive that value and its CONT/CONC continuation from the entity in a
	//   single split so a hand-built SharedNote's text is not lost. NOTE entities are
	//   not derived — an entity-only Note must set record.Value itself.
	tags := record.Tags
	value := record.Value
	if len(tags) == 0 && record.Entity != nil {
		tags = entityToTags(record, opts)
		if value == "" {
			var contTags []*gedcom.Tag
			value, contTags = entityRecordText(record, opts)
			tags = append(contTags, tags...)
		}
	}

	// Write record line
	if record.XRef != "" {
		if value != "" {
			if _, err := fmt.Fprintf(w, "0 %s %s %s%s", record.XRef, record.Type, value, opts.LineEnding); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "0 %s %s%s", record.XRef, record.Type, opts.LineEnding); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Fprintf(w, "0 %s%s", record.Type, opts.LineEnding); err != nil {
			return err
		}
	}

	// Filter out custom tags if PreserveUnknownTags is false
	tags = filterTags(tags, opts.PreserveUnknownTags)

	// Write tags
	for _, tag := range tags {
		if err := writeTag(w, tag, opts); err != nil {
			return err
		}
	}

	return nil
}

func writeTag(w io.Writer, tag *gedcom.Tag, opts *EncodeOptions) error {
	if tag.Value != "" {
		if _, err := fmt.Fprintf(w, "%d %s %s%s", tag.Level, tag.Tag, tag.Value, opts.LineEnding); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "%d %s%s", tag.Level, tag.Tag, opts.LineEnding); err != nil {
			return err
		}
	}
	return nil
}

func writeTrailer(w io.Writer, opts *EncodeOptions) error {
	_, err := fmt.Fprintf(w, "0 TRLR%s", opts.LineEnding)
	return err
}

// isCustomTag returns true if the tag name is a custom/extension tag.
// Custom tags are underscore-prefixed by convention (e.g., _CUSTOM, _UID).
func isCustomTag(tagName string) bool {
	return strings.HasPrefix(tagName, "_")
}

// filterTags returns tags with custom tags filtered out if PreserveUnknownTags is false.
// When a custom tag is filtered, its child tags (higher level) are also removed.
func filterTags(tags []*gedcom.Tag, preserveUnknown bool) []*gedcom.Tag {
	if preserveUnknown {
		return tags
	}

	result := make([]*gedcom.Tag, 0, len(tags))
	skipUntilLevel := -1 // -1 means not skipping

	for _, tag := range tags {
		// If we're skipping and encounter a tag at same or lower level, stop skipping
		if skipUntilLevel >= 0 && tag.Level <= skipUntilLevel {
			skipUntilLevel = -1
		}

		// If still skipping, continue
		if skipUntilLevel >= 0 {
			continue
		}

		// Check if this tag should be skipped
		if isCustomTag(tag.Tag) {
			skipUntilLevel = tag.Level
			continue
		}

		result = append(result, tag)
	}

	return result
}
