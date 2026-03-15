package generator

// registryFieldExpr renders a field value in register/both modes.
// Simple slice maps use mapper.MapSlice; complex ones reference pre-computed vars.
const registryFieldExpr = `{{- if and .IsSliceMap .SliceSrc (not .SliceElemDeref) (not .SliceElemAddrOf)}}` +
	`mapper.MapSlice[{{.SliceSrc}}, {{.SliceDst}}](src.{{.SrcAccessor}})` +
	`{{- else if .IsSliceMap}}` +
	`_{{.DstField}}` +
	`{{- else if and .IsStructMap .AddrOf}}` +
	`&_{{.DstField}}` +
	`{{- else if and .IsStructMap .Deref (not $.NilSafe)}}` +
	`mapper.Map[{{.StructDst}}](*src.{{.SrcAccessor}})` +
	`{{- else if and .IsStructMap .Deref $.NilSafe}}` +
	`_{{.DstField}}` +
	`{{- else if .IsStructMap}}` +
	`mapper.Map[{{.StructDst}}](src.{{.SrcAccessor}})` +
	`{{- else if .IsSliceMap}}` +
	`_{{.DstField}}` +
	`{{- else if and .Deref $.NilSafe}}` +
	`_{{.DstField}}` +
	`{{- else if and .NeedsConv .Deref}}` +
	`{{.ConvType}}(*src.{{.SrcAccessor}})` +
	`{{- else if .NeedsConv}}` +
	`{{.ConvType}}(src.{{.SrcAccessor}})` +
	`{{- else if .Deref}}` +
	`*src.{{.SrcAccessor}}` +
	`{{- else if .AddrOf}}` +
	`&src.{{.SrcAccessor}}` +
	`{{- else}}` +
	`src.{{.SrcAccessor}}` +
	`{{- end}}`

// pureFieldExpr renders a field value in func mode.
// Slice fields reference a pre-computed local variable (no mapper dependency).
const pureFieldExpr = `{{- if and .IsStructMap .IsSliceMap}}` +
	`_{{.DstField}}` +
	`{{- else if and .IsStructMap .AddrOf}}` +
	`&_{{.DstField}}` +
	`{{- else if and .IsStructMap .Deref (not $.NilSafe)}}` +
	`Map{{.StructSrc}}To{{.StructDst}}(*src.{{.SrcAccessor}})` +
	`{{- else if and .IsStructMap .Deref $.NilSafe}}` +
	`_{{.DstField}}` +
	`{{- else if .IsStructMap}}` +
	`Map{{.StructSrc}}To{{.StructDst}}(src.{{.SrcAccessor}})` +
	`{{- else if .IsSliceMap}}` +
	`_{{.DstField}}` +
	`{{- else if and .Deref $.NilSafe}}` +
	`_{{.DstField}}` +
	`{{- else if and .NeedsConv .Deref}}` +
	`{{.ConvType}}(*src.{{.SrcAccessor}})` +
	`{{- else if .NeedsConv}}` +
	`{{.ConvType}}(src.{{.SrcAccessor}})` +
	`{{- else if .Deref}}` +
	`*src.{{.SrcAccessor}}` +
	`{{- else if .AddrOf}}` +
	`&src.{{.SrcAccessor}}` +
	`{{- else}}` +
	`src.{{.SrcAccessor}}` +
	`{{- end}}`

// nilSafeBlock generates variable declarations and nil checks for pointer dereferences.
const nilSafeBlock = `{{- if $.NilSafe}}
{{- range .Mappings}}
{{- if and .Deref .IsStructMap}}
	var _{{.DstField}} {{.DstTypeName}}
	if src.{{.SrcAccessor}} != nil {
		_{{.DstField}} = Map{{.StructSrc}}To{{.StructDst}}(*src.{{.SrcAccessor}})
	}
{{- else if .Deref}}
	var _{{.DstField}} {{.DstTypeName}}
	if src.{{.SrcAccessor}} != nil {
		_{{.DstField}} = {{if .NeedsConv}}{{.ConvType}}(*src.{{.SrcAccessor}}){{else}}*src.{{.SrcAccessor}}{{end}}
	}
{{- end}}
{{- end}}
{{- end}}`

// addrOfStructBlock generates temp variables for addr-of struct mappings.
const addrOfStructBlock = `{{- range .Mappings}}
{{- if and .IsStructMap .AddrOf (not .IsSliceMap)}}
	_{{.DstField}} := Map{{.StructSrc}}To{{.StructDst}}(src.{{.SrcAccessor}})
{{- end}}
{{- end}}`

