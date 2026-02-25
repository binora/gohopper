# GoHopper 11.0 Drop-In Replacement Plan (Package-Level, Test-Gated)

## Summary
This plan is a strict, package-by-package parity roadmap with a hard-stop gate after every logical conclusion.  
No downstream package work advances until the current package’s full test matrix is green.

Locked decisions:
1. Milestones are package-level and granular.
2. Tests must mirror GraphHopper package tests (ported to Go).
3. Gate policy is hard-stop on failures.
4. Diff policy is strict, with only a tiny explicit nondeterminism allowlist.
5. Fixtures are mirrored from `../graphhopper` with provenance mapping.
6. CI full matrix is mandatory at each package milestone.
7. API milestones start after core route engine parity.

## Milestone Gate Template (applies to every milestone)
1. Implement package scope only.
2. Port matching GraphHopper tests for that package.
3. Add GoHopper package unit tests.
4. Add package integration tests (if package has runtime interactions).
5. Run strict parity checks (against expected behavior or GH outputs where applicable).
6. Run full matrix (`go test ./...` + package-specific conformance jobs).
7. Close milestone only if all tests pass and no open parity diffs remain.

## Milestones

## M0: Parity Harness Foundation
Scope:
1. Build test harness framework and reporting format.
2. Add provenance map from GH test class/file to Go test file.
3. Define strict diff rules and allowlist file for nondeterministic fields.

Tests:
1. Harness self-tests.
2. Diff engine tests with pass/fail fixtures.
3. CI job verifies harness can run empty baseline and produce report.

Exit gate:
1. Harness reproducible locally and in CI.
2. Provenance map checked in and validated.

## M1: `core/config` Parity
Scope:
1. Port GraphHopper config parsing behavior to Go package equivalents.
2. Align defaults and key handling for `graphhopper` config keys.
3. Preserve profile/ch/lm parsing semantics and error behavior.

Tests:
1. Port GH config parser tests.
2. Add YAML edge-case tests from GH fixtures.
3. Add negative tests for malformed config and missing required fields.

Exit gate:
1. Config behavior matches GH expected outcomes for mirrored fixtures.

## M2: `core/util` Parity
Scope:
1. Port geometry, distance, bbox, and polyline behaviors used by route flow.
2. Match numeric semantics and encoding/decoding behavior from GH util expectations.

Tests:
1. Port GH util/polyline test cases.
2. Add deterministic numeric comparison tests.
3. Add fuzz/property tests for polyline roundtrip.

Exit gate:
1. All mirrored util tests and property tests pass.

## M3: `core/storage` Base Graph Format Read Parity
Scope:
1. Implement GH11-compatible base graph binary readers.
2. Load GH-generated graph-cache artifacts into Go structures.
3. Validate metadata and bounds/parsing correctness.

Tests:
1. Port storage read tests from GH equivalents.
2. Add fixture-based binary compatibility tests using GH-produced cache files.
3. Add corruption/invalid-header tests.

Exit gate:
1. GoHopper can read GH11 cache files required for routing path.

## M4: `core/storage` Base Graph Format Write Parity
Scope:
1. Implement GH11-compatible binary writers.
2. Ensure written cache artifacts are GH-readable.

Tests:
1. Roundtrip tests: GH write -> Go read -> Go write -> GH read.
2. Byte-level structural assertions for key cache files.
3. Backward compatibility tests across repeated writes.

Exit gate:
1. Read/write compatibility gate passes both directions.

## M5: `core/storage/index` Location Index Parity
Scope:
1. Implement GH-compatible snap/index behavior.
2. Match nearest-point and snap validity semantics.

Tests:
1. Port GH location index tests.
2. Fixture tests for snapping edge cases and out-of-bounds behavior.
3. Performance sanity tests for index lookup regressions.

Exit gate:
1. Index behavior parity for mirrored fixtures and edge cases.

## M6: `core/reader/osm` Import Pipeline Skeleton Parity
Scope:
1. Port GH OSM import flow boundaries and invariants.
2. Build compatible import lifecycle hooks into storage/index layers.

Tests:
1. Port GH import validation tests.
2. Fixture tests on small OSM extracts.
3. Failure tests for missing/invalid inputs.

Exit gate:
1. Import pipeline produces route-queryable graph structures on baseline fixtures.

## M7: `core/routing` Request Validation Parity
Scope:
1. Match GH request validation logic exactly (headings, hints, curbsides, legacy params, bounds).
2. Match error type/message class and status mapping expectations.

Tests:
1. Port GH router validation tests.
2. Add strict error payload tests for API-layer integration.
3. Regression tests for invalid parameter combinations.

Exit gate:
1. Validation parity green with strict error equivalence.

## M8: `core/routing` Flexible Route Solver Parity
Scope:
1. Implement flexible mode route calculation parity.
2. Match path selection semantics for baseline profiles.

Tests:
1. Port GH flexible routing tests.
2. Route corpus tests on mirrored fixtures.
3. Deterministic path metrics tests (distance/time/path points).

Exit gate:
1. Flexible mode parity green for locked corpus.

## M9: `core/routing/ch` Preparation Parity
Scope:
1. Implement CH preprocessing parity.
2. Match profile preparation constraints and metadata behavior.

Tests:
1. Port GH CH prep tests.
2. Prep artifact compatibility tests.
3. Failure tests for unsupported profile/prep combinations.

Exit gate:
1. CH prep parity green with compatible artifacts.

