// Package matcher implements field matching between source and destination structs.
package matcher

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/KARTIKrocks/gomapper/internal/loader"
)

// FieldMapping represents a matched pair of fields between source and destination.
type FieldMapping struct {
	SrcAccessor       string // e.g. "src.Name" or "src.Address.Street"
	DstField          string // destination field name, e.g. "Name"
	NeedsConv         bool   // true if type conversion is needed
	ConvType          string // destination type string for conversion, e.g. "int64"
	Deref             bool   // prepend * to accessor (src is *T, dst is T)
	DstTypeName       string // destination type name for nil-safe declarations (set when Deref=true)
	AddrOf            bool   // prepend & to accessor (src is T, dst is *T)
	IsSliceMap        bool   // use mapper.MapSlice for []T → []U
	SliceSrc          string // source element base type name (for MapXToY function name)
	SliceDst          string // destination element base type name (for MapXToY function name)
	SliceDstFull      string // full destination element type (for make[], may include *)
	IsStructMap       bool   // nested struct mapping (different named struct types)
	StructSrc         string // source struct type name
	StructDst         string // destination struct type name
	SliceElemDeref    bool   // dereference source element in slice loop
	SliceElemAddrOf   bool   // addr-of result in slice loop
	SliceElemConv     bool   // type-convert element in slice loop (no map function)
	SliceElemConvType string // conversion type for slice element
}

// UnmappedField represents a destination field that could not be matched.
type UnmappedField struct {
	Name string
	Type string
}

// NestedDstAssignment represents a source field mapped to a nested destination
// path via a source-side tag like `map:"Address.Street"`.
// Generated as: result.DstPath = src.SrcAccessor (with optional conversion).
type NestedDstAssignment struct {
	DstPath     string // e.g., "Address.Street"
	SrcAccessor string // e.g., "Street"
	NeedsConv   bool
	ConvType    string
}

// Result holds the matching outcome for a struct pair.
type Result struct {
	Mappings             []FieldMapping
	Unmapped             []UnmappedField
	NestedDstAssignments []NestedDstAssignment
}

// Config controls matching behavior.
type Config struct {
	TagKey          string // struct tag key for field renaming (default: "map")
	Strict          bool   // fail if any destination field is unmapped
	Verbose         bool   // print matching decisions
	CaseInsensitive bool   // match fields by case-insensitive name comparison
}

// sourceIndices holds lookup maps for source fields.
type sourceIndices struct {
	byName     map[string]loader.StructField
	byAccessor map[string]loader.StructField
	byTag      map[string]loader.StructField
}

// buildSourceIndices builds source field lookup maps (only exported, non-embedded).
func buildSourceIndices(src *loader.StructInfo, tagKey string) sourceIndices {
	idx := sourceIndices{
		byName:     make(map[string]loader.StructField),
		byAccessor: make(map[string]loader.StructField),
		byTag:      make(map[string]loader.StructField),
	}
	for _, f := range src.Fields {
		if !f.Exported || f.Embedded {
			continue
		}
		tagVal := f.Tag.Get(tagKey)
		if tagVal == "-" {
			continue
		}
		idx.byName[f.Name] = f
		idx.byAccessor[f.Accessor] = f
		if tagVal != "" {
			idx.byTag[tagVal] = f
		}
	}
	return idx
}

// matchDstFields matches destination fields against source, populating mappings and unmapped.
func matchDstFields(src, dst *loader.StructInfo, idx sourceIndices, cfg Config) ([]FieldMapping, []UnmappedField) {
	var mappings []FieldMapping
	var unmapped []UnmappedField
	for _, df := range dst.Fields {
		if !df.Exported || df.Embedded {
			continue
		}
		if df.Tag.Get(cfg.TagKey) == "-" {
			continue
		}

		mapping, ok := matchField(df, idx.byName, idx.byAccessor, idx.byTag, cfg)
		if ok {
			if cfg.Verbose {
				fmt.Printf("  %s.%s → %s.%s\n", dst.Name, df.Name, src.Name, mapping.SrcAccessor)
			}
			mappings = append(mappings, mapping)
		} else {
			if cfg.Verbose {
				fmt.Printf("  %s.%s → (unmapped)\n", dst.Name, df.Name)
			}
			unmapped = append(unmapped, UnmappedField{
				Name: df.Name,
				Type: localTypeName(df.Type),
			})
		}
	}
	return mappings, unmapped
}

