package gedcom

import (
	"math"
	"time"
)

// DefaultLivingMaxAge is the age threshold, in years, that
// [Individual.PresumedLiving] applies when LivingConfig.MaxAge is not set.
const DefaultLivingMaxAge = 110

// LivingConfig configures [Individual.PresumedLiving].
type LivingConfig struct {
	// MaxAge is the age threshold in years: a dated event more than MaxAge
	// years before ReferenceYear rules out a living individual. Zero or
	// negative means DefaultLivingMaxAge.
	MaxAge int

	// ReferenceYear is the Gregorian year the test is made in. Zero means
	// the current year; set it to make results reproducible, as in tests.
	ReferenceYear int
}

// LivingReason names the rule that decided [Individual.PresumedLiving].
type LivingReason string

const (
	// LivingReasonDeath: the individual has a death, burial or cremation
	// event (DEAT, BURI, CREM), with or without a date, including the bare
	// death flag "1 DEAT Y". Not living.
	LivingReasonDeath LivingReason = "death"

	// LivingReasonOldEvent: one of the individual's own events or
	// attributes is dated more than MaxAge years before ReferenceYear.
	// Not living.
	LivingReasonOldEvent LivingReason = "old-event"

	// LivingReasonOldFamilyEvent: an event of a family the individual is a
	// spouse in (marriage, divorce, ...) is dated more than MaxAge years
	// before ReferenceYear. Not living.
	LivingReasonOldFamilyEvent LivingReason = "old-family-event"

	// LivingReasonOldChildEvent: an event or attribute of one of the
	// individual's children (a birth, say) is dated more than MaxAge years
	// before ReferenceYear, so the individual was born before it. Not living.
	LivingReasonOldChildEvent LivingReason = "old-child-event"

	// LivingReasonWithinThreshold: at least one date could be placed, but
	// none is more than MaxAge years before ReferenceYear. Presumed living.
	LivingReasonWithinThreshold LivingReason = "within-threshold"

	// LivingReasonNoDates: no date that can be placed in time was found,
	// so the individual is presumed living, the protective default.
	LivingReasonNoDates LivingReason = "no-dates"
)

// PresumedLiving reports whether the individual is presumed to be living, and
// which rule decided, so that every consumer that publishes or shares tree
// data can apply the same test before redacting. The rules, in order:
//
//  1. A death, burial or cremation event (DEAT, BURI, CREM), dated or not,
//     including the death flag "1 DEAT Y": not living (LivingReasonDeath).
//  2. A dated event or attribute of the individual's own more than
//     cfg.MaxAge years before cfg.ReferenceYear: not living
//     (LivingReasonOldEvent).
//  3. The same for an event of a family the individual is a spouse in
//     (LivingReasonOldFamilyEvent).
//  4. The same for an event or attribute of one of the individual's
//     children (LivingReasonOldChildEvent).
//  5. Otherwise presumed living: LivingReasonWithinThreshold when some date
//     could be placed, LivingReasonNoDates when none could. An individual
//     with no dates at all is presumed living, the protective default.
//
// Only those close relatives are consulted. Parents' and spouses' dates are
// not: they bound the individual's birth only by assuming an age gap, and
// the test assumes nothing. Negative assertions (NO DEAT, IsNegative) are
// ignored, and so are LDS ordinances, which can be performed by proxy.
//
// An event counts as more than MaxAge years old when the latest year its
// date can denote ([Date.Bounds], in the proleptic Gregorian calendar) is
// more than MaxAge years before ReferenceYear: with the default of 110 and a
// reference year of 2026, a birth in 1915 or earlier decides, one in 1916
// does not, and "AFT 1800" never does. A phrase date, or one Bounds cannot
// place, is treated as no date. Rules 3 and 4 need doc; with a nil doc only
// the individual's own record is read. A nil individual is presumed living
// (LivingReasonNoDates).
func (i *Individual) PresumedLiving(doc *Document, cfg LivingConfig) (bool, LivingReason) {
	if i == nil {
		return true, LivingReasonNoDates
	}
	for _, event := range i.OccurredEvents() {
		switch event.Type {
		case EventDeath, EventBurial, EventCremation:
			return false, LivingReasonDeath
		}
	}

	maxAge := cfg.MaxAge
	if maxAge <= 0 {
		maxAge = DefaultLivingMaxAge
	}
	refYear := cfg.ReferenceYear
	if refYear == 0 {
		refYear = time.Now().Year()
	}
	s := livingScan{cutoffYear: refYear - maxAge}

	if s.individual(i) {
		return false, LivingReasonOldEvent
	}
	for _, fam := range i.FamiliesAsSpouse(doc) {
		if s.events(fam.OccurredEvents()) {
			return false, LivingReasonOldFamilyEvent
		}
	}
	for _, child := range i.Children(doc) {
		if s.individual(child) {
			return false, LivingReasonOldChildEvent
		}
	}
	if s.dated {
		return true, LivingReasonWithinThreshold
	}
	return true, LivingReasonNoDates
}

// livingScan looks for a date before cutoffYear, remembering whether any date
// could be placed at all.
type livingScan struct {
	cutoffYear int
	dated      bool
}

// individual reports whether any of ind's occurred events or attributes is
// dated before the cutoff.
func (s *livingScan) individual(ind *Individual) bool {
	if s.events(ind.OccurredEvents()) {
		return true
	}
	for _, attr := range ind.Attributes {
		if attr != nil && s.old(attr.ParsedDate) {
			return true
		}
	}
	return false
}

func (s *livingScan) events(events []*Event) bool {
	for _, event := range events {
		if s.old(event.ParsedDate) {
			return true
		}
	}
	return false
}

// old reports whether the latest year d can denote is before the cutoff.
func (s *livingScan) old(d *Date) bool {
	_, latest, ok := d.Bounds()
	if !ok {
		return false
	}
	s.dated = true
	if latest == math.MaxInt {
		return false
	}
	year, _, _ := JDNToGregorian(latest)
	return year < s.cutoffYear
}
