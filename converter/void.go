package converter

import (
	"slices"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// voidPointerReason explains each @VOID@ pointer a downgrade removes.
const voidPointerReason = "GEDCOM 5.5/5.5.1 has no null pointer; @VOID@ would be an XRef no record defines"

// dropVoidPointers removes every GEDCOM 7.0 @VOID@ null pointer on a 7.0 ->
// 5.5/5.5.1 downgrade, together with the structure that holds it, and reports
// each as data loss (issue #670). In 5.x @VOID@ is an ordinary identifier, so
// a kept one would be a dangling pointer that Subset and the validator reject.
//
// Both representations are pruned, as dropV7OnlyTags does: each tag whose
// value or XRef is @VOID@ is removed with its subordinate block from the
// header's, each record's and each entity's Tags, and each typed structure
// whose pointer is @VOID@ (a FAMC/FAMS link, ASSO, citation, media link,
// repository link, or note pointer) is removed from its entity. A HUSB, WIFE,
// header SUBM or LDS FAMC pointer is cleared instead, since its field is a
// plain string. A record without Tags is reported from its entity.
func dropVoidPointers(doc *gedcom.Document, report *gedcom.ConversionReport, targetVersion gedcom.Version) {
	var affected []string
	lost := false

	if h := doc.Header; h != nil {
		var removed map[string]bool
		h.Tags, removed = dropTagBlocksIn(h.Tags, isVoidPointerTag)
		if h.Submitter == v7NullXRef {
			h.Submitter = ""
			if removed == nil {
				removed = make(map[string]bool)
			}
			removed["SUBM"] = true
		}
		for _, tag := range sortedKeys(removed) {
			lost = true
			report.AddDropped(gedcom.ConversionNote{
				Path:     BuildNestedPath("HEAD", "", tag),
				Original: tag + " " + v7NullXRef,
				Reason:   voidPointerReason,
			})
		}
	}

	for _, record := range doc.Records {
		if record == nil {
			continue
		}
		hadTags := len(record.Tags) > 0
		removed := dropTagBlocks(record, isVoidPointerTag)
		entityRemoved := pruneVoidEntity(record.Entity)
		if !hadTags && entityRemoved {
			removed = map[string]bool{"": true}
		}
		if len(removed) == 0 {
			continue
		}
		lost = true
		affected = append(affected, record.XRef)
		for _, tag := range sortedKeys(removed) {
			path := BuildNestedPath(string(record.Type), record.XRef)
			original := v7NullXRef
			if tag != "" {
				path = BuildNestedPath(string(record.Type), record.XRef, tag)
				original = tag + " " + v7NullXRef
			}
			report.AddDropped(gedcom.ConversionNote{
				Path:     path,
				Original: original,
				Reason:   voidPointerReason,
			})
		}
	}

	if lost {
		report.AddDataLoss(gedcom.DataLossItem{
			Feature:         "@VOID@ null pointers",
			Reason:          "GEDCOM " + targetVersion.String() + " has no null pointer",
			AffectedRecords: affected,
		})
	}
}

// isVoidPointerTag reports whether tag points at the 7.0 null pointer.
func isVoidPointerTag(tag *gedcom.Tag) bool {
	return tag.Value == v7NullXRef || tag.XRef == v7NullXRef
}

func isVoid(s string) bool { return s == v7NullXRef }

// sortedKeys returns m's keys in sorted order, so reports are deterministic.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// voidPruner removes @VOID@ pointers from a typed entity, mirroring the
// pointer fields gedcom's XRef walker visits, and records whether it removed
// any.
type voidPruner struct{ removed bool }

// pruneVoidEntity removes every @VOID@ pointer from entity and reports whether
// it removed any.
func pruneVoidEntity(entity any) bool {
	p := &voidPruner{}
	switch e := entity.(type) {
	case *gedcom.Individual:
		p.individual(e)
	case *gedcom.Family:
		p.family(e)
	case *gedcom.Source:
		p.source(e)
	case *gedcom.Repository:
		if e != nil {
			e.NoteXRefs = p.strings(e.NoteXRefs)
			e.Tags = p.tags(e.Tags)
		}
	case *gedcom.Note:
		if e != nil {
			e.Tags = p.tags(e.Tags)
		}
	case *gedcom.MediaObject:
		p.media(e)
	case *gedcom.Submitter:
		if e != nil {
			e.NoteXRefs = p.strings(e.NoteXRefs)
			e.Tags = p.tags(e.Tags)
		}
	case *gedcom.SharedNote:
		if e != nil {
			e.SourceCitations = p.citations(e.SourceCitations)
			p.changeDate(e.ChangeDate)
			e.Tags = p.tags(e.Tags)
		}
	}
	return p.removed
}

// deleteVoid removes each element of s whose pointer (ref) is @VOID@.
func deleteVoid[T any](p *voidPruner, s []T, ref func(T) string) []T {
	out := slices.DeleteFunc(s, func(v T) bool { return isVoid(ref(v)) })
	if len(out) != len(s) {
		p.removed = true
	}
	return out
}

func (p *voidPruner) strings(ss []string) []string {
	return deleteVoid(p, ss, func(s string) string { return s })
}

// clear empties a single pointer field that holds @VOID@.
func (p *voidPruner) clear(s *string) {
	if isVoid(*s) {
		*s = ""
		p.removed = true
	}
}

func (p *voidPruner) tags(tags []*gedcom.Tag) []*gedcom.Tag {
	out, removed := dropTagBlocksIn(tags, isVoidPointerTag)
	if len(removed) > 0 {
		p.removed = true
	}
	return out
}

func (p *voidPruner) individual(i *gedcom.Individual) {
	if i == nil {
		return
	}
	i.ChildInFamilies = p.familyLinks(i.ChildInFamilies)
	i.SpouseInFamilies = p.familyLinks(i.SpouseInFamilies)
	for _, n := range i.Names {
		if n != nil {
			n.NoteXRefs = p.strings(n.NoteXRefs)
			n.SourceCitations = p.citations(n.SourceCitations)
		}
	}
	i.NoteXRefs = p.strings(i.NoteXRefs)
	i.Associations = p.associations(i.Associations)
	i.SourceCitations = p.citations(i.SourceCitations)
	i.Media = p.mediaLinks(i.Media)
	p.events(i.Events)
	p.attributes(i.Attributes)
	p.ordinances(i.LDSOrdinances)
	p.changeDate(i.ChangeDate)
	p.changeDate(i.CreationDate)
	i.Tags = p.tags(i.Tags)
}

func (p *voidPruner) family(f *gedcom.Family) {
	if f == nil {
		return
	}
	p.clear(&f.Husband)
	p.clear(&f.Wife)
	f.Children = p.strings(f.Children)
	f.NoteXRefs = p.strings(f.NoteXRefs)
	f.SourceCitations = p.citations(f.SourceCitations)
	f.Media = p.mediaLinks(f.Media)
	p.events(f.Events)
	p.attributes(f.Attributes)
	p.ordinances(f.LDSOrdinances)
	p.changeDate(f.ChangeDate)
	p.changeDate(f.CreationDate)
	f.Tags = p.tags(f.Tags)
}

func (p *voidPruner) source(s *gedcom.Source) {
	if s == nil {
		return
	}
	s.RepositoryLinks = deleteVoid(p, s.RepositoryLinks, func(l *gedcom.SourceRepositoryLink) string {
		if l == nil {
			return ""
		}
		return l.XRef
	})
	for _, l := range s.RepositoryLinks {
		if l != nil {
			l.NoteXRefs = p.strings(l.NoteXRefs)
		}
	}
	s.NoteXRefs = p.strings(s.NoteXRefs)
	s.Media = p.mediaLinks(s.Media)
	p.changeDate(s.ChangeDate)
	p.changeDate(s.CreationDate)
	s.Tags = p.tags(s.Tags)
}

func (p *voidPruner) media(m *gedcom.MediaObject) {
	if m == nil {
		return
	}
	m.NoteXRefs = p.strings(m.NoteXRefs)
	m.SharedNoteXRefs = p.strings(m.SharedNoteXRefs)
	m.SourceCitations = p.citations(m.SourceCitations)
	p.changeDate(m.ChangeDate)
	p.changeDate(m.CreationDate)
	m.Tags = p.tags(m.Tags)
}

func (p *voidPruner) familyLinks(links []gedcom.FamilyLink) []gedcom.FamilyLink {
	links = deleteVoid(p, links, func(l gedcom.FamilyLink) string { return l.FamilyXRef })
	for k := range links {
		links[k].NoteXRefs = p.strings(links[k].NoteXRefs)
	}
	return links
}

func (p *voidPruner) associations(assocs []*gedcom.Association) []*gedcom.Association {
	assocs = deleteVoid(p, assocs, func(a *gedcom.Association) string {
		if a == nil {
			return ""
		}
		return a.IndividualXRef
	})
	for _, a := range assocs {
		if a != nil {
			a.NoteXRefs = p.strings(a.NoteXRefs)
			a.SourceCitations = p.citations(a.SourceCitations)
		}
	}
	return assocs
}

func (p *voidPruner) citations(cites []*gedcom.SourceCitation) []*gedcom.SourceCitation {
	cites = deleteVoid(p, cites, func(c *gedcom.SourceCitation) string {
		if c == nil {
			return ""
		}
		return c.SourceXRef
	})
	for _, c := range cites {
		if c != nil {
			c.NoteXRefs = p.strings(c.NoteXRefs)
		}
	}
	return cites
}

func (p *voidPruner) mediaLinks(links []*gedcom.MediaLink) []*gedcom.MediaLink {
	links = deleteVoid(p, links, func(l *gedcom.MediaLink) string {
		if l == nil {
			return ""
		}
		return l.MediaXRef
	})
	for _, l := range links {
		if l != nil {
			l.NoteXRefs = p.strings(l.NoteXRefs)
		}
	}
	return links
}

func (p *voidPruner) events(events []*gedcom.Event) {
	for _, e := range events {
		if e == nil {
			continue
		}
		e.NoteXRefs = p.strings(e.NoteXRefs)
		p.place(e.PlaceDetail)
		e.SourceCitations = p.citations(e.SourceCitations)
		e.Media = p.mediaLinks(e.Media)
		e.Associations = p.associations(e.Associations)
	}
}

func (p *voidPruner) attributes(attrs []*gedcom.Attribute) {
	for _, a := range attrs {
		if a == nil {
			continue
		}
		a.NoteXRefs = p.strings(a.NoteXRefs)
		p.place(a.PlaceDetail)
		a.SourceCitations = p.citations(a.SourceCitations)
		a.Media = p.mediaLinks(a.Media)
		a.Associations = p.associations(a.Associations)
	}
}

func (p *voidPruner) place(pd *gedcom.PlaceDetail) {
	if pd != nil {
		pd.NoteXRefs = p.strings(pd.NoteXRefs)
	}
}

func (p *voidPruner) ordinances(ords []*gedcom.LDSOrdinance) {
	for _, o := range ords {
		if o != nil {
			p.clear(&o.FamilyXRef)
			o.NoteXRefs = p.strings(o.NoteXRefs)
		}
	}
}

func (p *voidPruner) changeDate(cd *gedcom.ChangeDate) {
	if cd != nil {
		cd.NoteXRefs = p.strings(cd.NoteXRefs)
	}
}
