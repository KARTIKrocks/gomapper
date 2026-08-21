# Style and pattern rationale

Context for the scoped rules in `config.json`. This file is freeform prose read
alongside the diff; `config.json` is what actually gates comment scope and
severity.

gomapper has no bug-fix history to draw worked examples from yet — the git log
is a single `0.0.1` initial release plus dependabot bumps, and `CHANGELOG.md`
only lists features added, not bugs fixed. The examples below are instead
mechanisms verified directly against the current code during this review setup
(March 2026 codebase): each one is a real gap in the code as it stands today,
not a historical incident. Treat them as "here's exactly how this would break"
rather than "here's what broke before."

## Silent unmapped fields — the core failure mode

`matcher.matchField` (`internal/matcher/matcher.go`) tries, in order: the
destination's own `map:"..."` tag, a source field whose tag targets this
destination name, an exact name match, then (with `-ci`) a case-insensitive
name match. If none hit, `matchDstFields` records an `UnmappedField` and the
templates render `// TODO: unmapped field Name (Type)` — the destination
struct literal simply has no line for that field, so it silently gets its zero
value. This only becomes a hard error with `-strict`.

That means a typo in a `map:"..."` tag, or a field rename on either side that
isn't mirrored on the other, doesn't produce a compile error or a wrong-value
bug you'd catch in a diff — it produces a field that's just missing from the
generated code, indistinguishable at a glance from an intentionally-skipped
field. This is the single most consequential correctness property of the
whole tool, which is why `silent-unmapped-field-mismatch` is the first rule.

## Numeric conversions have no narrowing check

`makeMapping` falls through to `types.ConvertibleTo(src.Type, dst.Type)` for
any pair that isn't directly assignable, and the templates emit that as
`ConvType(src.Field)` — a bare Go type conversion. `types.ConvertibleTo`
returns true for `int64→int32`, `uint64→uint8`, `float64→float32`, and
`int→uint` just as readily as for safe widening conversions like `int→int64`
(the one actually used in `examples/basic/types.go`, `Age int` → `Age int64`).
Go's own explicit conversion silently truncates or wraps on overflow for the
narrowing direction — there's no generated bounds check, and no signal in the
output that a given field is narrowing rather than widening. A caller reading
`mapper_gen.go` sees the same one-line conversion either way.

## Two maps whose iteration order leaks into generated output

`processNestedDstTags` iterates `idx.byTag` — built as a `map[string]loader.StructField`
in `buildSourceIndices` — to build `NestedDstAssignments`, which the templates
then range over in the order they arrive. `matchField`'s `-ci` fallback
similarly ranges over `srcByName` to find a case-insensitive candidate. Go
map iteration order is randomized per process. Right now this only bites when
there's more than one dot-notation `map:"Parent.Child"` tag in a single
struct pair, or more than one source field that case-insensitively matches a
destination field — both narrow cases today — but the two loops sit exactly
on the boundary of "generated file must be byte-identical across reruns,"
which is the property `go generate` idempotency and any golden-file/CI diff
check depends on. Widening either matching feature without sorting the map
keys first would turn a narrow edge case into a common one.

## Three template modes, one decision tree, three copies of it

`internal/generator/templates.go` defines `registryFieldExpr` (register mode),
`pureFieldExpr` (func mode), and `sliceLoopExpr` (slice element rendering,
shared by both) as three separate template strings, each re-implementing the
same branch order over `Deref` / `AddrOf` / `IsStructMap` / `IsSliceMap` /
`NeedsConv`. `-mode both` runs both `pureFieldExpr` and the register path in
the same file. There is no single source of truth for "what does this
FieldMapping shape render as" — it's whatever each template's `if/else if`
chain says, independently. A `matcher.go` change that introduces a new flag
combination (or changes what an existing combination means) has to be carried
into all three by hand; the integration tests build golden output for `func`
mode's `both`/`register` variants but a mismatch that still *compiles* (e.g.
register mode silently falling through to a subtly different but valid
expression) would not necessarily fail `go build` the way a missing case
would.

## `-nil-safe` only protects the paths that were wired up

The flag's promise is unconditional: no generated dereference panics on nil.
In practice that promise is implemented as two separate opt-in blocks —
`nilSafeBlock` for field-level `Deref`, and the `if _v != nil` branches inside
`sliceLoopExpr` for `SliceElemDeref`. Both are template-level, keyed off flags
set in `matcher.go`. If a future change adds a new way for `Deref` (or a
slice-element equivalent) to become true — say, a new pointer-unwrapping rule
in `tryDerefMapping` — and the corresponding nil-guard branch isn't added to
`nilSafeBlock`/`sliceLoopExpr`, the generated code still compiles, `-nil-safe`
still runs without error, and the one new field just isn't nil-checked. That's
strictly worse than not having the flag, because the caller has explicitly
asked for the safety property and gomapper reported success.

## A missing nested pair fails downstream, not in gomapper

`tryStructMapping` and its slice/pointer variants decide `IsStructMap` purely
from Go type shape (`areDifferentNamedStructs`) — they never check that the
nested `Src:Dst` pair is actually among the pairs `main.go` is generating
functions for. `examples/advanced` deliberately exercises this correctly
(`-pairs Address:AddressDTO,Order:OrderDTO` includes both), but nothing stops
someone from running `-pairs Order:OrderDTO` alone: gomapper would still
"succeed," write a file calling `MapAddressToAddressDTO`, and the first error
the user sees is a `go build` failure for an undefined function, with no
indication that the fix is "add Address:AddressDTO to -pairs."

## Golden files are the regression net for everything above

`integration_test.go` byte-compares fresh output against
`testdata/{basic,advanced,embedded}/expected_gen.go.golden` (stripping the
`//go:build ignore` line). Every mechanism above — unmapped-field rendering,
conversion expressions, nil-safe blocks, slice loops — is exercised through
these fixtures, not through unit assertions on template strings. A PR that
changes generated output without touching the matching `.golden` file is
either untested or (more likely) will fail CI; a PR that updates the
`.golden` file without a corresponding source change, or that "fixes" a
golden file to match a bug rather than fixing the bug, defeats the point of
the fixture.
