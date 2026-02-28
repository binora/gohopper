# Plan: core/routing/ev Core Type Implementations

## Goal
Port `DecimalEncodedValueImpl` and `EnumEncodedValue` with their full test suites. These two types are the highest-priority missing pieces in `core/routing/ev` — they unlock all domain EV factories (MaxSpeed, RoadClass, Surface, etc.).

## Steps

### Step 1: Implement `DecimalEncodedValueImpl`
File: `core/routing/ev/decimal_encoded_value_impl.go`

- Struct embedding `*IntEncodedValueImpl`
- Fields: `Factor float64`, `UseMaximumAsInfinity bool`
- Constructor: `NewDecimalEncodedValueImpl(name, bits, factor, storeTwoDirections)`
- Full constructor: `NewDecimalEncodedValueImplFull(name, bits, factor, minStorableValue, negateReverseDirection, storeTwoDirections, useMaximumAsInfinity)`
- Methods implementing `DecimalEncodedValue`:
  - `SetDecimal` — converts float64 → int32 via factor, handles infinity
  - `GetDecimal` — retrieves int32, multiplies by factor, handles infinity
  - `GetNextStorableValue` — rounds up to next storable quantum
  - `GetSmallestNonZeroValue` — returns factor
  - `GetMaxStorableDecimal` — returns max (or +Inf if useMaximumAsInfinity)
  - `GetMinStorableDecimal` — returns min × factor
  - `GetMaxOrMaxStorableDecimal` — returns max of actual vs storable

### Step 2: Port `DecimalEncodedValueImpl` tests
File: `core/routing/ev/decimal_encoded_value_impl_test.go`

Port from `DecimalEncodedValueImplTest.java`:
- `TestGetDecimal` — basic set/get
- `TestSetMaxToInfinity` — infinity handling
- `TestNegative` — negative values and factor validation
- `TestInfinityWithMinValue` — infinity + min bounds
- `TestNegateReverse` — reverse direction negation
- `TestNextStorableValue` — round-trip quantum checks
- `TestSmallestNonZeroValue` — factor-based smallest value
- `TestNextStorableValueMaxInfinity` — next storable with infinity
- `TestLowestUpperBoundWithNegateReverseDirection` — max tracking
- `TestMinStorableBug` — edge case regression

Also port from `DecimalEncodedValueTest.java`:
- `TestDecimalInit` — basic init
- `TestDecimalNegativeBounds` — negative value rejection

### Step 3: Implement `EnumEncodedValue`
File: `core/routing/ev/enum_encoded_value.go`

- Generic struct `EnumEncodedValue[E ~int]` embedding `*IntEncodedValueImpl`
- Fields: `Values []E` (all enum constants)
- Constructor: `NewEnumEncodedValue[E](name, values)` — computes bits from len(values)
- Methods:
  - `SetEnum` — stores enum as its ordinal index
  - `GetEnum` — retrieves ordinal, returns values[ordinal]
  - `GetValues` — returns the values slice

### Step 4: Port `EnumEncodedValue` tests
File: `core/routing/ev/enum_encoded_value_test.go`

Port from `EnumEncodedValueTest.java`:
- `TestEnumInit` — init with a test enum, check default and set/get
- `TestEnumSize` — bit calculation for various enum sizes

### Step 5: Refactor
Spin up a subagent to refactor the new code without touching tests.

## Dependencies
- `IntEncodedValueImpl` (already implemented)
- `DecimalEncodedValue` interface (already defined)
- `ArrayEdgeIntAccess` (already implemented, used in tests)
- `InitializerConfig` (already implemented, used in tests)

## Exit Criteria
- `make test` passes
- All ported tests match Java test behavior
- Parity check shows DecimalEncodedValueImpl and EnumEncodedValue as MATCHED
