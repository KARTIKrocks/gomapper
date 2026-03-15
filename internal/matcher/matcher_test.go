package matcher

import (
	"go/types"
	"reflect"
	"testing"

	"github.com/KARTIKrocks/gomapper/internal/loader"
)

func intType() types.Type             { return types.Typ[types.Int] }
func i64Type() types.Type             { return types.Typ[types.Int64] }
func strType() types.Type             { return types.Typ[types.String] }
func boolType() types.Type            { return types.Typ[types.Bool] }
func ptrTo(t types.Type) types.Type   { return types.NewPointer(t) }
func sliceOf(t types.Type) types.Type { return types.NewSlice(t) }

// field is a shorthand to build a loader.StructField with Name == Accessor.
func field(name string, typ types.Type, exported bool) loader.StructField {
	return loader.StructField{Name: name, Accessor: name, Type: typ, Exported: exported}
}

func TestMatch_ExactNameAndType(t *testing.T) {
	src := &loader.StructInfo{
		Name: "User",
		Fields: []loader.StructField{
			field("ID", intType(), true),
			field("Name", strType(), true),
			field("Email", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "UserDTO",
		Fields: []loader.StructField{
			field("ID", intType(), true),
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(result.Mappings))
	}
	if result.Mappings[0].SrcAccessor != "ID" || result.Mappings[0].DstField != "ID" {
		t.Errorf("unexpected mapping[0]: %+v", result.Mappings[0])
	}
	if result.Mappings[1].SrcAccessor != "Name" || result.Mappings[1].DstField != "Name" {
		t.Errorf("unexpected mapping[1]: %+v", result.Mappings[1])
	}
	if len(result.Unmapped) != 0 {
		t.Errorf("expected no unmapped, got %d", len(result.Unmapped))
	}
}

func TestMatch_ConvertibleType(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Age", intType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Age", i64Type(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.NeedsConv {
		t.Error("expected NeedsConv=true")
	}
	if m.ConvType != "int64" {
		t.Errorf("expected ConvType=int64, got %s", m.ConvType)
	}
}

func TestMatch_UnmappedField(t *testing.T) {
	src := &loader.StructInfo{
		Name:   "Src",
		Fields: []loader.StructField{},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Missing", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unmapped) != 1 {
		t.Fatalf("expected 1 unmapped, got %d", len(result.Unmapped))
	}
	if result.Unmapped[0].Name != "Missing" {
		t.Errorf("unexpected unmapped: %+v", result.Unmapped[0])
	}
}

func TestMatch_StrictMode(t *testing.T) {
	src := &loader.StructInfo{
		Name:   "Src",
		Fields: []loader.StructField{},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Required", strType(), true),
		},
	}

	_, err := Match(src, dst, Config{Strict: true})
	if err == nil {
		t.Fatal("expected error in strict mode")
	}
}

func TestMatch_SkipsUnexported(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("id", intType(), false),
			field("Name", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("id", intType(), false),
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping (only exported), got %d", len(result.Mappings))
	}
}

func TestMatch_DstTagOverride(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("FullName", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			{Name: "Name", Accessor: "Name", Type: strType(), Exported: true, Tag: reflect.StructTag(`map:"FullName"`)},
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	if result.Mappings[0].SrcAccessor != "FullName" {
		t.Errorf("expected SrcAccessor=FullName, got %s", result.Mappings[0].SrcAccessor)
	}
}

func TestMatch_SrcTagTarget(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			{Name: "FullName", Accessor: "FullName", Type: strType(), Exported: true, Tag: reflect.StructTag(`map:"Name"`)},
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	if result.Mappings[0].SrcAccessor != "FullName" {
		t.Errorf("expected SrcAccessor=FullName, got %s", result.Mappings[0].SrcAccessor)
	}
}

func TestMatch_SkipsEmbeddedField(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			{Name: "Base", Accessor: "Base", Type: strType(), Exported: true, Embedded: true},
			field("Name", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
}

func TestMatch_IncompatibleTypesAreUnmapped(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Value", boolType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Value", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	// bool→string: incompatible types, should be unmapped (not silently generate broken code)
	if len(result.Mappings) != 0 {
		t.Fatalf("expected 0 mappings for incompatible types, got %d", len(result.Mappings))
	}
	if len(result.Unmapped) != 1 {
		t.Fatalf("expected 1 unmapped, got %d", len(result.Unmapped))
	}
	if result.Unmapped[0].Name != "Value" {
		t.Errorf("expected unmapped field Value, got %s", result.Unmapped[0].Name)
	}
}

func TestMatch_SkipTagDash_Src(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Name", strType(), true),
			{Name: "Secret", Accessor: "Secret", Type: strType(), Exported: true, Tag: reflect.StructTag(`map:"-"`)},
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
			field("Secret", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping (Secret skipped), got %d", len(result.Mappings))
	}
	if result.Mappings[0].DstField != "Name" {
		t.Errorf("expected mapping for Name, got %s", result.Mappings[0].DstField)
	}
	if len(result.Unmapped) != 1 || result.Unmapped[0].Name != "Secret" {
		t.Errorf("expected Secret unmapped, got %+v", result.Unmapped)
	}
}

func TestMatch_SkipTagDash_Dst(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Name", strType(), true),
			field("Internal", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
			{Name: "Internal", Accessor: "Internal", Type: strType(), Exported: true, Tag: reflect.StructTag(`map:"-"`)},
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping (Internal skipped on dst), got %d", len(result.Mappings))
	}
	// Internal should NOT appear in unmapped either — it was explicitly skipped.
	if len(result.Unmapped) != 0 {
		t.Errorf("expected no unmapped (dst skip), got %+v", result.Unmapped)
	}
}

func TestMatch_PromotedFieldByShortName(t *testing.T) {
	// Source has an embedded struct with promoted field "Street"
	// whose accessor is "Address.Street"
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			{Name: "Street", Accessor: "Address.Street", Type: strType(), Exported: true},
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Street", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	if result.Mappings[0].SrcAccessor != "Address.Street" {
		t.Errorf("expected SrcAccessor=Address.Street, got %s", result.Mappings[0].SrcAccessor)
	}
}

func TestMatch_PointerDeref(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Name", ptrTo(strType()), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.Deref {
		t.Error("expected Deref=true")
	}
	if m.NeedsConv {
		t.Error("expected NeedsConv=false")
	}
}

func TestMatch_PointerAddrOf(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", ptrTo(strType()), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.AddrOf {
		t.Error("expected AddrOf=true")
	}
}

func TestMatch_PointerDerefConvert(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Age", ptrTo(intType()), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Age", i64Type(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.Deref {
		t.Error("expected Deref=true")
	}
	if !m.NeedsConv {
		t.Error("expected NeedsConv=true")
	}
	if m.ConvType != "int64" {
		t.Errorf("expected ConvType=int64, got %s", m.ConvType)
	}
}

func TestMatch_DerefSetsDstTypeName(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Name", ptrTo(strType()), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.Deref {
		t.Error("expected Deref=true")
	}
	if m.DstTypeName != "string" {
		t.Errorf("expected DstTypeName=string, got %s", m.DstTypeName)
	}
}

func TestMatch_DerefConvertSetsDstTypeName(t *testing.T) {
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Age", ptrTo(intType()), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Age", i64Type(), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.Deref {
		t.Error("expected Deref=true")
	}
	if m.DstTypeName != "int64" {
		t.Errorf("expected DstTypeName=int64, got %s", m.DstTypeName)
	}
}

func TestMatch_StructMapping(t *testing.T) {
	// Create named struct types Address and AddressDTO.
	pkg := types.NewPackage("example.com/pkg", "pkg")
	addrType := types.NewNamed(types.NewTypeName(0, pkg, "Address", nil), types.NewStruct(nil, nil), nil)
	addrDTOType := types.NewNamed(types.NewTypeName(0, pkg, "AddressDTO", nil), types.NewStruct(nil, nil), nil)

	src := &loader.StructInfo{
		Name: "Order",
		Fields: []loader.StructField{
			field("Addr", addrType, true),
		},
	}
	dst := &loader.StructInfo{
		Name: "OrderDTO",
		Fields: []loader.StructField{
			field("Addr", addrDTOType, true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.IsStructMap {
		t.Error("expected IsStructMap=true")
	}
	if m.StructSrc != "Address" {
		t.Errorf("expected StructSrc=Address, got %s", m.StructSrc)
	}
	if m.StructDst != "AddressDTO" {
		t.Errorf("expected StructDst=AddressDTO, got %s", m.StructDst)
	}
}

func TestMatch_IdenticalStructNotStructMap(t *testing.T) {
	// Same named type on both sides — should be assignable, not struct-map.
	pkg := types.NewPackage("example.com/pkg", "pkg")
	addrType := types.NewNamed(types.NewTypeName(0, pkg, "Address", nil), types.NewStruct(nil, nil), nil)

	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Addr", addrType, true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Addr", addrType, true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if m.IsStructMap {
		t.Error("expected IsStructMap=false for identical types")
	}
}

func TestMatch_CaseInsensitiveMatch(t *testing.T) {
	src := &loader.StructInfo{
		Name: "CISrc",
		Fields: []loader.StructField{
			field("UserName", strType(), true),
			field("EMail", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "CIDst",
		Fields: []loader.StructField{
			field("Username", strType(), true),
			field("Email", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{CaseInsensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(result.Mappings))
	}
	if result.Mappings[0].SrcAccessor != "UserName" || result.Mappings[0].DstField != "Username" {
		t.Errorf("unexpected mapping[0]: %+v", result.Mappings[0])
	}
	if result.Mappings[1].SrcAccessor != "EMail" || result.Mappings[1].DstField != "Email" {
		t.Errorf("unexpected mapping[1]: %+v", result.Mappings[1])
	}
	if len(result.Unmapped) != 0 {
		t.Errorf("expected no unmapped, got %+v", result.Unmapped)
	}
}

func TestMatch_CaseInsensitiveExactTakesPriority(t *testing.T) {
	// Both "Name" (exact) and "name" (ci) exist in source.
	// Exact match should always win.
	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("name", strType(), false), // unexported, won't match
			field("Name", strType(), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Name", strType(), true),
		},
	}

	result, err := Match(src, dst, Config{CaseInsensitive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	// Should use exact match "Name", not fall through to CI.
	if result.Mappings[0].SrcAccessor != "Name" {
		t.Errorf("expected exact match SrcAccessor=Name, got %s", result.Mappings[0].SrcAccessor)
	}
}

func TestMatch_SliceMapping(t *testing.T) {
	// Create named types to simulate []Item → []ItemDTO
	itemPkg := types.NewPackage("example.com/pkg", "pkg")
	itemType := types.NewNamed(types.NewTypeName(0, itemPkg, "Item", nil), types.NewStruct(nil, nil), nil)
	itemDTOType := types.NewNamed(types.NewTypeName(0, itemPkg, "ItemDTO", nil), types.NewStruct(nil, nil), nil)

	src := &loader.StructInfo{
		Name: "Src",
		Fields: []loader.StructField{
			field("Items", sliceOf(itemType), true),
		},
	}
	dst := &loader.StructInfo{
		Name: "Dst",
		Fields: []loader.StructField{
			field("Items", sliceOf(itemDTOType), true),
		},
	}

	result, err := Match(src, dst, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(result.Mappings))
	}
	m := result.Mappings[0]
	if !m.IsSliceMap {
		t.Error("expected IsSliceMap=true")
	}
	if m.SliceSrc != "Item" {
		t.Errorf("expected SliceSrc=Item, got %s", m.SliceSrc)
	}
	if m.SliceDst != "ItemDTO" {
		t.Errorf("expected SliceDst=ItemDTO, got %s", m.SliceDst)
	}
}
