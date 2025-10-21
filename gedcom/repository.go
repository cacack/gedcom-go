package gedcom

// Repository represents a physical or digital location where sources are stored.
type Repository struct {
	// XRef is the cross-reference identifier for this repository
	XRef string

	// Name is the repository name
	Name string

	// Address is the physical address
	Address *Address

	// Notes are references to note records
	Notes []string

	// Tags contains all raw tags for this repository (for unknown/custom tags)
	Tags []*Tag
}

// Address represents a physical or digital address.
type Address struct {
	// Line1 is the first address line
	Line1 string

	// Line2 is the second address line (optional)
	Line2 string

	// Line3 is the third address line (optional)
	Line3 string

	// City is the city name
	City string

	// State is the state/province
	State string

	// PostalCode is the postal/zip code
	PostalCode string

	// Country is the country name
	Country string

	// Phone is the phone number (optional)
	Phone string

	// Email is the email address (optional)
	Email string

	// Website is the website URL (optional)
	Website string
}
