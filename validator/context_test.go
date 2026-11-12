package validator

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
	"github.com/cacack/gedcom-go/v3/internal/ctxcheck"
)

// deathBeforeBirthDoc returns three individuals, each dying before birth.
func deathBeforeBirthDoc() *gedcom.Document {
	return makeDocument([]*gedcom.Individual{
		makeIndividual("@I1@", 1900, 1850),
		makeIndividual("@I2@", 1900, 1850),
		makeIndividual("@I3@", 1900, 1850),
	}, nil)
}

// deathBeforeBirthXRefs returns the RecordXRef of each CodeDeathBeforeBirth
// issue, in order.
func deathBeforeBirthXRefs(issues []Issue) []string {
	var xrefs []string
	for _, is := range issues {
		if is.Code == CodeDeathBeforeBirth {
			xrefs = append(xrefs, is.RecordXRef)
		}
	}
	return xrefs
}

// TestValidateAllWithErrorChecksPerRecord pins that the date logic check
// consults the context before each individual (#743). The context reports
// done from its fourth check: before the header check, before the date
// logic check, before @I1@, then before @I2@. So @I1@ is reported, @I2@ and
// @I3@ are not, and the error wraps context.Canceled.
func TestValidateAllWithErrorChecksPerRecord(t *testing.T) {
	v := NewWithOptions(&ValidateOptions{Context: ctxcheck.CancelAfter(3)})

	issues, err := v.ValidateAllWithError(deathBeforeBirthDoc())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one wrapping context.Canceled", err)
	}
	if got, want := deathBeforeBirthXRefs(issues), []string{"@I1@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("death-before-birth issues for %v, want %v (only those before the cancellation)", got, want)
	}
}

// TestValidateAllWithErrorCancelledBeforeStart pins that a context already
// done runs no check.
func TestValidateAllWithErrorCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	v := NewWithOptions(&ValidateOptions{Context: ctx})

	issues, err := v.ValidateAllWithError(deathBeforeBirthDoc())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want one wrapping context.Canceled", err)
	}
	if len(issues) != 0 {
		t.Errorf("issues = %v, want none: no check should run", issues)
	}
}

// TestValidateAllWithErrorLiveContext pins that a live context changes
// nothing: the issues are ValidateAll's and the error is nil.
func TestValidateAllWithErrorLiveContext(t *testing.T) {
	doc := deathBeforeBirthDoc()
	v := NewWithOptions(&ValidateOptions{Context: context.Background()})

	issues, err := v.ValidateAllWithError(doc)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got, want := deathBeforeBirthXRefs(issues), []string{"@I1@", "@I2@", "@I3@"}; !reflect.DeepEqual(got, want) {
		t.Errorf("death-before-birth issues for %v, want %v", got, want)
	}
	if want := New().ValidateAll(doc); !reflect.DeepEqual(issues, want) {
		t.Errorf("ValidateAllWithError = %v, want ValidateAll's %v", issues, want)
	}
}

// TestCancellableInnerChecksStop pins the per-record checks inside the
// nil-element walk and duplicate detection: given a context already done,
// each finds nothing even in a document where it would find something.
func TestCancellableInnerChecksStop(t *testing.T) {
	done, cancel := context.WithCancel(context.Background())
	cancel()

	nilDoc := &gedcom.Document{Records: []*gedcom.Record{nil}}
	if got := validateNilElements(context.Background(), nilDoc); len(got) == 0 {
		t.Fatal("live walk found no nil record; the test document is wrong")
	}
	if got := validateNilElements(done, nilDoc); len(got) != 0 {
		t.Errorf("cancelled walk found %v, want nothing", got)
	}

	dupDoc := makeDocument([]*gedcom.Individual{
		{XRef: "@I1@", Names: []*gedcom.PersonalName{{Full: "John /Smith/", Given: "John", Surname: "Smith"}}},
		{XRef: "@I2@", Names: []*gedcom.PersonalName{{Full: "John /Smith/", Given: "John", Surname: "Smith"}}},
	}, nil)
	d := NewDuplicateDetector(&DuplicateConfig{MinConfidence: 0.1})
	if got := d.findDuplicatesReport(context.Background(), dupDoc); len(got.Pairs) == 0 {
		t.Fatal("live detection found no pair; the test document is wrong")
	}
	if got := d.findDuplicatesReport(done, dupDoc); len(got.Pairs) != 0 {
		t.Errorf("cancelled detection found %v, want nothing", got.Pairs)
	}
}