## M10: `core/routing/ch` Query Parity
Scope:
1. Implement CH query solver behavior parity.
2. Match mode-selection and CH-specific constraints.

Tests:
1. Port GH CH routing tests.
2. Compare CH vs flexible consistency constraints from GH tests.
3. Performance gate for CH query latency regression.

Exit gate:
1. CH query parity green on corpus and package tests.

## M11: `core/routing/lm` Preparation Parity
Scope:
1. Implement LM preprocessing parity.
2. Match landmark selection/config semantics.

Tests:
1. Port GH LM prep tests.
2. Artifact and metadata compatibility tests.
3. Failure tests for invalid LM setup.

Exit gate:
1. LM prep parity green.

## M12: `core/routing/lm` Query Parity
Scope:
1. Implement LM query solver behavior parity.
2. Match flexible/CH/LM mode resolution semantics.

Tests:
1. Port GH LM routing tests.
2. Cross-mode consistency tests against GH expectations.
3. Performance guardrails for LM query behavior.

Exit gate:
1. LM query parity green.

## M13: `core/routing/weighting/custom` DSL Parity
Scope:
1. Implement full custom model DSL parser and evaluator parity.
2. Match weighting semantics for all supported operators/rules.

Tests:
1. Port GH custom model parser/weighting tests.
2. Add rule-combination matrix tests.
3. Negative tests for invalid models with exact error behavior.

Exit gate:
1. DSL and weighting parity green.

## M14: Route Engine End-to-End Package Gate
Scope:
1. Integrate config, import, storage, index, flexible/CH/LM, custom weighting.
2. Lock route-engine behavior under one end-to-end gate.

Tests:
1. Full route corpus against GH baseline outputs.
2. Stress tests for repeated import/query cycles.
3. Determinism tests across repeated runs.

Exit gate:
1. Route engine parity passes full matrix with strict diff policy.

## M15: `web-api` Type/Serialization Parity
Scope:
1. Align request/response/error model structures with GH route API expectations.
2. Ensure JSON/GPX serialization behavior matches contract.

Tests:
1. Port GH web-api serialization tests where applicable.
2. Contract tests for required/optional fields and defaults.
3. Error body schema strict tests.

Exit gate:
1. API type/serialization parity green.

## M16: `web-bundle/resources` `/route` GET/POST Parity
Scope:
1. Match `/route` request parsing and response formatting behavior.
2. Match headers and GPX response semantics.

Tests:
1. Port GH route resource tests.
2. Strict request/response conformance tests (GET and POST).
3. Error path conformance tests.

Exit gate:
1. `/route` strict parity green (allowlist only for approved nondeterministic fields).

## M17: `/nearest`, `/info`, `/health` Parity
Scope:
1. Match non-route core operational endpoints.
2. Align info payload and health behavior semantics.

Tests:
1. Port GH tests for these resources.
2. Endpoint contract tests.
3. Operational behavior tests with loaded/unloaded states.

Exit gate:
1. Endpoint parity green.

## M18: Isochrone/Matrix/Map-Matching/PT Package Sequence
Scope:
1. Implement remaining API surfaces in this order:
2. `isochrone`
3. `matrix`
4. `map-matching`
5. `pt`

Tests:
1. Port corresponding GH package tests per endpoint package.
2. Add endpoint-specific conformance corpora.
3. Add performance sanity tests where GH has equivalent benchmarks.

Exit gate:
1. Each endpoint package closes only when strict parity matrix is green.

## M19: Full Drop-In Conformance Gate
Scope:
1. Run complete cross-package parity suite.
2. Verify cache compatibility + route/API behavior in one integrated gate.

Tests:
1. End-to-end side-by-side runs against GH11 using mirrored fixture sets.
2. Full API contract and regression suite.
3. Upgrade/reload/restart stability tests.

Exit gate:
1. All package gates and full system gate green; no unresolved parity diffs.

## M20: Release Readiness Gate
Scope:
1. Final docs, contributor mapping, migration notes.
2. Freeze parity baselines and CI policy.

Tests:
1. Re-run full matrix on clean environment.
2. Run reproducibility checks from scratch (fresh import to full API tests).
3. Validate milestone evidence artifacts are complete.

Exit gate:
1. Release candidate tagged with full parity evidence.

## Important Public APIs / Interfaces / Types
1. CLI commands remain: `import`, `server`, `route`, `conformance`.
2. Config contract centers on GraphHopper-compatible `graphhopper:` keys and profile/ch/lm fields.
3. HTTP contract prioritizes exact `/route` parity first, then `/nearest`, `/info`, `/health`, then remaining APIs.
4. Error payload shape follows GH style: message + hints list.
5. Cache contract is GH11 read/write compatible for `graph.location` artifacts.

## Test Strategy (after every logical conclusion)
1. Package unit tests (ported GH tests + Go-native additions).
2. Package integration tests for package interactions.
3. Endpoint contract tests for web-facing packages.
4. Strict conformance tests where applicable.
5. Full matrix rerun at each milestone close.
6. Hard-stop policy: no next milestone until gate is fully green.

## Explicit Assumptions and Defaults
1. GraphHopper test suites are ported to Go package-by-package, not only used as informal references.
2. `../graphhopper` is the authoritative source for fixture and behavior provenance.
3. Strict diffs are default; allowlist is minimal and explicit.
4. Route engine parity is completed before broader API parity work.
5. CI full matrix is mandatory at each package milestone closure.
