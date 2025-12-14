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

	// Family events
	EventMarriage   EventType = "MARR" // Marriage
	EventDivorce    EventType = "DIV"  // Divorce
	EventEngagement EventType = "ENGA" // Engagement
	EventAnnulment  EventType = "ANUL" // Annulment
)

// Event represents a life event with date, place, and source information.
type Event struct {
	// Type is the event type (birth, death, marriage, etc.)
	Type EventType

	// Date is when the event occurred (in GEDCOM date format)
	Date string

	// Place is where the event occurred
	Place string

	// Description provides additional details
	Description string

	// SourceCitations are source citations with page/quality details
	SourceCitations []*SourceCitation

	// Notes are references to note records
	Notes []string

	// MediaRefs are references to media objects
	MediaRefs []string

	// Tags contains all raw tags for this event (for unknown/custom fields)
	Tags []*Tag
}
