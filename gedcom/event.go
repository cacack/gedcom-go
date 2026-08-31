package gedcom

// EventType represents the type of life event.
type EventType string

const (
	// EventBirth represents a birth event.
	EventBirth EventType = "BIRT"
	// EventDeath represents a death event.
	EventDeath EventType = "DEAT"
	// EventBaptism represents a baptism event.
	EventBaptism EventType = "BAPM"
	// EventBurial represents a burial event.
	EventBurial EventType = "BURI"
	// EventCensus represents a census event.
	EventCensus EventType = "CENS"
	// EventChristening represents a christening event.
	EventChristening EventType = "CHR"
	// EventAdoption represents an adoption event.
	EventAdoption EventType = "ADOP"
	// EventOccupation represents an occupation event.
	EventOccupation EventType = "OCCU"
	// EventResidence represents a residence event.
	EventResidence EventType = "RESI"
	// EventImmigration represents an immigration event.
	EventImmigration EventType = "IMMI"
	// EventEmigration represents an emigration event.
	EventEmigration EventType = "EMIG"

	// EventBarMitzvah represents a Bar Mitzvah event.
	EventBarMitzvah EventType = "BARM"
	// EventBasMitzvah represents a Bas Mitzvah (also Bat Mitzvah) event.
	EventBasMitzvah EventType = "BASM"
	// EventBlessing represents a blessing event.
	EventBlessing EventType = "BLES"
	// EventAdultChristening represents an adult christening event.
	EventAdultChristening EventType = "CHRA"
	// EventConfirmation represents a confirmation event.
	EventConfirmation EventType = "CONF"
	// EventFirstCommunion represents a first communion event.
	EventFirstCommunion EventType = "FCOM"

	// EventGraduation represents a graduation event.
	EventGraduation EventType = "GRAD"
	// EventRetirement represents a retirement event.
	EventRetirement EventType = "RETI"
	// EventNaturalization represents a naturalization event.
	EventNaturalization EventType = "NATU"
	// EventOrdination represents an ordination event.
	EventOrdination EventType = "ORDN"
	// EventProbate represents a probate event.
	EventProbate EventType = "PROB"
	// EventWill represents a will event.
	EventWill EventType = "WILL"
	// EventCremation represents a cremation event.
	EventCremation EventType = "CREM"

	// EventMarriage represents a marriage event.
	EventMarriage EventType = "MARR"
	// EventDivorce represents a divorce event.
	EventDivorce EventType = "DIV"
	// EventEngagement represents an engagement event.
	EventEngagement EventType = "ENGA"
	// EventAnnulment represents an annulment event.
	EventAnnulment EventType = "ANUL"

	// EventMarriageBann represents a marriage bann event.
	EventMarriageBann EventType = "MARB"
	// EventMarriageContract represents a marriage contract event.
	EventMarriageContract EventType = "MARC"
	// EventMarriageLicense represents a marriage license event.
	EventMarriageLicense EventType = "MARL"
	// EventMarriageSettlement represents a marriage settlement event.
	EventMarriageSettlement EventType = "MARS"
	// EventDivorceFiling represents a divorce filing event.
	EventDivorceFiling EventType = "DIVF"
)

// Coordinates represents geographic coordinates for a place.
type Coordinates struct {
	// Latitude in GEDCOM format (e.g., "N42.3601")
	Latitude string

	// Longitude in GEDCOM format (e.g., "W71.0589")
	Longitude string
}

// PlaceDetail represents a structured place with optional coordinates and format.
type PlaceDetail struct {
	// Name is the place name string
	Name string

	// Form is the hierarchical format of the place (e.g., "City, County, State, Country")
	Form string

	// Coordinates are optional geographic coordinates (MAP/LATI/LONG)
	Coordinates *Coordinates
}

