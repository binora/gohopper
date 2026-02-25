# GoHopper

GoHopper is a GraphHopper 11.0 rewrite in Go with a deliberately familiar structure for contributors coming from the Java codebase.

## Repository layout

- `core/` mirrors GraphHopper core logic and type boundaries:
  - `core/graphhopper_config.go` (GraphHopperConfig equivalent)
  - `core/graphhopper.go` (GraphHopper equivalent)
  - `core/routing/router.go` (Router equivalent)
  - `core/storage/` and `core/storage/index/` (BaseGraph and LocationIndex equivalents)
  - `core/reader/osm/` (OSM reader entry points)
- `web-api/` request/response types (`GHRequest`, `GHResponse`, `ResponsePath`)
- `web-bundle/resources/` HTTP resources (`/route`, `/nearest`, `/info`, `/health`)
- `web-bundle/server.go` server wiring
- `tools/` conformance tooling and fixtures

## Commands

- `gohopper import config-example.yml`
- `gohopper server config-example.yml`
- `gohopper route --config config-example.yml --point 52.53,13.35 --point 52.50,13.43`
- `gohopper conformance --cases testdata/conformance/route_cases.json --gh-url http://localhost:8989 --go-url http://localhost:8989`

## Status

This commit establishes GraphHopper-like structure and routing request/response flow.
Binary cache compatibility and full algorithm parity are scaffolded but not complete yet.