// sliceLoopExpr renders the loop body expression for slice mappings.
// Handles: struct map, deref, addr-of, conversion, and combinations.
const sliceLoopExpr = `{{- if and .SliceSrc .SliceElemDeref .SliceElemAddrOf}}` +
	// []*StructA → []*StructB: deref, map, addr-of
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_mapped := Map{{.SliceSrc}}To{{.SliceDst}}(*_v)
			_{{.DstField}}[_i] = &_mapped
		}` +
	`{{- else}}` +
	`_mapped := Map{{.SliceSrc}}To{{.SliceDst}}(*_v)
		_{{.DstField}}[_i] = &_mapped` +
	`{{- end}}` +
	`{{- else if and .SliceSrc .SliceElemDeref}}` +
	// []*StructA → []StructB: deref, map
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = Map{{.SliceSrc}}To{{.SliceDst}}(*_v)
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = Map{{.SliceSrc}}To{{.SliceDst}}(*_v)` +
	`{{- end}}` +
	`{{- else if and .SliceSrc .SliceElemAddrOf}}` +
	// []StructA → []*StructB: map, addr-of
	`_mapped := Map{{.SliceSrc}}To{{.SliceDst}}(_v)
		_{{.DstField}}[_i] = &_mapped` +
	`{{- else if .SliceSrc}}` +
	// []StructA → []StructB: just map
	`_{{.DstField}}[_i] = Map{{.SliceSrc}}To{{.SliceDst}}(_v)` +
	`{{- else if and .SliceElemConv .SliceElemDeref}}` +
	// []*int → []int64: deref + convert
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = {{.SliceElemConvType}}(*_v)
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = {{.SliceElemConvType}}(*_v)` +
	`{{- end}}` +
	`{{- else if .SliceElemConv}}` +
	// []int → []int64: convert
	`_{{.DstField}}[_i] = {{.SliceElemConvType}}(_v)` +
	`{{- else if .SliceElemDeref}}` +
	// []*string → []string: deref
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = *_v
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = *_v` +
	`{{- end}}` +
	`{{- else if .SliceElemAddrOf}}` +
	// []string → []*string: addr-of
	`_{{.DstField}}[_i] = &_v` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = _v` +
	`{{- end}}`

// sliceLoopExprRegistry renders the loop body for register mode slice mappings.
const sliceLoopExprRegistry = `{{- if and .SliceSrc .SliceElemDeref .SliceElemAddrOf}}` +
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_mapped := mapper.Map[{{.SliceDst}}](*_v)
			_{{.DstField}}[_i] = &_mapped
		}` +
	`{{- else}}` +
	`_mapped := mapper.Map[{{.SliceDst}}](*_v)
		_{{.DstField}}[_i] = &_mapped` +
	`{{- end}}` +
	`{{- else if and .SliceSrc .SliceElemDeref}}` +
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = mapper.Map[{{.SliceDst}}](*_v)
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = mapper.Map[{{.SliceDst}}](*_v)` +
	`{{- end}}` +
	`{{- else if and .SliceSrc .SliceElemAddrOf}}` +
	`_mapped := mapper.Map[{{.SliceDst}}](_v)
		_{{.DstField}}[_i] = &_mapped` +
	`{{- else if .SliceSrc}}` +
	`_{{.DstField}}[_i] = mapper.Map[{{.SliceDst}}](_v)` +
	`{{- else if and .SliceElemConv .SliceElemDeref}}` +
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = {{.SliceElemConvType}}(*_v)
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = {{.SliceElemConvType}}(*_v)` +
	`{{- end}}` +
	`{{- else if .SliceElemConv}}` +
	`_{{.DstField}}[_i] = {{.SliceElemConvType}}(_v)` +
	`{{- else if .SliceElemDeref}}` +
	`{{- if $.NilSafe}}` +
	`if _v != nil {
			_{{.DstField}}[_i] = *_v
		}` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = *_v` +
	`{{- end}}` +
	`{{- else if .SliceElemAddrOf}}` +
	`_{{.DstField}}[_i] = &_v` +
	`{{- else}}` +
	`_{{.DstField}}[_i] = _v` +
	`{{- end}}`

// nilSafeBlockRegistry is the nil-safe block for register mode (uses mapper.Map instead of MapXToY).
const nilSafeBlockRegistry = `{{- if $.NilSafe}}
{{- range .Mappings}}
{{- if and .Deref .IsStructMap}}
	var _{{.DstField}} {{.DstTypeName}}
	if src.{{.SrcAccessor}} != nil {
		_{{.DstField}} = mapper.Map[{{.StructDst}}](*src.{{.SrcAccessor}})
	}
{{- else if .Deref}}
	var _{{.DstField}} {{.DstTypeName}}
	if src.{{.SrcAccessor}} != nil {
		_{{.DstField}} = {{if .NeedsConv}}{{.ConvType}}(*src.{{.SrcAccessor}}){{else}}*src.{{.SrcAccessor}}{{end}}
	}
{{- end}}
{{- end}}
{{- end}}`

// addrOfStructBlockRegistry generates temp variables for addr-of struct mappings in register mode.
const addrOfStructBlockRegistry = `{{- range .Mappings}}
{{- if and .IsStructMap .AddrOf (not .IsSliceMap)}}
	_{{.DstField}} := mapper.Map[{{.StructDst}}](src.{{.SrcAccessor}})
{{- end}}
{{- end}}`

// nestedDstAssignBlock generates post-assignment statements for source-side
// dot-notation tags (e.g., result.Address.Street = src.Street).
const nestedDstAssignBlock = `{{- range .NestedDstAssignments}}
	_result.{{.DstPath}} = {{if .NeedsConv}}{{.ConvType}}(src.{{.SrcAccessor}}){{else}}src.{{.SrcAccessor}}{{end}}
{{- end}}`

const registerTemplate = `// Code generated by gomapper; DO NOT EDIT.