// processNestedDstTags handles source-side tags with dot notation targeting nested dst paths.
func processNestedDstTags(
	src, dst *loader.StructInfo,
	idx sourceIndices,
	mappings []FieldMapping,
	cfg Config,
) []NestedDstAssignment {
	var assignments []NestedDstAssignment
	for tagVal, sf := range idx.byTag {
		if !strings.Contains(tagVal, ".") {
			continue
		}

		// Skip if this source field was already consumed in the main loop.
		if isAccessorUsed(mappings, sf.Accessor) {
			continue
		}

		parts := strings.SplitN(tagVal, ".", 2)
		parentName := parts[0]

		parentField := findDstParentField(dst, parentName)
		if parentField == nil {
			continue
		}

		assignment := NestedDstAssignment{
			DstPath:     tagVal,
			SrcAccessor: sf.Accessor,
		}

		if childType, ok := resolveNestedFieldType(parentField.Type, parts[1]); ok {
			if !types.AssignableTo(sf.Type, childType) && types.ConvertibleTo(sf.Type, childType) {
				assignment.NeedsConv = true
				assignment.ConvType = localTypeName(childType)
			}
		}

		assignments = append(assignments, assignment)
		if cfg.Verbose {
			fmt.Printf("  %s.%s → %s.%s (nested dst tag)\n", dst.Name, tagVal, src.Name, sf.Accessor)
		}
	}
	return assignments
}

// isAccessorUsed checks if a source accessor is already used in mappings.
func isAccessorUsed(mappings []FieldMapping, accessor string) bool {
	for _, m := range mappings {
		if m.SrcAccessor == accessor {
			return true
		}
	}
	return false
}

// findDstParentField finds an exported, non-embedded destination field by name.
func findDstParentField(dst *loader.StructInfo, name string) *loader.StructField {
	for i := range dst.Fields {
		if dst.Fields[i].Name == name && dst.Fields[i].Exported && !dst.Fields[i].Embedded {
			return &dst.Fields[i]
		}
	}
	return nil
}

