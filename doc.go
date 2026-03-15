// Command gomapper generates type-safe struct mapping functions from Go struct definitions.
//
// gomapper parses Go source files in the current directory, resolves struct types,
// matches fields between source and destination structs, and writes a generated file
// containing pure mapping functions — no reflection, no runtime dependencies.
//
// # Install
//
//	go install github.com/KARTIKrocks/gomapper@latest
//
// # Usage
//
// Add a go:generate directive to any Go file in your package:
//
//	//go:generate gomapper -src User -dst UserDTO
//
// Then run:
//
//	go generate ./...
//
// This produces a mapper_gen.go file with a MapUserToUserDTO function.
//
// # Multiple Pairs
//
// Map several struct pairs in a single invocation:
//
//	//go:generate gomapper -pairs User:UserDTO,Order:OrderDTO
//
// Use -bidirectional to generate both forward and reverse mappings:
//
//	//go:generate gomapper -pairs User:UserDTO -bidirectional
//
// # Flags
//
//   - -src: source type name
//   - -dst: destination type name
//   - -pairs: comma-separated Src:Dst pairs (alternative to -src/-dst)
//   - -output: output file name (default: mapper_gen.go)
//   - -mode: generation mode — "func" (default), "register", or "both"
//   - -bidirectional: generate both S→D and D→S mappings
//   - -tag: struct tag key for field renaming (default: "map")
//   - -strict: fail if any destination field is unmapped
//   - -ci: case-insensitive field name matching
//   - -nil-safe: generate nil checks for pointer dereferences
//   - -v: verbose output
//
// # Generation Modes
//
// func (default) — generates pure MapSrcToDst functions with no imports:
//
//	func MapUserToUserDTO(src User) UserDTO {
//	    return UserDTO{
//	        ID:   src.ID,
//	        Name: src.Name,
//	    }
//	}
//
// register — generates mapper.Register calls for the mapper library's runtime registry:
//
//	func init() {
//	    mapper.Register(func(src User) UserDTO {
//	        return UserDTO{
//	            ID:   src.ID,
//	            Name: src.Name,
//	        }
//	    })
//	}
//
// both — generates named functions and registers them via mapper.Register.
//
// # Field Matching
//
// Fields are matched in priority order:
//
//  1. Fields tagged map:"-" are skipped entirely
//  2. Destination map:"SourcePath" tag (supports dot notation for nested access)
//  3. Source map:"DstName" tag
//  4. Exact name match with assignable type → direct assignment
//  5. Exact name match with convertible type → type conversion
//  6. Case-insensitive name match (with -ci flag)
//  7. Pointer handling: *T→T (deref), T→*T (addr), *T→U (deref+convert)
//  8. Nested struct mapping: generates MapAToB call for different named struct types
//  9. Slice handling: []T→[]U generates inline loop calling MapTToU
//  10. No match → TODO comment (or error in -strict mode)
//
// Unexported fields are skipped. Embedded struct promoted fields are included.
//
// # Nil-Safe Mode
//
// Use -nil-safe to generate nil checks for pointer-to-value mappings:
//
//	// Without -nil-safe: *src.Name (panics on nil)
//	// With -nil-safe:
//	var _Name string
//	if src.Name != nil {
//	    _Name = *src.Name
//	}
//
// # Examples
//
// See the examples directory for complete working examples:
//   - examples/basic: Simple struct-to-DTO mapping with go:generate
//   - examples/advanced: Nested structs, slices, pointers, and tags
package main
