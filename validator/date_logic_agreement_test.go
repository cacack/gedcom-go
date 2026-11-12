package validator

import (
	"errors"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestMinimumAgeAgreesWithValidator runs the same date pairs through the
// gedcom package's age rules and the validator's UNREASONABLE_PARENT_AGE
// check, and asserts they reach the same too-young verdict (#817).
func TestMinimumAgeAgreesWithValidator(t *testing.T) {
	const minAge = 12
	pairs := []struct {
		birth, event string
		tooYoung     bool
	}{
		{"1910", "1905", false}, // event before birth: an ordering problem, not an age
		{"1850", "1861", true},
		{"1850", "1862", false},
		{"15 JAN 1850", "14 JAN 1862", true},
		{"15 JAN 1850", "15 JAN 1862", false},
		{"DEC 1850", "JAN 1862", true},
		{"BEF 1850", "1855", false},
		{"AFT 1850", "1855", true},
		{"1850", "BET 1855 AND 1870", false},
		{"1850", "MAR 1850", true},
		{"@#DJULIAN@ 1 JAN 1850", "13 JAN 1862", false},
		{"@#DJULIAN@ 1 JAN 1850", "12 JAN 1862", true},
	}

	v := NewDateLogicValidator(&DateLogicConfig{MinParentAge: minAge})
	for _, p := range pairs {
		t.Run(p.birth+" to "+p.event, func(t *testing.T) {
			birth, event := mustParseDate(t, p.birth), mustParseDate(t, p.event)

			parent := &gedcom.Individual{
				XRef:             "@I1@",
				Sex:              "M",
				Events:           []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: birth}},
				SpouseInFamilies: []gedcom.FamilyLink{{FamilyXRef: "@F1@"}},
			}
			child := &gedcom.Individual{
				XRef:   "@I2@",
				Events: []*gedcom.Event{{Type: gedcom.EventBirth, ParsedDate: event}},
			}
			family := &gedcom.Family{XRef: "@F1@", Husband: "@I1@", Children: []string{"@I2@"}}
			doc := makeDocument([]*gedcom.Individual{parent, child}, []*gedcom.Family{family})

			validatorTooYoung := false
			for _, issue := range v.checkReasonableParentAge(doc, parent) {
				if issue.Details["min_age"] != "" {
					validatorTooYoung = true
				}
			}

			parentErr := gedcom.ValidateParentChildDates(birth, event, minAge)
			parentTooYoung := errors.Is(parentErr, gedcom.ErrBelowMinimumAge)

			marriageErr := gedcom.ValidateMarriageDates(event, birth, nil, minAge)
			marriageTooYoung := errors.Is(marriageErr, gedcom.ErrBelowMinimumAge)

			if validatorTooYoung != p.tooYoung {
				t.Errorf("validator too young = %v, want %v", validatorTooYoung, p.tooYoung)
			}
			if parentTooYoung != validatorTooYoung {
				t.Errorf("ValidateParentChildDates too young = %v (%v), validator = %v",
					parentTooYoung, parentErr, validatorTooYoung)
			}
			if marriageTooYoung != validatorTooYoung {
				t.Errorf("ValidateMarriageDates too young = %v (%v), validator = %v",
					marriageTooYoung, marriageErr, validatorTooYoung)
			}
		})
	}
}
