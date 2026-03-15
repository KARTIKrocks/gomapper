# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2026-03-15

### Added

- Code generation tool for type-safe struct mapping functions
- Field matching by name, struct tags (`map:"..."`), and type compatibility
- Three generation modes: `func` (pure functions), `register` (mapper library), `both`
- Bidirectional mapping generation with `-bidirectional` flag
- Multiple pair support via `-pairs Src:Dst,Src2:Dst2` syntax
- Nested struct mapping — generates `MapAToB` calls for different named struct types
- Slice field mapping — generates inline loops with per-element mapping
- Pointer handling: `*T`→`T` dereference, `T`→`*T` address-of, `*T`→`U` deref+convert
- Nil-safe pointer dereference mode (`-nil-safe`) with generated nil checks
- Case-insensitive field name matching (`-ci`) as a fallback after exact match
- Struct tag renaming via configurable tag key (`-tag`, default: `map`)
- `map:"-"` tag to skip fields on either source or destination
- Dot-notation tag paths for nested struct field access (`map:"Address.Street"`)
- Strict mode (`-strict`) — fails if any destination field is unmapped
- Embedded struct field flattening (promoted fields are included)
- Type conversion for convertible types (e.g., `int` → `int64`)
- Verbose mode (`-v`) with field matching decision output
- Configurable output file name (`-output`, default: `mapper_gen.go`)
- Integration test suite with golden file comparison
- Unit tests for loader, matcher, and generator packages
- Zero dependencies in generated code (func mode)

[0.0.1]: https://github.com/KARTIKrocks/gomapper/releases/tag/v0.0.1
