package converter

//go:generate go run ../internal/cmd/genforeign

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// specGraph is one version's structure graph (structures_gen.go): for each
// structure, its substructures by tag. The empty structure is the top level.
type specGraph map[string]map[string][]string

// structurePair is a tag under the structure it appears in.
type structurePair struct {
	super string
	tag   string
}

// structureTriple is a tag under the structure it appears in, under that
// structure's own parent.
type structureTriple struct {
	grand string
	super string
	tag   string
}

// foreignStructures describes one conversion direction for
// dropForeignStructures: the source version's structure graph and what of it
// the target version has no form for, all generated from the vendored
// specification files (issue #705).
type foreignStructures struct {
	source specGraph
	// foreign are the pairs the target lacks wherever they appear.
	foreign map[structurePair]bool
	// foreignIn are the pairs the target lacks only under some parents of
	// their superstructure, keyed with that parent.
	foreignIn map[structureTriple]bool
	// aliases names the source record type a record renamed by an earlier
	// pass started as (a 5.5.1 NOTE record is an SNOTE by then), so its
	// substructures are read in their source context.
	aliases map[gedcom.RecordType]string
	// mappedElsewhere are foreign tags another pass maps rather than drops;
	// the walk leaves them, and everything under them, for it.
	mappedElsewhere map[string]bool
	// exempt are foreign pairs left in place because the structure that
	// holds them is not converted yet, so dropping its parts alone would
	// lose data without making the output valid.
	exempt map[structurePair]bool
}

// foreign70To551 is the 7.0 -> 5.5.1 direction.
var foreign70To551 = &foreignStructures{
	source:    specStructures70,
	foreign:   foreignStructures70To551,
	foreignIn: foreignStructuresIn70To551,
	// downgradeSharedNotes has renamed every SNOTE record NOTE by then; a 7.0
	// source has no NOTE records of its own.
	aliases: map[gedcom.RecordType]string{gedcom.RecordTypeNote: "SNOTE"},
	// ASSO.ROLE and FORM.MEDI become RELA and TYPE (the enumeration
	// translation, #704).
	mappedElsewhere: map[string]bool{"ROLE": true, "MEDI": true},
}

// foreign551To70 is the 5.5.1 -> 7.0 direction.
var foreign551To70 = &foreignStructures{
	source:    specStructures551,
	foreign:   foreignStructures551To70,
	foreignIn: foreignStructuresIn551To70,
	aliases:   map[gedcom.RecordType]string{gedcom.RecordTypeSharedNote: "NOTE"},
	// ASSO.RELA and the record FILE.FORM.TYPE become ROLE and MEDI (the
	// enumeration translation, #704).
	mappedElsewhere: map[string]bool{"RELA": true, "TYPE": true},
}

// foreign70To55 is the 7.0 -> 5.5 direction (issue #794).
var foreign70To55 = &foreignStructures{
	source:    specStructures70,
	foreign:   foreignStructures70To55,
	foreignIn: foreignStructuresIn70To55,
	aliases:   map[gedcom.RecordType]string{gedcom.RecordTypeNote: "SNOTE"},
	// ASSO.ROLE becomes RELA (the enumeration translation, #704).
	mappedElsewhere: map[string]bool{"ROLE": true},
	// A 5.5 OBJE record has no FILE; demoteMediaRecords moves each record's
	// files into inline links after this pass, so they are kept for it.
	exempt: map[structurePair]bool{{"record-OBJE", "FILE"}: true},
}

// foreign55To70 is the 5.5 -> 7.0 direction (issue #794).
var foreign55To70 = &foreignStructures{
	source:    specStructures55,
	foreign:   foreignStructures55To70,
	foreignIn: foreignStructuresIn55To70,
	aliases:   map[gedcom.RecordType]string{gedcom.RecordTypeSharedNote: "NOTE"},
	// ASSO.RELA becomes ROLE (the enumeration translation, #704).
	mappedElsewhere: map[string]bool{"RELA": true},
	// As for 5.5.1 (#793).
	exempt: map[structurePair]bool{{"SOURCE_CITATION.SOUR#2", "TEXT"}: true},
}

