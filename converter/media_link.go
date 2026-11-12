package converter

import (
	"slices"
	"strconv"
	"strings"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// promoteInlineMediaLinks turns every inline multimedia link -- an OBJE
// substructure with no pointer, carrying its file in place -- into an OBJE
// record and a pointer to it on a 5.5/5.5.1 -> 7.0 upgrade. GEDCOM 7.0 has
// only the pointer form; the migration guide (https://gedcom.io/migrate)
// gives this conversion:
//
//	1 OBJE                     1 OBJE @O1@
//	2 FILE photo.jpg     ->    ...
//	3 FORM jpg                 0 @O1@ OBJE
//	2 TITL A photo             1 FILE photo.jpg
//	                           2 FORM jpg
//	                           2 TITL A photo
//
// The link's subordinates move into the record one level up. 7.0 titles a
// file rather than the object, so a link-level TITL goes under the first
// FILE, and a link-level FORM (the 5.5 layout, FORM beside FILE) under it
// too; everything else (NOTE, extensions) stays at record level. A link with
// no FILE has nothing a 7.0 OBJE record could hold ({1:M} FILE) and is left
// as it is. The new record's XRef is the first free @O<n>@. Each promotion is
// a Normalized note, with one aggregate transformation.
//
// It runs after normalizeXRefsForV7, so the XRefs it checks for collisions
// are final, and before transformMediaTypes and the enumeration translation,
// so the new records' FORM and MEDI are translated with every other record's.
func promoteInlineMediaLinks(doc *gedcom.Document, report *gedcom.ConversionReport) {
	used := xrefsInUse(doc)
	next := 1
	var added []*gedcom.Record
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		for i := 0; i < len(record.Tags); i++ {
			tag := record.Tags[i]
			if tag == nil || tag.Tag != "OBJE" || tag.Level < 1 || strings.TrimSpace(tag.Value) != "" {
				continue
			}
			end := blockEnd(record.Tags, i)
			tags := mediaRecordTags(record.Tags[i+1:end], tag.Level)
			if tags == nil {
				continue
			}
			var xref string
			xref, next = freeXRef(used, "O", next)
			used[xref] = true
			promoted := &gedcom.Record{XRef: xref, Type: gedcom.RecordTypeMedia, Tags: tags}
			added = append(added, promoted)
			if doc.XRefMap == nil {
				doc.XRefMap = map[string]*gedcom.Record{}
			}
			doc.XRefMap[xref] = promoted

			tag.Value = xref
			record.Tags = slices.Delete(record.Tags, i+1, end)
			report.AddNormalized(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, tagPath(record.Tags, i)...),
				Original: "OBJE (inline multimedia link)",
				Result:   "OBJE " + xref,
				Reason:   "GEDCOM 7.0 has no inline multimedia link; its file moved to a new OBJE record",
			})
		}
	}
	if len(added) == 0 {
		return
	}
	doc.Records = append(doc.Records, added...)
	report.AddTransformation(gedcom.Transformation{
		Type:        "INLINE_OBJE_TO_RECORD",
		Description: "Converted inline multimedia links to OBJE records",
		Count:       len(added),
	})
}

// mediaRecordTags returns the level-1-rooted tags of the OBJE record that
// holds the inline link block (the link's subordinates, the link at level
// linkLevel), or nil when the block has no FILE.
func mediaRecordTags(block []*gedcom.Tag, linkLevel int) []*gedcom.Tag {
	var files, moved, rest []*gedcom.Tag // FILE blocks; TITL/FORM blocks to put under the first FILE; the rest
	for i := 0; i < len(block); i++ {
		tag := block[i]
		if tag == nil {
			continue
		}
		end := blockEnd(block, i)
		sub := relevel(block[i:end], linkLevel)
		switch {
		case tag.Level != linkLevel+1:
			rest = append(rest, sub...)
		case tag.Tag == "FILE":
			files = append(files, sub...)
		case tag.Tag == "TITL" || tag.Tag == "FORM":
			for _, t := range sub {
				t.Level++
			}
			moved = append(moved, sub...)
		default:
			rest = append(rest, sub...)
		}
		i = end - 1
	}
	if len(files) == 0 {
		return nil
	}
	firstEnd := 1
	for firstEnd < len(files) && files[firstEnd].Level > 1 {
		firstEnd++
	}
	out := make([]*gedcom.Tag, 0, len(files)+len(moved)+len(rest))
	out = append(out, files[:firstEnd]...)
	out = append(out, moved...)
	out = append(out, files[firstEnd:]...)
	return append(out, rest...)
}

// relevel returns copies of tags (nil ones dropped) moved up by linkLevel
// levels, so a link's direct subordinates are at level 1.
func relevel(tags []*gedcom.Tag, linkLevel int) []*gedcom.Tag {
	out := make([]*gedcom.Tag, 0, len(tags))
	for _, t := range tags {
		if t == nil {
			continue
		}
		c := *t
		c.Level -= linkLevel
		out = append(out, &c)
	}
	return out
}

// freeXRef returns the first @<prefix><n>@, from n, not in used, and the n to
// continue from.
func freeXRef(used map[string]bool, prefix string, n int) (xref string, next int) {
	for {
		xref = "@" + prefix + strconv.Itoa(n) + "@"
		n++
		if !used[xref] {
			return xref, n
		}
	}
}

// tagPath returns the tags from the record's level 1 down to tags[i].
func tagPath(tags []*gedcom.Tag, i int) []string {
	path := []string{tags[i].Tag}
	level := tags[i].Level
	for j := i - 1; j >= 0 && level > 1; j-- {
		if tags[j] != nil && tags[j].Level < level {
			path = append([]string{tags[j].Tag}, path...)
			level = tags[j].Level
		}
	}
	return path
}

// blockEnd returns the index just past the block tags[i] opens: tags[i] and
// every deeper (or nil) tag after it.
func blockEnd(tags []*gedcom.Tag, i int) int {
	end := i + 1
	for end < len(tags) && (tags[end] == nil || tags[end].Level > tags[i].Level) {
		end++
	}
	return end
}

// xrefsInUse returns every XRef doc's records and XRefMap hold.
func xrefsInUse(doc *gedcom.Document) map[string]bool {
	used := map[string]bool{}
	for xref := range doc.XRefMap {
		used[xref] = true
	}
	for _, r := range doc.Records {
		if r != nil {
			used[r.XRef] = true
		}
	}
	return used
}
