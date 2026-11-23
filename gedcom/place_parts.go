package gedcom

import "strings"

// PlaceFormSource records where the form that labels a place's parts came from.
type PlaceFormSource int

const (
	// PlaceFormNone means no form was available, so no part carries a label.
	PlaceFormNone PlaceFormSource = iota

	// PlaceFormOwn means the labels come from the place's own PLAC.FORM.
	PlaceFormOwn

	// PlaceFormHeader means the labels come from the default form passed in,
	// which is HEAD.PLAC.FORM when called through Document.PlaceParts.
	PlaceFormHeader
)

// PlacePart is one comma-separated part of a place name, paired with the label
// the file states for that position.
type PlacePart struct {
	// Name is the part's text, trimmed of surrounding whitespace; "" for an
	// empty part.
	Name string

	// Label is the form's label at the same position, trimmed; "" when the
	// file states none.
	Label string
}

// PlaceParts is a place name split into its parts, in file order, together
// with the form that labels them. See [PlaceDetail.Parts].
type PlaceParts struct {
	// Parts are the comma-separated parts of the place name in file order.
	// Empty parts are kept so positions stay aligned. Nil when the name is
	// empty.
	Parts []PlacePart

	// Labels are the form's labels, trimmed, in file order; nil when Source
	// is PlaceFormNone. They are set even when Mismatch is true, so a caller
	// can show what the file said.
	Labels []string

	// Source records which form supplied Labels.
	Source PlaceFormSource

	// Mismatch is true when a form exists but its label count differs from
	// the part count. Every PlacePart.Label is then "". It is false when the
	// name is empty.
	Mismatch bool
}

// Parts returns the comma-separated parts of the place name in file order,
// each paired with the label the file states for that position. Safe on a nil
// receiver, which returns the zero PlaceParts.
//
// The labels come from the place's own Form when that is non-empty after
// trimming, otherwise from defaultForm (the file's HEAD.PLAC.FORM; see
// [Document.PlaceParts]) when that is non-empty after trimming, otherwise
// there is no form and every label is "". PlaceParts.Source records which.
//
// The name and the form are split on commas and each piece is trimmed of
// surrounding whitespace. Empty parts are kept, so ", , Utah, USA" is four
// parts, the first two empty. An empty name yields no parts.
//
// Labels are never inferred from position. A part carries a label only when
// the form has exactly as many labels as the name has parts; when the counts
// differ, PlaceParts.Mismatch is true and no part is labeled -- the parts are
// not anchored to either end of the form. Nothing here assigns a meaning such
// as "country" to a position.
//
// A comma inside a part cannot be distinguished from a separator: GEDCOM
// defines no escape for it, so such a name splits into more parts than its
// author intended and typically reports a mismatch.
func (p *PlaceDetail) Parts(defaultForm string) PlaceParts {
	if p == nil {
		return PlaceParts{}
	}

	var result PlaceParts
	if labels := splitPlaceList(p.Form); labels != nil {
		result.Labels, result.Source = labels, PlaceFormOwn
	} else if labels := splitPlaceList(defaultForm); labels != nil {
		result.Labels, result.Source = labels, PlaceFormHeader
	}

	names := splitPlaceList(p.Name)
	if names == nil {
		return result
	}

	result.Mismatch = result.Source != PlaceFormNone && len(result.Labels) != len(names)
	result.Parts = make([]PlacePart, len(names))
	for i, name := range names {
		result.Parts[i].Name = name
		if result.Source != PlaceFormNone && !result.Mismatch {
			result.Parts[i].Label = result.Labels[i]
		}
	}
	return result
}

// PlaceParts returns p.Parts with the document's default place form,
// Header.PlaceForm (HEAD.PLAC.FORM), as the fallback for a place that has no
// FORM of its own. Safe on a nil receiver and a nil Header, which are treated
// as having no default form, and on a nil p.
//
// The default form is document-wide. A document produced by merge.Combine
// carries one input's form, so a place from the other input that has no FORM
// of its own is labeled by a form its source file did not state for it.
func (d *Document) PlaceParts(p *PlaceDetail) PlaceParts {
	if d == nil || d.Header == nil {
		return p.Parts("")
	}
	return p.Parts(d.Header.PlaceForm)
}

// splitPlaceList splits a comma-separated place name or form and trims each
// piece, keeping empty pieces. It returns nil when s is empty after trimming.
func splitPlaceList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	pieces := strings.Split(s, ",")
	for i, piece := range pieces {
		pieces[i] = strings.TrimSpace(piece)
	}
	return pieces
}