// foreignDrop is one structure dropForeignStructures removed.
type foreignDrop struct {
	recordType string
	xref       string
	path       []string // tags from the record (or header) down to the dropped one
}

// dropForeignStructures removes, with its subordinate block, every structure
// in the header and records that the source version defines and the target
// version does not (dir.foreign), and reports each as data loss: one
// DataLoss item per parent.tag pair, naming the records, and a Dropped note
// per occurrence with its full path.
//
// It reads each tag in the context of the source structure it is: a record's
// level-1 tags under that record's structure, each deeper tag under the
// structures its parent can be. A tag the source does not define there (an
// extension, a nonconforming tag, or one an earlier pass has already
// rewritten into its target form) is kept, and so is everything under it,
// since the source grammar no longer says what it is. CONT and CONC are
// line continuations, never structures, and are always kept.
//
// It runs after every pass that maps a structure into its target form, so it
// removes only what is left: what has no target form at all. A record with no
// Tags is written from its entity and is not walked; the one typed field
// such a record can hold that the target lacks, and no earlier pass clears,
// is cleared instead (clearForeignEntityFields).
func dropForeignStructures(doc *gedcom.Document, dir *foreignStructures, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	var drops []foreignDrop
	if doc.Header != nil {
		var dropped [][]string
		doc.Header.Tags, dropped = dir.walk(doc.Header.Tags, "HEAD")
		for _, path := range dropped {
			drops = append(drops, foreignDrop{recordType: "HEAD", path: path})
		}
	}
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		root := string(record.Type)
		if alias, ok := dir.aliases[record.Type]; ok {
			root = alias
		}
		var dropped [][]string
		if len(record.Tags) == 0 {
			dropped = dir.clearForeignEntityFields(record, root)
		} else {
			record.Tags, dropped = dir.walk(record.Tags, root)
		}
		for _, path := range dropped {
			drops = append(drops, foreignDrop{recordType: string(record.Type), xref: record.XRef, path: path})
		}
	}
	reportForeignDrops(report, drops, targetVersion)
}

