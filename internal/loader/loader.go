// Package loader provides Go package loading and struct type resolution.
package loader

import (
	"fmt"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/packages"
)

// StructField represents a single field extracted from a Go struct.
type StructField struct {
	Name     string // short field name used for matching (e.g. "Street")
	Accessor string // full accessor path from root (e.g. "Address.Street")
	Type     types.Type
	Tag      reflect.StructTag
	Exported bool
	Embedded bool
}

// StructInfo holds the resolved struct type information.
type StructInfo struct {
	Name    string
	PkgPath string
	Fields  []StructField // includes promoted fields from embedded structs
}

// Load parses the Go package at the given directory and returns the package object.
func Load(dir string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes,
		Dir:  dir,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("loading package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", dir)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package errors: %v", pkg.Errors[0])
	}
	return pkg, nil
}

// LookupStruct finds a named struct type in the package.
func LookupStruct(pkg *packages.Package, name string) (*StructInfo, error) {
	obj := pkg.Types.Scope().Lookup(name)
	if obj == nil {
		return nil, fmt.Errorf("type %q not found in package %s", name, pkg.PkgPath)
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%q is not a named type", name)
	}
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%q is not a struct type", name)
	}

	fields := flattenFields(st, "")
	return &StructInfo{
		Name:    name,
		PkgPath: pkg.PkgPath,
		Fields:  fields,
	}, nil
}

// flattenFields extracts all fields including promoted fields from embedded structs.
// prefix is used to track the accessor path for embedded fields.
func flattenFields(st *types.Struct, prefix string) []StructField {
	var fields []StructField
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))
		accessor := prefix + f.Name()

		if f.Embedded() {
			// Resolve the underlying struct of the embedded type.
			embedded := f.Type().Underlying()
			if ptr, ok := embedded.(*types.Pointer); ok {
				embedded = ptr.Elem().Underlying()
			}
			if est, ok := embedded.(*types.Struct); ok {
				promoted := flattenFields(est, accessor+".")
				fields = append(fields, promoted...)
			}
			// Also add the embedded field itself.
			fields = append(fields, StructField{
				Name:     f.Name(),
				Accessor: accessor,
				Type:     f.Type(),
				Tag:      tag,
				Exported: f.Exported(),
				Embedded: true,
			})
			continue
		}

		fields = append(fields, StructField{
			Name:     f.Name(),
			Accessor: accessor,
			Type:     f.Type(),
			Tag:      tag,
			Exported: f.Exported(),
		})
	}
	return fields
}