// Event represents a life event with date, place, and source information.
//
// An event carries no raw-tag store of its own: Record.Tags is the single
// store for an event's unknown or custom subtags, and is what governs what a
// decoded record writes (see Record.Tags).
type Event struct {
	// Type is the event type (birth, death, marriage, etc.)
	Type EventType

	// Date is when the event occurred (in GEDCOM date format)
	Date string

	// ParsedDate is the parsed representation of Date.
	// This is nil if the date string could not be parsed.
	ParsedDate *Date

	// PlaceDetail is where the event occurred, with optional coordinates.
	// It is the sole place carrier; [Event.PlaceName] is the nil-safe read
	// path for just the name.
	PlaceDetail *PlaceDetail

	// Description provides additional details
	Description string

	// EventTypeDetail provides a descriptive type of the event (TYPE subordinate)
	EventTypeDetail string

	// Cause is the cause of the event (CAUS subordinate)
	Cause string

	// Age is the age at the time of the event (AGE subordinate)
	Age string

	// Agency is the responsible agency (AGNC subordinate)
	Agency string

	// ReligiousAffiliation is the religious denomination associated with the
	// event (RELI subordinate)
	ReligiousAffiliation string

	// Address is the event address structure (ADDR subordinate)
	Address *Address

	// Phone numbers associated with the event (PHON subordinate, can repeat)
	Phone []string

	// Email addresses associated with the event (EMAIL subordinate, can repeat)
	Email []string

	// Fax numbers associated with the event (FAX subordinate, can repeat)
	Fax []string

	// Websites associated with the event (WWW subordinate, can repeat)
	Website []string

	// Restriction notice for privacy controls (RESN subordinate)
	// Common values: "confidential", "locked", "privacy" (or combinations)
	Restriction string

	// UID is a unique identifier for the event (UID subordinate)
	UID string

	// SortDate is the date used for sorting events (SDATE subordinate, GEDCOM 7.0)
	// Typically in ISO 8601 format (e.g., "1900-01-01")
	SortDate string

	// IsNegative indicates this is a GEDCOM 7.0 negative assertion.
	// When true, it means the event did NOT occur (e.g., NO MARR = never married).
	// The NO tag is used to record explicit conclusions from research that an event
	// did not happen, which is different from simply having no information.
	IsNegative bool

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Associations are links to individuals associated with this event
	// (ASSO subordinate), e.g. witnesses, officiants or godparents
	Associations []*Association

	// NoteXRefs are XRef pointers to shared NOTE/SNOTE records (e.g. "@N1@").
	NoteXRefs []string

	// InlineNotes are note text values written directly on this event
	// (NOTE <text> form, including CONT/CONC continuations).
	InlineNotes []string

	// Notes is deprecated: use NoteXRefs and InlineNotes instead. It is kept
	// for backward compatibility and populated during decode with the inline
	// note text and shared-note XRefs interleaved in their original GEDCOM
	// order (not the NoteXRefs-then-InlineNotes order of the split fields).
	//
	// Deprecated: use NoteXRefs and InlineNotes.
	Notes []string

	// Media are references to media objects with optional crop/title
	Media []*MediaLink
}

// PlaceName returns PlaceDetail.Name, or "" when no place is recorded.
// Safe on a nil receiver and a nil PlaceDetail.
func (e *Event) PlaceName() string {
	if e == nil || e.PlaceDetail == nil {
		return ""
	}
	return e.PlaceDetail.Name
}

// SetPlaceName records name as the event's place, allocating PlaceDetail if the
// event does not have one yet and leaving any existing Form and Coordinates
// intact. Safe on a nil receiver, where it does nothing.
//
// This is the counterpart of [Event.PlaceName] and the supported write path:
// assigning through PlaceDetail directly panics on an event built without one,
// and leaving PlaceDetail nil means the encoder writes no PLAC line at all.
//
// An empty name on an event that has no PlaceDetail is a no-op, so threading an
// optional place through this method cannot conjure a placeless PLAC line. To
// clear a place that is already recorded, set PlaceDetail to nil -- calling this
// with an empty name only blanks the name, and a PlaceDetail whose Name is empty
// still encodes as a valueless PLAC line.
func (e *Event) SetPlaceName(name string) {
	if e == nil {
		return
	}
	if e.PlaceDetail == nil {
		if name == "" {
			return
		}
		e.PlaceDetail = &PlaceDetail{}
	}
	e.PlaceDetail.Name = name
}

// AllNotes returns this event's inline notes followed by the text of any
// shared notes referenced by NoteXRefs, resolved against doc. Shared notes that
// do not resolve are skipped. Returns nil when there are no notes.
func (e *Event) AllNotes(doc *Document) []string {
	return allNotes(doc, e.InlineNotes, e.NoteXRefs)
}