// filterUnmappedByCoveredParents removes parent fields from unmapped when covered by nested assignments.
func filterUnmappedByCoveredParents(unmapped []UnmappedField, assignments []NestedDstAssignment) []UnmappedField {
	if len(assignments) == 0 {
		return unmapped
	}
	coveredParents := make(map[string]bool)
	for _, na := range assignments {
		parent := strings.SplitN(na.DstPath, ".", 2)[0]
		coveredParents[parent] = true
	}
	var filtered []UnmappedField
	for _, u := range unmapped {
		if !coveredParents[u.Name] {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// Match performs field matching from source to destination struct.
func Match(src, dst *loader.StructInfo, cfg Config) (*Result, error) {
	if cfg.TagKey == "" {
		cfg.TagKey = "map"
	}

	idx := buildSourceIndices(src, cfg.TagKey)

	mappings, unmapped := matchDstFields(src, dst, idx, cfg)

	assignments := processNestedDstTags(src, dst, idx, mappings, cfg)

	unmapped = filterUnmappedByCoveredParents(unmapped, assignments)

	if cfg.Strict && len(unmapped) > 0 {
		var names []string
		for _, u := range unmapped {
			names = append(names, u.Name)
		}
		return nil, fmt.Errorf("strict mode: unmapped destination fields: %s", strings.Join(names, ", "))
	}

	return &Result{
		Mappings:             mappings,
		Unmapped:             unmapped,
		NestedDstAssignments: assignments,
	}, nil
}

// matchField tries to find a source field for the given destination field.
func matchField(
	dst loader.StructField,
	srcByName map[string]loader.StructField,
	srcByAccessor map[string]loader.StructField,
	srcByTag map[string]loader.StructField,
	cfg Config,
) (FieldMapping, bool) {
	// 1. Check destination field's map tag → explicit source path
	if tagVal := dst.Tag.Get(cfg.TagKey); tagVal != "" && tagVal != "-" {
		// Try as full accessor path first (e.g. "Address.Street")
		if sf, ok := srcByAccessor[tagVal]; ok {
			return makeMapping(sf, dst, sf.Accessor)
		}
		// Try as short name
		if sf, ok := srcByName[tagVal]; ok {
			return makeMapping(sf, dst, sf.Accessor)
		}
		// Treat as raw accessor path — trust the user's tag.
		return FieldMapping{
			SrcAccessor: tagVal,
			DstField:    dst.Name,
		}, true
	}

	// 2. Check if any source field has a tag targeting this destination field name
	if sf, ok := srcByTag[dst.Name]; ok {
		return makeMapping(sf, dst, sf.Accessor)
	}

	// 3. Match by exact name
	if sf, ok := srcByName[dst.Name]; ok {
		return makeMapping(sf, dst, sf.Accessor)
	}

	// 4. Case-insensitive name match (if enabled)
	if cfg.CaseInsensitive {
		for name, sf := range srcByName {
			if strings.EqualFold(name, dst.Name) {
				return makeMapping(sf, dst, sf.Accessor)
			}
		}
	}

	return FieldMapping{}, false
}

// areDifferentNamedStructs checks if both types are different named struct types.
func areDifferentNamedStructs(a, b types.Type) bool {
	aNamed, ok := a.(*types.Named)
	if !ok {
		return false
	}
	bNamed, ok := b.(*types.Named)
	if !ok {
		return false
	}
	if _, ok := aNamed.Underlying().(*types.Struct); !ok {
		return false
	}
	if _, ok := bNamed.Underlying().(*types.Struct); !ok {
		return false
	}
	return !types.Identical(a, b)
}

// tryStructMapping checks for nested struct mapping (StructA → StructB).
func tryStructMapping(m *FieldMapping, srcType, dstType types.Type) bool {
	if areDifferentNamedStructs(srcType, dstType) {
		m.IsStructMap = true
		m.StructSrc = localTypeName(srcType)
		m.StructDst = localTypeName(dstType)
		return true
	}
	return false
}

// tryDerefMapping handles pointer-to-value mapping (*T → U).
func tryDerefMapping(m *FieldMapping, srcType, dstType types.Type) bool {
	srcPtr, ok := srcType.(*types.Pointer)
	if !ok {
		return false
	}
	elem := srcPtr.Elem()

	if types.AssignableTo(elem, dstType) {
		m.Deref = true
		m.DstTypeName = localTypeName(dstType)
		return true
	}

	if areDifferentNamedStructs(elem, dstType) {
		m.Deref = true
		m.DstTypeName = localTypeName(dstType)
		m.IsStructMap = true
		m.StructSrc = localTypeName(elem)
		m.StructDst = localTypeName(dstType)
		return true
	}

	if types.ConvertibleTo(elem, dstType) {
		m.Deref = true
		m.DstTypeName = localTypeName(dstType)
		m.NeedsConv = true
		m.ConvType = localTypeName(dstType)
		return true
	}

	return false
}

// tryAddrOfMapping handles value-to-pointer mapping (T → *U).
func tryAddrOfMapping(m *FieldMapping, srcType, dstType types.Type) bool {
	dstPtr, ok := dstType.(*types.Pointer)
	if !ok {
		return false
	}
	elem := dstPtr.Elem()

	if types.AssignableTo(srcType, elem) {
		m.AddrOf = true
		return true
	}

	if areDifferentNamedStructs(srcType, elem) {
		m.AddrOf = true
		m.IsStructMap = true
		m.StructSrc = localTypeName(srcType)
		m.StructDst = localTypeName(elem)
		return true
	}

	if types.ConvertibleTo(srcType, elem) {
		m.AddrOf = true
		m.NeedsConv = true
		m.ConvType = localTypeName(elem)
		return true
	}

	return false
}

// trySliceMapping handles slice mapping []T → []U where element types differ.
func trySliceMapping(m *FieldMapping, srcType, dstType types.Type) bool {
	srcSlice, ok := srcType.(*types.Slice)
	if !ok {
		return false
	}
	dstSlice, ok := dstType.(*types.Slice)
	if !ok {
		return false
	}

	srcElem := srcSlice.Elem()
	dstElem := dstSlice.Elem()
	if types.Identical(srcElem, dstElem) {
		return false
	}

	m.IsSliceMap = true
	m.SliceDstFull = localTypeName(dstElem)

	srcBase, srcIsPtr := unwrapPointer(srcElem)
	dstBase, dstIsPtr := unwrapPointer(dstElem)
	m.SliceElemDeref = srcIsPtr && !dstIsPtr
	m.SliceElemAddrOf = !srcIsPtr && dstIsPtr

	// Both pointers to different named types: deref + map + addr-of.
	if srcIsPtr && dstIsPtr {
		m.SliceElemDeref = true
		m.SliceElemAddrOf = true
	}

	// Check if base types are different named structs → struct mapping.
	if areDifferentNamedStructs(srcBase, dstBase) {
		m.SliceSrc = localTypeName(srcBase)
		m.SliceDst = localTypeName(dstBase)
		return true
	}

	// Same base type, only pointer difference → inline deref/addr-of.
	if types.Identical(srcBase, dstBase) {
		m.SliceDst = localTypeName(dstBase)
		return true
	}

	// Convertible base types → inline conversion.
	if types.ConvertibleTo(srcBase, dstBase) {
		m.SliceElemConv = true
		m.SliceElemConvType = localTypeName(dstBase)
		m.SliceDst = localTypeName(dstBase)
		return true
	}

	// Fallback.
	m.SliceSrc = localTypeName(srcBase)
	m.SliceDst = localTypeName(dstBase)
	return true
}

// makeMapping creates a FieldMapping, detecting whether type conversion is needed.
// Returns false if types are incompatible (neither assignable nor convertible).
func makeMapping(src, dst loader.StructField, accessor string) (FieldMapping, bool) {
	m := FieldMapping{
		SrcAccessor: accessor,
		DstField:    dst.Name,
	}

	if types.AssignableTo(src.Type, dst.Type) {
		return m, true
	}

	if tryStructMapping(&m, src.Type, dst.Type) {
		return m, true
	}

	if types.ConvertibleTo(src.Type, dst.Type) {
		m.NeedsConv = true
		m.ConvType = localTypeName(dst.Type)
		return m, true
	}

	if tryDerefMapping(&m, src.Type, dst.Type) {
		return m, true
	}

	if tryAddrOfMapping(&m, src.Type, dst.Type) {
		return m, true
	}

	if trySliceMapping(&m, src.Type, dst.Type) {
		return m, true
	}

	// Types are incompatible — treat as unmapped so we don't generate broken code.
	return FieldMapping{}, false
}

// resolveNestedFieldType resolves a dot-separated field path within a type.
// e.g., given type Address{Street string, City string} and path "Street",
// returns the type of the Street field.
func resolveNestedFieldType(t types.Type, path string) (types.Type, bool) {
	parts := strings.Split(path, ".")
	current := t
	for _, part := range parts {
		// Unwrap named type to get underlying struct.
		underlying := current.Underlying()
		if ptr, ok := underlying.(*types.Pointer); ok {
			underlying = ptr.Elem().Underlying()
		}
		st, ok := underlying.(*types.Struct)
		if !ok {
			return nil, false
		}
		found := false
		for i := 0; i < st.NumFields(); i++ {
			if st.Field(i).Name() == part {
				current = st.Field(i).Type()
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
	}
	return current, true
}

// unwrapPointer strips one layer of pointer from a type.
// Returns the element type and whether it was a pointer.
func unwrapPointer(t types.Type) (types.Type, bool) {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem(), true
	}
	return t, false
}

// localTypeName returns the short (unqualified) name for a type.
// For named types it strips the package path; for others it uses the full string.
// Handles pointer and slice wrappers recursively.
func localTypeName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	if ptr, ok := t.(*types.Pointer); ok {
		return "*" + localTypeName(ptr.Elem())
	}
	if sl, ok := t.(*types.Slice); ok {
		return "[]" + localTypeName(sl.Elem())
	}
	return t.String()
}
