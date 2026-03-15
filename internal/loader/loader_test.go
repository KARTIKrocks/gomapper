package loader

import (
	"os"
	"path/filepath"
	"testing"
)

// testdataDir returns the absolute path to a testdata fixture package.
func testdataDir(t *testing.T, name string) string {
	t.Helper()
	// loader_test.go lives in internal/loader/, testdata is at repo root.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "..", "..", "testdata", name)
}

// --- Load tests ---

func TestLoad_ValidPackage(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if pkg.Name != "basic" {
		t.Errorf("expected package name 'basic', got %q", pkg.Name)
	}
}

func TestLoad_NonExistentDirectory(t *testing.T) {
	_, err := Load("/tmp/gomapper-nonexistent-dir-test")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestLoad_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for empty directory with no Go files")
	}
}

// --- LookupStruct tests ---

func TestLookupStruct_Found(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "User")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}
	if info.Name != "User" {
		t.Errorf("expected name 'User', got %q", info.Name)
	}
}

func TestLookupStruct_NotFound(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err = LookupStruct(pkg, "DoesNotExist")
	if err == nil {
		t.Fatal("expected error for non-existent type")
	}
}

func TestLookupStruct_NotAStruct(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, err = LookupStruct(pkg, "MyString")
	if err == nil {
		t.Fatal("expected error for non-struct type")
	}
}

// --- Field extraction tests ---

func TestLookupStruct_BasicFields(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "User")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	// User has 4 exported fields: ID, Name, Email, Age.
	want := []struct {
		name, accessor string
		exported       bool
	}{
		{"ID", "ID", true},
		{"Name", "Name", true},
		{"Email", "Email", true},
		{"Age", "Age", true},
	}
	if len(info.Fields) != len(want) {
		t.Fatalf("expected %d fields, got %d: %+v", len(want), len(info.Fields), info.Fields)
	}
	for i, w := range want {
		f := info.Fields[i]
		if f.Name != w.name {
			t.Errorf("field[%d]: expected Name=%q, got %q", i, w.name, f.Name)
		}
		if f.Accessor != w.accessor {
			t.Errorf("field[%d]: expected Accessor=%q, got %q", i, w.accessor, f.Accessor)
		}
		if f.Exported != w.exported {
			t.Errorf("field[%d]: expected Exported=%v, got %v", i, w.exported, f.Exported)
		}
		if f.Embedded {
			t.Errorf("field[%d]: should not be embedded", i)
		}
	}
}

func TestLookupStruct_TypeConversion(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "UserDTO")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	// UserDTO.Age should be int64.
	var ageField *StructField
	for i := range info.Fields {
		if info.Fields[i].Name == "Age" {
			ageField = &info.Fields[i]
			break
		}
	}
	if ageField == nil {
		t.Fatal("Age field not found in UserDTO")
	}
	if ageField.Type.String() != "int64" {
		t.Errorf("expected Age type int64, got %s", ageField.Type.String())
	}
}

func TestLookupStruct_StructTags(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "PersonFlat")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	// PersonFlat.Street has map:"Address.Street".
	var streetField *StructField
	for i := range info.Fields {
		if info.Fields[i].Name == "Street" {
			streetField = &info.Fields[i]
			break
		}
	}
	if streetField == nil {
		t.Fatal("Street field not found in PersonFlat")
	}
	if got := streetField.Tag.Get("map"); got != "Address.Street" {
		t.Errorf("expected map tag 'Address.Street', got %q", got)
	}
}

// --- Embedded struct / promoted field tests ---

func TestLookupStruct_EmbeddedPromotedFields(t *testing.T) {
	pkg, err := Load(testdataDir(t, "embedded"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "User")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	// User embeds Base (ID, CreatedAt) and has Name, Email.
	// Expected fields (in order): Base.ID, Base.CreatedAt, Base (embedded), Name, Email.
	type expect struct {
		name, accessor string
		embedded       bool
	}
	want := []expect{
		{"ID", "Base.ID", false},
		{"CreatedAt", "Base.CreatedAt", false},
		{"Base", "Base", true},
		{"Name", "Name", false},
		{"Email", "Email", false},
	}
	if len(info.Fields) != len(want) {
		names := make([]string, len(info.Fields))
		for i, f := range info.Fields {
			names[i] = f.Accessor
		}
		t.Fatalf("expected %d fields, got %d: %v", len(want), len(info.Fields), names)
	}
	for i, w := range want {
		f := info.Fields[i]
		if f.Name != w.name {
			t.Errorf("field[%d]: expected Name=%q, got %q", i, w.name, f.Name)
		}
		if f.Accessor != w.accessor {
			t.Errorf("field[%d]: expected Accessor=%q, got %q", i, w.accessor, f.Accessor)
		}
		if f.Embedded != w.embedded {
			t.Errorf("field[%d] (%s): expected Embedded=%v, got %v", i, w.name, w.embedded, f.Embedded)
		}
		if !f.Exported {
			t.Errorf("field[%d] (%s): expected Exported=true", i, w.name)
		}
	}
}

func TestLookupStruct_PromotedFieldAccessorPath(t *testing.T) {
	pkg, err := Load(testdataDir(t, "embedded"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	info, err := LookupStruct(pkg, "User")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	// The promoted field ID should have Name="ID" (short) and Accessor="Base.ID" (full path).
	var idField *StructField
	for i := range info.Fields {
		if info.Fields[i].Name == "ID" && !info.Fields[i].Embedded {
			idField = &info.Fields[i]
			break
		}
	}
	if idField == nil {
		t.Fatal("promoted ID field not found")
	}
	if idField.Accessor != "Base.ID" {
		t.Errorf("expected Accessor='Base.ID', got %q", idField.Accessor)
	}
}

func TestLookupStruct_NestedStructFieldNotFlattened(t *testing.T) {
	pkg, err := Load(testdataDir(t, "basic"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Person has a non-embedded Address field — its sub-fields should NOT be promoted.
	info, err := LookupStruct(pkg, "Person")
	if err != nil {
		t.Fatalf("LookupStruct: %v", err)
	}

	for _, f := range info.Fields {
		if f.Name == "Street" || f.Name == "City" {
			t.Errorf("non-embedded struct field Address should not have its sub-fields promoted, found %q", f.Name)
		}
	}

	// Should have exactly 2 fields: Name and Address.
	if len(info.Fields) != 2 {
		t.Errorf("expected 2 fields (Name, Address), got %d", len(info.Fields))
	}
}
