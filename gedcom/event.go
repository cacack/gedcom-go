package gedcom

// EventType represents the type of life event.
type EventType string

const (
	// Individual events
	EventBirth       EventType = "BIRT" // Birth
	EventDeath       EventType = "DEAT" // Death
	EventBaptism     EventType = "BAPM" // Baptism
	EventBurial      EventType = "BURI" // Burial
	EventCensus      EventType = "CENS" // Census
	EventChristening EventType = "CHR"  // Christening
	EventAdoption    EventType = "ADOP" // Adoption
	EventOccupation  EventType = "OCCU" // Occupation
	EventResidence   EventType = "RESI" // Residence
	EventImmigration EventType = "IMMI" // Immigration
	EventEmigration  EventType = "EMIG" // Emigration

	// Religious events
	EventBarMitzvah       EventType = "BARM" // Bar Mitzvah
	EventBasMitzvah       EventType = "BASM" // Bas Mitzvah (also Bat Mitzvah)
	EventBlessing         EventType = "BLES" // Blessing
	EventAdultChristening EventType = "CHRA" // Adult Christening
	EventConfirmation     EventType = "CONF" // Confirmation
	EventFirstCommunion   EventType = "FCOM" // First Communion

	// Life events
	EventGraduation     EventType = "GRAD" // Graduation
	EventRetirement     EventType = "RETI" // Retirement
	EventNaturalization EventType = "NATU" // Naturalization
	EventOrdination     EventType = "ORDN" // Ordination
	EventProbate        EventType = "PROB" // Probate
	EventWill           EventType = "WILL" // Will
	EventCremation      EventType = "CREM" // Cremation

	// Family events
	EventMarriage   EventType = "MARR" // Marriage
	EventDivorce    EventType = "DIV"  // Divorce
	EventEngagement EventType = "ENGA" // Engagement
	EventAnnulment  EventType = "ANUL" // Annulment

	// Family events - extended
	EventMarriageBann       EventType = "MARB" // Marriage Bann
	EventMarriageContract   EventType = "MARC" // Marriage Contract
	EventMarriageLicense    EventType = "MARL" // Marriage License
	EventMarriageSettlement EventType = "MARS" // Marriage Settlement
	EventDivorceFiling      EventType = "DIVF" // Divorce Filing
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
type Event struct {
	// Type is the event type (birth, death, marriage, etc.)
	Type EventType

	// Date is when the event occurred (in GEDCOM date format)
	Date string

	// Place is where the event occurred (kept for backward compatibility)
	Place string

	// PlaceDetail provides structured place information with optional coordinates
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

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// Media are references to media objects with optional crop/title
	Media []*MediaLink

	// Tags contains all raw tags for this event (for unknown/custom fields)
	Tags []*Tag
}
