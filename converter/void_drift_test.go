package converter

import (
	"reflect"
	"testing"

	"github.com/cacack/gedcom-go/v3/gedcom"
)

// TestPruneVoidEntityCoversWalker guards voidPruner against drift from
// gedcom's XRef walker (issue #670): every string field reachable from each
// entity type is set to @VOID@, and after pruning the walker must find none,
// so a pointer field added to the walker but not the pruner fails here.
func TestPruneVoidEntityCoversWalker(t *testing.T) {
	entities := []any{
		&gedcom.Individual{}, &gedcom.Family{}, &gedcom.Source{},
		&gedcom.Repository{}, &gedcom.Note{}, &gedcom.MediaObject{},
		&gedcom.Submitter{}, &gedcom.SharedNote{},
	}
	for _, entity := range entities {
		name := reflect.TypeOf(entity).Elem().Name()
		t.Run(name, func(t *testing.T) {
			fillVoid(reflect.ValueOf(entity).Elem(), 0)
			record := &gedcom.Record{Entity: entity}
			if countVoid(record) == 0 {
				t.Fatal("filler produced no @VOID@ pointers for the walker to visit")
			}
			if !pruneVoidEntity(entity) {
				t.Error("pruneVoidEntity reported no removal")
			}
			if n := countVoid(record); n != 0 {
				t.Errorf("walker still visits %d @VOID@ pointers after pruning", n)
			}
		})
	}
}

// countVoid counts the @VOID@ pointers the walker visits in r, reading it as
// 5.5.1, where @VOID@ is an ordinary identifier.
func countVoid(r *gedcom.Record) int {
	n := 0
	gedcom.VisitForVersion(r, gedcom.Version551, func(s string) {
		if s == v7NullXRef {
			n++
		}
	})
	return n
}

// fillVoid sets every settable string reachable from v to @VOID@, allocating
// one element for each slice and a value for each nil struct pointer. depth
// bounds recursive types.
func fillVoid(v reflect.Value, depth int) {
	if depth > 8 || !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(v7NullXRef)
	case reflect.Pointer:
		if v.Type().Elem().Kind() != reflect.Struct {
			return
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		fillVoid(v.Elem(), depth+1)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fillVoid(v.Field(i), depth+1)
		}
	case reflect.Slice:
		v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		fillVoid(v.Index(0), depth+1)
	}
}