// walk returns tags without the foreign blocks under the top-level structure
// root, and the tag path of each block it removed. tags is returned itself
// when nothing is removed.
func (dir *foreignStructures) walk(tags []*gedcom.Tag, root string) (kept []*gedcom.Tag, dropped [][]string) {
	type frame struct {
		level      int
		tag        string
		candidates []candidate // what this tag can be in the source; nil when unknown
	}
	top := dir.recordCandidates(root)
	if len(top) == 0 {
		return tags, nil
	}
	stack := []frame{{level: 0, candidates: top}}

	var out []*gedcom.Tag
	for i := 0; i < len(tags); i++ {
		tag := tags[i]
		if tag == nil {
			if out != nil {
				out = append(out, tag)
			}
			continue
		}
		for len(stack) > 1 && stack[len(stack)-1].level >= tag.Level {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		candidates, foreign := dir.classify(parent.candidates, tag.Tag)
		if !foreign {
			if out != nil {
				out = append(out, tag)
			}
			stack = append(stack, frame{level: tag.Level, tag: tag.Tag, candidates: candidates})
			continue
		}

		path := make([]string, 0, len(stack))
		for _, f := range stack[1:] {
			path = append(path, f.tag)
		}
		dropped = append(dropped, append(path, tag.Tag))
		if out == nil {
			out = append(make([]*gedcom.Tag, 0, len(tags)), tags[:i]...)
		}
		var end int
		end, out = skipBlock(tags, i, out)
		i = end - 1
	}
	if out == nil {
		return tags, dropped
	}
	return out, dropped
}

// recordCandidates returns what the top-level tag root can be in the source.
func (dir *foreignStructures) recordCandidates(root string) []candidate {
	var top []candidate
	for _, s := range dir.source[""][root] {
		top = append(top, candidate{structure: s})
	}
	return top
}

// clearForeignEntityFields clears, on the entity of a record with no Tags,
// each typed field the encoder writes as a level-1 structure the target
// lacks, and returns a path per value cleared, as walk does per block. Only
// Submitter.RefNumbers needs it: REFN is 7.0-only on SUBM but valid on the
// other records, so it is not in v7OnlyTags, whose fields
// clearV7OnlyEntityFields clears earlier.
func (dir *foreignStructures) clearForeignEntityFields(record *gedcom.Record, root string) [][]string {
	subm, ok := record.Entity.(*gedcom.Submitter)
	if !ok || subm == nil || len(subm.RefNumbers) == 0 {
		return nil
	}
	if _, foreign := dir.classify(dir.recordCandidates(root), "REFN"); !foreign {
		return nil
	}
	dropped := make([][]string, len(subm.RefNumbers))
	for i := range dropped {
		dropped[i] = []string{"REFN"}
	}
	subm.RefNumbers = nil
	return dropped
}

// skipBlock returns the index just past the block tags[i] opens, and out
// with the block's nil tags appended: they have no level to place them in the
// block, so they are kept, as dropTagBlocksIn keeps them.
func skipBlock(tags []*gedcom.Tag, i int, out []*gedcom.Tag) (end int, kept []*gedcom.Tag) {
	end = i + 1
	for end < len(tags) && (tags[end] == nil || tags[end].Level > tags[i].Level) {
		if tags[end] == nil {
			out = append(out, nil)
		}
		end++
	}
	return end, out
}

// candidate is one source structure a tag can be, with the structure its
// parent was for it to be that.
type candidate struct {
	structure string
	parent    string
}

// classify reads tag under a parent that can be any of parents, returning
// what the tag can be and whether it is to be dropped as foreign: some
// parent defines it, and every parent that does defines it as a pair (or, in
// that parent's own context, a triple) the target lacks, not exempt and not
// mapped by another pass.
func (dir *foreignStructures) classify(parents []candidate, tag string) (candidates []candidate, foreign bool) {
	if tag == "CONT" || tag == "CONC" {
		return nil, false
	}
	defined, allForeign := false, true
	for _, p := range parents {
		children, ok := dir.source[p.structure][tag]
		if !ok {
			continue
		}
		defined = true
		pair := structurePair{p.structure, tag}
		lacks := dir.foreign[pair] || dir.foreignIn[structureTriple{p.parent, p.structure, tag}]
		if !lacks || dir.exempt[pair] {
			allForeign = false
		}
		for _, c := range children {
			candidates = append(candidates, candidate{structure: c, parent: p.structure})
		}
	}
	if defined && allForeign {
		return nil, !dir.mappedElsewhere[tag]
	}
	return candidates, false
}

// reportForeignDrops reports drops: a DataLoss item per parent.tag pair, in
// the order first dropped, naming each record it was dropped from, and a
// Dropped note per occurrence.
func reportForeignDrops(report *gedcom.ConversionReport, drops []foreignDrop, targetVersion gedcom.Version) {
	if len(drops) == 0 {
		return
	}
	reason := "Structure not defined in GEDCOM " + targetVersion.String()
	var features []string
	affected := map[string][]string{}
	for _, d := range drops {
		feature := d.path[len(d.path)-1]
		if len(d.path) > 1 {
			feature = d.path[len(d.path)-2] + "." + feature
		} else {
			feature = d.recordType + "." + feature
		}
		if _, ok := affected[feature]; !ok {
			features = append(features, feature)
			affected[feature] = nil
		}
		if d.xref != "" && !slices.Contains(affected[feature], d.xref) {
			affected[feature] = append(affected[feature], d.xref)
		}
		report.AddDropped(gedcom.ConversionNote{
			Path:     BuildNestedPath(d.recordType, d.xref, d.path...),
			Original: d.path[len(d.path)-1],
			Reason:   reason,
		})
	}
	for _, feature := range features {
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         feature + " structures",
			Reason:          reason,
			AffectedRecords: affected[feature],
		})
	}
	report.AddTransformation(gedcom.Transformation{
		Type:        "FOREIGN_STRUCTURES_DROPPED",
		Description: "Dropped structures GEDCOM " + targetVersion.String() + " does not define",
		Count:       len(drops),
	})
}
