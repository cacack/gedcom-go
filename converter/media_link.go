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
// as it is, and so is an OBJE in an extension record or beneath an extension
// tag (#840): the specification does not define an extension's substructures,
// so they are not links to promote. The new record's XRef is the first free @O<n>@. Each promotion is
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
		if record == nil || isExtension(string(record.Type)) {
			continue
		}
		for i := 0; i < len(record.Tags); i++ {
			tag := record.Tags[i]
			if tag != nil && isExtension(tag.Tag) {
				i = blockEnd(record.Tags, i) - 1
				continue
			}
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
		sub := shiftLevels(block[i:end], -linkLevel) // the link's subordinates to level 1
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

// isExtension reports whether a tag or record type is an extension (its name
// starts with "_"). The converter does not reinterpret what is beneath one:
// the specification does not define an extension's substructures.
func isExtension(tag string) bool {
	return strings.HasPrefix(tag, "_")
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

// xrefsInUse returns every XRef doc's records and XRefMap hold, and every
// XRef a pointer in the header or a record names, so a new record never
// takes the XRef of a dangling pointer and silently becomes its target.
func xrefsInUse(doc *gedcom.Document) map[string]bool {
	used := map[string]bool{}
	addPointers := func(tags []*gedcom.Tag) {
		for _, t := range tags {
			if t == nil {
				continue
			}
			if v := strings.TrimSpace(t.Value); doc.IsPointerXRef(v) {
				used[v] = true
			}
		}
	}
	for xref := range doc.XRefMap {
		used[xref] = true
	}
	if doc.Header != nil {
		addPointers(doc.Header.Tags)
	}
	for _, r := range doc.Records {
		if r != nil {
			used[r.XRef] = true
			addPointers(r.Tags)
		}
	}
	return used
}

// demoteMediaRecords replaces each OBJE record that holds a file with inline
// multimedia links on a 7.0 -> 5.5 downgrade, the reverse of
// promoteInlineMediaLinks. A 5.5 OBJE record has no FILE -- it embeds its
// object as a BLOB -- so a file reference has a 5.5 form only in an inline
// link:
//
//	1 OBJE @O1@                1 OBJE
//	2 TITL Portrait      ->    2 FORM image/jpeg
//	0 @O1@ OBJE                2 TITL Portrait
//	1 FILE photo.jpg           2 FILE photo.jpg
//	2 FORM image/jpeg          2 NOTE A note
//	1 NOTE A note
//
// (transformMediaTypes, which runs after, then maps the links' FORM values
// to their 5.5 form, e.g. image/jpeg -> JPG.)
//
// Every pointer to the record becomes one inline link per FILE, holding that
// file's FORM, TITL (the pointer's own TITL in preference) and the record's
// NOTEs; whatever else the pointer carried stays on the first link. The
// record is then removed. What an inline link cannot hold -- the record's
// other structures (REFN, RIN, CHAN, extensions) and a FILE's other
// substructures (FORM.MEDI, TRAN) -- is reported as Dropped and DataLoss, and
// so is a record no structure points to, whose files have nowhere to go. A
// record with no FILE, or no Tags, is left alone, and so is one a record
// with no Tags points to (written from its entity, whose pointer would
// dangle) or whose links would take the copies past maxInlineMediaTags.
//
// It runs after dropForeignStructures, which keeps the record's FILE for it
// (foreign70To55.exempt) and has already removed the 7.0-only parts of the
// rest, so the NOTEs it copies hold only 5.5 structures.
func demoteMediaRecords(doc *gedcom.Document, report *gedcom.ConversionReport) {
	records := map[string]*mediaRecordParts{}
	for _, record := range doc.Records {
		if record != nil && record.Type == gedcom.RecordTypeMedia {
			if parts := splitMediaRecord(record.Tags); parts != nil {
				records[record.XRef] = parts
			}
		}
	}
	if len(records) == 0 {
		return
	}
	keepTypedPointerMediaRecords(doc, records, report)
	keepOversizedMediaRecords(doc, records, report)
	if len(records) == 0 {
		return
	}
	linked, links := inlineMediaPointers(doc, records, report)
	removeDemotedMediaRecords(doc, records, linked, report)
	report.AddTransformation(gedcom.Transformation{
		Type:        "OBJE_RECORD_TO_INLINE",
		Description: "Converted OBJE record pointers to inline multimedia links",
		Count:       links,
	})
}

// keepTypedPointerMediaRecords removes from records each record a record
// with no Tags points to: that record is written from its entity, which
// inlineMediaPointers does not rewrite, so removing the target would leave
// its pointer dangling. Each kept record is a Preserved note and `OBJE records
// with typed pointers` data loss, since its FILE is not valid 5.5.
func keepTypedPointerMediaRecords(doc *gedcom.Document, records map[string]*mediaRecordParts, report *gedcom.ConversionReport) {
	keep := map[string]bool{}
	for _, record := range doc.Records {
		if record != nil && len(record.Tags) == 0 {
			gedcom.Visit(record, func(xref string) {
				if records[xref] != nil {
					keep[xref] = true
				}
			})
		}
	}
	const reason = "A record with no Tags points to this OBJE record, so it and its FILE are kept, which is not valid GEDCOM 5.5"
	keepMediaRecords(doc, records, keep, "OBJE records with typed pointers", reason, report)
}

// keepMediaRecords removes the records in keep from records, in document
// order, reporting each as a Preserved note and all as one data loss item.
func keepMediaRecords(doc *gedcom.Document, records map[string]*mediaRecordParts, keep map[string]bool, feature, reason string, report *gedcom.ConversionReport) {
	var kept []string
	for _, record := range doc.Records {
		if record == nil || records[record.XRef] == nil || !keep[record.XRef] {
			continue
		}
		delete(records, record.XRef)
		kept = append(kept, record.XRef)
		report.AddPreserved(gedcom.ConversionNote{
			Path:     BuildRecordPath(string(record.Type), record.XRef),
			Original: "OBJE",
			Result:   "OBJE",
			Reason:   reason,
		})
	}
	if len(kept) > 0 {
		report.AddDataLoss(gedcom.DataLossItem{Feature: feature, Reason: reason, AffectedRecords: kept})
	}
}

// maxInlineMediaTags caps the tags demoteMediaRecords copies into inline
// links across a document. Each pointer gets a link per FILE, each with the
// record's NOTEs, so a hostile file can ask for pointers * files * notes tags
// from a few lines of each; a real one copies a few per pointer. Past the
// cap a record is kept as it is rather than demoted (in the spirit of ADR
// 0009's bounded work).
const maxInlineMediaTags = 1 << 20

// keepOversizedMediaRecords removes from records, in document order, each
// record whose inline links would take the copies past maxInlineMediaTags,
// leaving it and its pointers in place. Each is a Preserved note and
// `Oversized OBJE records` data loss, since its FILE is not valid 5.5.
func keepOversizedMediaRecords(doc *gedcom.Document, records map[string]*mediaRecordParts, report *gedcom.ConversionReport) {
	cost := map[string]int{}
	for _, record := range doc.Records {
		if record == nil || records[record.XRef] != nil {
			continue
		}
		for i, tag := range record.Tags {
			if tag == nil || tag.Tag != "OBJE" || tag.Level < 1 {
				continue
			}
			xref := strings.TrimSpace(tag.Value)
			if parts := records[xref]; parts != nil {
				// An upper bound: the pointer's subordinates are copied at
				// most once per file.
				pointerSubs := blockEnd(record.Tags, i) - i - 1
				cost[xref] = min(cost[xref]+parts.linkTags()+len(parts.files)*pointerSubs, maxInlineMediaTags+1)
			}
		}
	}

	keep := map[string]bool{}
	total := 0
	for _, record := range doc.Records {
		if record == nil || records[record.XRef] == nil {
			continue
		}
		if total+cost[record.XRef] <= maxInlineMediaTags {
			total += cost[record.XRef]
		} else {
			keep[record.XRef] = true
		}
	}
	const reason = "Its inline links would exceed the conversion's size bound; the OBJE record and its FILE are kept, which is not valid GEDCOM 5.5"
	keepMediaRecords(doc, records, keep, "Oversized OBJE records", reason, report)
}

// inlineMediaPointers replaces each pointer to one of records with its inline
// links, returning the records pointed to and the number of pointers replaced.
func inlineMediaPointers(doc *gedcom.Document, records map[string]*mediaRecordParts, report *gedcom.ConversionReport) (linked map[string]bool, links int) {
	linked = map[string]bool{}
	for _, record := range doc.Records {
		if record == nil || records[record.XRef] != nil {
			continue
		}
		for i := 0; i < len(record.Tags); i++ {
			tag := record.Tags[i]
			if tag == nil || tag.Tag != "OBJE" || tag.Level < 1 {
				continue
			}
			xref := strings.TrimSpace(tag.Value)
			parts := records[xref]
			if parts == nil {
				continue
			}
			path := BuildNestedPath(string(record.Type), record.XRef, tagPath(record.Tags, i)...)
			end := blockEnd(record.Tags, i)
			inline := parts.inlineLinks(tag.Level, record.Tags[i+1:end])
			record.Tags = slices.Replace(record.Tags, i, end, inline...)
			i += len(inline) - 1
			linked[xref] = true
			links++
			report.AddNormalized(gedcom.ConversionNote{
				Path:     path,
				Original: "OBJE " + xref,
				Result:   "OBJE (inline multimedia link)",
				Reason:   "A GEDCOM 5.5 OBJE record cannot hold a file; the record's files moved to inline links",
			})
		}
	}
	return linked, links
}

// removeDemotedMediaRecords removes records from doc, reporting what of each
// linked one no inline link holds, and each unlinked one whole.
func removeDemotedMediaRecords(doc *gedcom.Document, records map[string]*mediaRecordParts, linked map[string]bool, report *gedcom.ConversionReport) {
	const reason = "GEDCOM 5.5 has no OBJE record that holds a file, and an inline multimedia link cannot hold this"
	const unlinkedReason = "No structure points to this OBJE record, and GEDCOM 5.5 has no OBJE record that holds a file"
	var lost, unlinked []string
	kept := doc.Records[:0]
	for _, record := range doc.Records {
		if record == nil || records[record.XRef] == nil {
			kept = append(kept, record)
			continue
		}
		delete(doc.XRefMap, record.XRef)
		if !linked[record.XRef] {
			unlinked = append(unlinked, record.XRef)
			report.AddDropped(gedcom.ConversionNote{
				Path:     BuildRecordPath(string(record.Type), record.XRef),
				Original: "OBJE",
				Reason:   unlinkedReason,
			})
			continue
		}
		parts := records[record.XRef]
		for _, f := range parts.files {
			if len(f.title) > 0 && !f.titleUsed {
				parts.lost = append(parts.lost, []string{"FILE", "TITL"})
			}
		}
		for _, p := range parts.lost {
			report.AddDropped(gedcom.ConversionNote{
				Path:     BuildNestedPath(string(record.Type), record.XRef, p...),
				Original: p[len(p)-1],
				Reason:   reason,
			})
		}
		if len(parts.lost) > 0 {
			lost = append(lost, record.XRef)
		}
	}
	clear(doc.Records[len(kept):])
	doc.Records = kept

	if len(lost) > 0 {
		report.AddDataLoss(gedcom.DataLossItem{Feature: "OBJE record structures", Reason: reason, AffectedRecords: lost})
	}
	if len(unlinked) > 0 {
		report.AddDataLoss(gedcom.DataLossItem{Feature: "Unlinked OBJE records", Reason: unlinkedReason, AffectedRecords: unlinked})
	}
}

// mediaRecordParts is what of a 7.0 OBJE record an inline 5.5 link holds.
type mediaRecordParts struct {
	files []mediaFileParts
	notes []*gedcom.Tag // the record's NOTE blocks, at level 1
	lost  [][]string    // tag paths of what no link holds
}

// mediaFileParts is one FILE of the record: the FILE (with what stays under
// it, at level 2), and its FORM and TITL blocks, each at level 2. titleUsed
// says some link holds the TITL; every pointer with its own TITL replaces it.
type mediaFileParts struct {
	file, form, title []*gedcom.Tag
	titleUsed         bool
}

// linkTags is the number of tags the record's inline links for one pointer
// copy, less the pointer's own subordinates.
func (parts *mediaRecordParts) linkTags() int {
	n := 0
	for _, f := range parts.files {
		n += 1 + len(f.form) + len(f.title) + len(f.file) + len(parts.notes)
	}
	return n
}

// splitMediaRecord sorts an OBJE record's tags into mediaRecordParts, or
// returns nil when the record has no FILE.
func splitMediaRecord(tags []*gedcom.Tag) *mediaRecordParts {
	parts := &mediaRecordParts{}
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil || tag.Level != 1 {
			continue
		}
		end := blockEnd(tags, i)
		switch tag.Tag {
		case "FILE":
			parts.files = append(parts.files, splitMediaFile(tags[i:end], &parts.lost))
		case "NOTE":
			parts.notes = append(parts.notes, tags[i:end]...)
		default:
			parts.lost = append(parts.lost, []string{tag.Tag})
		}
		i = end - 1
	}
	if len(parts.files) == 0 {
		return nil
	}
	return parts
}

// splitMediaFile sorts a FILE block (block[0] the FILE) into its parts,
// adding the path of each substructure no inline link holds to lost. Line
// continuations and extensions stay under the FILE; a FORM keeps only its
// continuations, losing its MEDI.
func splitMediaFile(block []*gedcom.Tag, lost *[][]string) mediaFileParts {
	f := mediaFileParts{file: []*gedcom.Tag{block[0]}}
	for i := 1; i < len(block); i++ {
		tag := block[i]
		if tag == nil || tag.Level != 2 {
			continue
		}
		end := blockEnd(block, i)
		switch {
		case tag.Tag == "FORM":
			f.form = append(f.form, tag)
			for _, sub := range block[i+1 : end] {
				switch {
				case sub == nil:
				case sub.Level == 3 && (sub.Tag == "CONT" || sub.Tag == "CONC"):
					f.form = append(f.form, sub)
				case sub.Level == 3:
					*lost = append(*lost, []string{"FILE", "FORM", sub.Tag})
				}
			}
		case tag.Tag == "TITL":
			f.title = append(f.title, block[i:end]...)
		case tag.Tag == "CONT" || tag.Tag == "CONC" || strings.HasPrefix(tag.Tag, "_"):
			f.file = append(f.file, block[i:end]...)
		default:
			*lost = append(*lost, []string{"FILE", tag.Tag})
		}
		i = end - 1
	}
	return f
}

// inlineLinks returns the inline links, one per file, that replace a pointer
// at level whose subordinate tags are pointerSubs. Each link holds copies, in
// the 5.5 order FORM, TITL, FILE, NOTE.
func (parts *mediaRecordParts) inlineLinks(level int, pointerSubs []*gedcom.Tag) []*gedcom.Tag {
	var title, rest []*gedcom.Tag // the pointer's TITL block; its other subordinates
	for i := 0; i < len(pointerSubs); i++ {
		tag := pointerSubs[i]
		if tag == nil {
			continue
		}
		end := blockEnd(pointerSubs, i)
		if tag.Tag == "TITL" && tag.Level == level+1 && title == nil {
			title = shiftLevels(pointerSubs[i:end], 0)
		} else {
			rest = append(rest, shiftLevels(pointerSubs[i:end], 0)...)
		}
		i = end - 1
	}

	var out []*gedcom.Tag
	for n, f := range parts.files {
		out = append(out, &gedcom.Tag{Level: level, Tag: "OBJE"})
		// The FILE's FORM and TITL move up beside it: level 2 to level+1.
		out = append(out, shiftLevels(f.form, level-1)...)
		if title != nil {
			out = append(out, shiftLevels(title, 0)...)
		} else {
			out = append(out, shiftLevels(f.title, level-1)...)
			parts.files[n].titleUsed = true
		}
		out = append(out, shiftLevels(f.file, level)...)
		out = append(out, shiftLevels(parts.notes, level)...)
		if n == 0 {
			out = append(out, rest...)
		}
	}
	return out
}

// shiftLevels returns copies of tags (nil ones dropped) moved down by delta
// levels.
func shiftLevels(tags []*gedcom.Tag, delta int) []*gedcom.Tag {
	out := make([]*gedcom.Tag, 0, len(tags))
	for _, t := range tags {
		if t == nil {
			continue
		}
		c := *t
		c.Level += delta
		out = append(out, &c)
	}
	return out
}

// nestMediaRecordForms moves a 5.5 OBJE record's level-1 FORM and TITL under
// its first FILE on a 5.5 -> 7.0 upgrade. A conforming 5.5 record has no FILE
// (it embeds a BLOB), but producers commonly write one beside the 5.5 FORM
// and TITL; 7.0 holds both under the FILE, and dropForeignStructures would
// otherwise remove them as 5.5-only record structures. A record with no
// level-1 FILE, or no level-1 FORM or TITL, is left alone. Each move is a
// Normalized note.
func nestMediaRecordForms(doc *gedcom.Document, report *gedcom.ConversionReport) {
	for _, record := range doc.Records {
		if record == nil || record.Type != gedcom.RecordTypeMedia {
			continue
		}
		var hasFile, hasForm bool
		for _, tag := range record.Tags {
			if tag != nil && tag.Level == 1 {
				hasFile = hasFile || tag.Tag == "FILE"
				hasForm = hasForm || tag.Tag == "FORM" || tag.Tag == "TITL"
			}
		}
		if !hasFile || !hasForm {
			continue
		}
		record.Tags = mediaRecordTags(record.Tags, 0)
		report.AddNormalized(gedcom.ConversionNote{
			Path:     BuildNestedPath(string(record.Type), record.XRef, "FILE"),
			Original: "OBJE record FORM/TITL beside FILE",
			Result:   "FORM/TITL under FILE",
			Reason:   "GEDCOM 7.0 gives a file's format and title under its FILE",
		})
	}
}