package {{.PkgName}}

import "github.com/KARTIKrocks/mapper"

func init() {
{{- range .Pairs}}
	mapper.Register(func(src {{.SrcType}}) {{.DstType}} {
{{- range .Mappings}}
{{- if and .IsSliceMap (or .SliceElemDeref .SliceElemAddrOf .SliceElemConv (not .SliceSrc))}}
	_{{.DstField}} := make([]{{.SliceDstFull}}, len(src.{{.SrcAccessor}}))
	for _i, _v := range src.{{.SrcAccessor}} {
		` + sliceLoopExprRegistry + `
	}
{{- end}}
{{- end}}
` + nilSafeBlockRegistry + `
` + addrOfStructBlockRegistry + `
{{- if .NestedDstAssignments}}
		_result := {{.DstType}}{
{{- else}}
		return {{.DstType}}{
{{- end}}
{{- range .Mappings}}
			{{.DstField}}: ` + registryFieldExpr + `,
{{- end}}
{{- range .Unmapped}}
			// TODO: unmapped field {{.Name}} ({{.Type}})
{{- end}}
		}
{{- if .NestedDstAssignments}}
` + nestedDstAssignBlock + `
		return _result
{{- end}}
	})
{{end -}}
}
`

const funcTemplate = `// Code generated by gomapper; DO NOT EDIT.

package {{.PkgName}}

{{range .Pairs}}
func Map{{.SrcType}}To{{.DstType}}(src {{.SrcType}}) {{.DstType}} {
{{- range .Mappings}}
{{- if .IsSliceMap}}
	_{{.DstField}} := make([]{{.SliceDstFull}}, len(src.{{.SrcAccessor}}))
	for _i, _v := range src.{{.SrcAccessor}} {
		` + sliceLoopExpr + `
	}
{{- end}}
{{- end}}
` + nilSafeBlock + `
` + addrOfStructBlock + `
{{- if .NestedDstAssignments}}
	_result := {{.DstType}}{
{{- else}}
	return {{.DstType}}{
{{- end}}
{{- range .Mappings}}
		{{.DstField}}: ` + pureFieldExpr + `,
{{- end}}
{{- range .Unmapped}}
		// TODO: unmapped field {{.Name}} ({{.Type}})
{{- end}}
	}
{{- if .NestedDstAssignments}}
` + nestedDstAssignBlock + `
	return _result
{{- end}}
}
{{end -}}
`

const bothTemplate = `// Code generated by gomapper; DO NOT EDIT.

package {{.PkgName}}

import "github.com/KARTIKrocks/mapper"

{{range .Pairs}}
func Map{{.SrcType}}To{{.DstType}}(src {{.SrcType}}) {{.DstType}} {
{{- range .Mappings}}
{{- if .IsSliceMap}}
	_{{.DstField}} := make([]{{.SliceDstFull}}, len(src.{{.SrcAccessor}}))
	for _i, _v := range src.{{.SrcAccessor}} {
		` + sliceLoopExpr + `
	}
{{- end}}
{{- end}}
` + nilSafeBlock + `
` + addrOfStructBlock + `
{{- if .NestedDstAssignments}}
	_result := {{.DstType}}{
{{- else}}
	return {{.DstType}}{
{{- end}}
{{- range .Mappings}}
		{{.DstField}}: ` + pureFieldExpr + `,
{{- end}}
{{- range .Unmapped}}
		// TODO: unmapped field {{.Name}} ({{.Type}})
{{- end}}
	}
{{- if .NestedDstAssignments}}
` + nestedDstAssignBlock + `
	return _result
{{- end}}
}
{{end}}
func init() {
{{- range .Pairs}}
	mapper.Register(Map{{.SrcType}}To{{.DstType}})
{{- end}}
}
`
