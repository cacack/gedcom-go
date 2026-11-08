package converter

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// nameVariationTags are the GEDCOM 5.5.1 phonetic and romanized variations
// (under NAME and PLAC) that neither 5.5 nor 7.0 defines, in the order they
// are reported. 7.0 replaced both with TRAN, which names a language (LANG)
// rather than the scheme FONE/ROMN record in TYPE.
var nameVariationTags = []string{"FONE", "ROMN"}

// dropNameVariations removes every FONE and ROMN tag, at any level and
// together with its subordinate block, from each record's Tags on a
// conversion to a version without them (5.5 or 7.0), and reports each as
// data loss (issue #694). They are dropped rather than mapped to 7.0's TRAN,
// as dropV7OnlyTags drops TRAN going the other way and as the encoder drops
// them from typed names (#692): TRAN has no slot for the TYPE scheme.
//
// An individual's typed PersonalName.Phonetic and Romanized are cleared too,
// so the two representations agree and the encoder cannot write them back
// for a record without Tags. Such a record is reported from its entity.
func dropNameVariations(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	found := make(map[string][]string)
	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		hadTags := len(record.Tags) > 0
		removed := dropTagBlocks(record, func(tag *gedcom.Tag) bool {
			return slices.Contains(nameVariationTags, tag.Tag)
		})
		cleared := clearNameVariations(record.Entity)
		if !hadTags {
			removed = cleared
		}
		for _, tag := range nameVariationTags {
			if removed[tag] {
				found[tag] = append(found[tag], record.XRef)
			}
		}
	}
	reportDroppedTags(doc, report, nameVariationTags, found, targetVersion)
}

// clearNameVariations clears Phonetic and Romanized on every name of an
// individual entity and returns the tags (FONE, ROMN) whose fields held data.
func clearNameVariations(entity any) map[string]bool {
	indi, ok := entity.(*gedcom.Individual)
	if !ok || indi == nil {
		return nil
	}
	cleared := make(map[string]bool)
	for _, name := range indi.Names {
		if name == nil {
			continue
		}
		if len(name.Phonetic) > 0 {
			cleared["FONE"] = true
			name.Phonetic = nil
		}
		if len(name.Romanized) > 0 {
			cleared["ROMN"] = true
			name.Romanized = nil
		}
	}
	return cleared
}
