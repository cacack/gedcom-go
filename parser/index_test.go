package parser

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildIndex_Basic(t *testing.T) {
	input := `0 HEAD
1 SOUR TestSystem
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Should have 3 XRef entries
	if len(idx.entries) != 3 {
		t.Errorf("Got %d XRef entries, want 3", len(idx.entries))
	}

	// Should have 2 type entries (HEAD, TRLR)
	if len(idx.typeEntries) != 2 {
		t.Errorf("Got %d type entries, want 2", len(idx.typeEntries))
	}
}

func TestRecordIndex_Lookup(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John /Doe/
0 @I2@ INDI
1 NAME Jane /Doe/
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Lookup existing XRef
	entry, ok := idx.Lookup("@I1@")
	if !ok {
		t.Error("Expected to find @I1@")
	}
	if entry.XRef != "@I1@" {
		t.Errorf("Entry XRef = %q, want @I1@", entry.XRef)
	}
	if entry.Type != "INDI" {
		t.Errorf("Entry Type = %q, want INDI", entry.Type)
	}

	// Lookup another XRef
	entry, ok = idx.Lookup("@I2@")
	if !ok {
		t.Error("Expected to find @I2@")
	}
	if entry.Type != "INDI" {
		t.Errorf("Entry Type = %q, want INDI", entry.Type)
	}

	// Lookup non-existent XRef
	_, ok = idx.Lookup("@I999@")
	if ok {
		t.Error("Should not find @I999@")
	}
}

func TestRecordIndex_LookupByType(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// Lookup HEAD
	entry, ok := idx.LookupByType("HEAD")
	if !ok {
		t.Error("Expected to find HEAD")
	}
	if entry.Type != "HEAD" {
		t.Errorf("Entry Type = %q, want HEAD", entry.Type)
	}
	if entry.ByteOffset != 0 {
		t.Errorf("HEAD ByteOffset = %d, want 0", entry.ByteOffset)
	}

	// Lookup TRLR
	entry, ok = idx.LookupByType("TRLR")
	if !ok {
		t.Error("Expected to find TRLR")
	}
	if entry.Type != "TRLR" {
		t.Errorf("Entry Type = %q, want TRLR", entry.Type)
	}

	// INDI has XRef, so shouldn't be in typeEntries
	_, ok = idx.LookupByType("INDI")
	if ok {
		t.Error("INDI should not be in type entries (it has XRef)")
	}
}

func TestRecordIndex_XRefs(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 @F1@ FAM
0 @S1@ SOUR
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	xrefs := idx.XRefs()
	if len(xrefs) != 3 {
		t.Errorf("Got %d XRefs, want 3", len(xrefs))
	}

	// Check all expected XRefs are present
	xrefMap := make(map[string]bool)
	for _, x := range xrefs {
		xrefMap[x] = true
	}

	for _, expected := range []string{"@I1@", "@F1@", "@S1@"} {
		if !xrefMap[expected] {
			t.Errorf("XRefs missing %s", expected)
		}
	}
}

func TestRecordIndex_Types(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	types := idx.Types()
	if len(types) != 2 {
		t.Errorf("Got %d types, want 2", len(types))
	}

	typeMap := make(map[string]bool)
	for _, ty := range types {
		typeMap[ty] = true
	}

	if !typeMap["HEAD"] {
		t.Error("Types missing HEAD")
	}
	if !typeMap["TRLR"] {
		t.Error("Types missing TRLR")
	}
}

func TestRecordIndex_Len(t *testing.T) {
	input := `0 HEAD
0 @I1@ INDI
0 @I2@ INDI
0 TRLR`

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// 2 XRef entries + 2 type entries = 4
	if idx.Len() != 4 {
		t.Errorf("Len() = %d, want 4", idx.Len())
	}
}

func TestRecordIndex_Encoding(t *testing.T) {
	idx := NewRecordIndex()

	// Default encoding is empty
	if idx.Encoding() != "" {
		t.Errorf("Default encoding = %q, want empty", idx.Encoding())
	}

	// Set encoding
	idx.SetEncoding("UTF-8")
	if idx.Encoding() != "UTF-8" {
		t.Errorf("Encoding() = %q, want UTF-8", idx.Encoding())
	}
}

func TestRecordIndex_SaveLoad(t *testing.T) {
	input := `0 HEAD
1 SOUR Test
0 @I1@ INDI
1 NAME John /Doe/
0 @F1@ FAM
1 HUSB @I1@
0 TRLR`

	// Build index
	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}
	idx.SetEncoding("UTF-8")

	// Save to buffer
	var buf bytes.Buffer
	if err := idx.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Load from buffer
	loadedIdx, err := LoadIndex(&buf)
	if err != nil {
		t.Fatalf("LoadIndex error: %v", err)
	}

	// Verify loaded index matches original
	if loadedIdx.Encoding() != idx.Encoding() {
		t.Errorf("Loaded encoding = %q, want %q", loadedIdx.Encoding(), idx.Encoding())
	}

	if loadedIdx.Len() != idx.Len() {
		t.Errorf("Loaded Len() = %d, want %d", loadedIdx.Len(), idx.Len())
	}

	// Verify XRef lookups work
	for _, xref := range idx.XRefs() {
		orig, _ := idx.Lookup(xref)
		loaded, ok := loadedIdx.Lookup(xref)
		if !ok {
			t.Errorf("Loaded index missing XRef %s", xref)
			continue
		}
		if loaded.ByteOffset != orig.ByteOffset {
			t.Errorf("XRef %s: ByteOffset = %d, want %d", xref, loaded.ByteOffset, orig.ByteOffset)
		}
		if loaded.ByteLength != orig.ByteLength {
			t.Errorf("XRef %s: ByteLength = %d, want %d", xref, loaded.ByteLength, orig.ByteLength)
		}
	}

	// Verify type lookups work
	for _, ty := range idx.Types() {
		orig, _ := idx.LookupByType(ty)
		loaded, ok := loadedIdx.LookupByType(ty)
		if !ok {
			t.Errorf("Loaded index missing type %s", ty)
			continue
		}
		if loaded.ByteOffset != orig.ByteOffset {
			t.Errorf("Type %s: ByteOffset = %d, want %d", ty, loaded.ByteOffset, orig.ByteOffset)
		}
	}
}

func TestRecordIndex_SaveLoadEmpty(t *testing.T) {
	idx := NewRecordIndex()

	var buf bytes.Buffer
	if err := idx.Save(&buf); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	loaded, err := LoadIndex(&buf)
	if err != nil {
		t.Fatalf("LoadIndex error: %v", err)
	}

	if loaded.Len() != 0 {
		t.Errorf("Loaded Len() = %d, want 0", loaded.Len())
	}
}

func TestBuildIndex_ParseError(t *testing.T) {
	input := "0 HEAD\nINVALID LINE\n0 TRLR"

	_, err := BuildIndex(strings.NewReader(input))
	if err == nil {
		t.Error("Expected error from invalid input")
	}
}

func TestBuildIndex_EmptyInput(t *testing.T) {
	idx, err := BuildIndex(strings.NewReader(""))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	if idx.Len() != 0 {
		t.Errorf("Len() = %d, want 0", idx.Len())
	}
}

func TestLoadIndex_InvalidData(t *testing.T) {
	// Invalid gob data
	_, err := LoadIndex(strings.NewReader("not valid gob data"))
	if err == nil {
		t.Error("Expected error from invalid data")
	}
}

func TestRecordIndex_ByteOffsets(t *testing.T) {
	// Carefully constructed for predictable byte offsets
	// "0 HEAD\n" = 7 bytes
	// "1 SOUR Test\n" = 12 bytes
	// "0 @I1@ INDI\n" = 12 bytes
	// "1 NAME John\n" = 12 bytes
	// "0 TRLR\n" = 7 bytes
	input := "0 HEAD\n1 SOUR Test\n0 @I1@ INDI\n1 NAME John\n0 TRLR\n"

	idx, err := BuildIndex(strings.NewReader(input))
	if err != nil {
		t.Fatalf("BuildIndex error: %v", err)
	}

	// HEAD at offset 0
	headEntry, ok := idx.LookupByType("HEAD")
	if !ok {
		t.Fatal("HEAD not found")
	}
	if headEntry.ByteOffset != 0 {
		t.Errorf("HEAD ByteOffset = %d, want 0", headEntry.ByteOffset)
	}
	// HEAD + SOUR = 7 + 12 = 19 bytes
	if headEntry.ByteLength != 19 {
		t.Errorf("HEAD ByteLength = %d, want 19", headEntry.ByteLength)
	}

	// @I1@ at offset 19
	i1Entry, ok := idx.Lookup("@I1@")
	if !ok {
		t.Fatal("@I1@ not found")
	}
	if i1Entry.ByteOffset != 19 {
		t.Errorf("@I1@ ByteOffset = %d, want 19", i1Entry.ByteOffset)
	}
	// INDI + NAME = 12 + 12 = 24 bytes
	if i1Entry.ByteLength != 24 {
		t.Errorf("@I1@ ByteLength = %d, want 24", i1Entry.ByteLength)
	}

	// TRLR at offset 43
	trlrEntry, ok := idx.LookupByType("TRLR")
	if !ok {
		t.Fatal("TRLR not found")
	}
	if trlrEntry.ByteOffset != 43 {
		t.Errorf("TRLR ByteOffset = %d, want 43", trlrEntry.ByteOffset)
	}
}

func TestNewRecordIndex(t *testing.T) {
	idx := NewRecordIndex()

	if idx.entries == nil {
		t.Error("entries map not initialized")
	}
	if idx.typeEntries == nil {
		t.Error("typeEntries map not initialized")
	}
	if idx.version != IndexVersion {
		t.Errorf("version = %d, want %d", idx.version, IndexVersion)
	}
}

func TestIndexEntry_Fields(t *testing.T) {
	entry := IndexEntry{
		XRef:       "@I1@",
		Type:       "INDI",
		ByteOffset: 100,
		ByteLength: 50,
	}

	if entry.XRef != "@I1@" {
		t.Errorf("XRef = %q, want @I1@", entry.XRef)
	}
	if entry.Type != "INDI" {
		t.Errorf("Type = %q, want INDI", entry.Type)
	}
	if entry.ByteOffset != 100 {
		t.Errorf("ByteOffset = %d, want 100", entry.ByteOffset)
	}
	if entry.ByteLength != 50 {
		t.Errorf("ByteLength = %d, want 50", entry.ByteLength)
	}
}
