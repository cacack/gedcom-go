// Package encoder provides functionality to write GEDCOM documents to files.
//
// The encoder package converts structured gedcom.Document objects back into
// the GEDCOM file format. It supports customizable line endings and ensures
// proper GEDCOM structure is maintained.
//
// Example usage:
//
//	doc := &gedcom.Document{
//	    Header: &gedcom.Header{Version: "5.5", Encoding: "UTF-8"},
//	    Records: records,
//	}
//
//	f, err := os.Create("output.ged")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer f.Close()
//
//	if err := encoder.Encode(f, doc); err != nil {
//	    log.Fatal(err)
//	}
package encoder

import (
	"fmt"
	"io"

	"github.com/cacack/gedcom-go/gedcom"
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

	if header.Version != "" {
		if _, err := fmt.Fprintf(w, "1 GEDC%s", opts.LineEnding); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "2 VERS %s%s", header.Version, opts.LineEnding); err != nil {
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
	// Write record line
	if record.XRef != "" {
		if _, err := fmt.Fprintf(w, "0 %s %s%s", record.XRef, record.Type, opts.LineEnding); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "0 %s%s", record.Type, opts.LineEnding); err != nil {
			return err
		}
	}

	// Determine which tags to write:
	// - If record.Tags has content, use those (preserves lossless behavior)
	// - If record.Tags is empty/nil but Entity is set, convert entity to tags
	tags := record.Tags
	if len(tags) == 0 && record.Entity != nil {
		tags = entityToTags(record, opts)
	}

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
