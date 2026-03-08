SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.ONESHELL:

GO ?= go
GOCACHE ?= $(CURDIR)/.gocache

GH_JAR := /tmp/graphhopper-web-11.0.jar
GH_JAR_URL := https://repo1.maven.org/maven2/com/graphhopper/graphhopper-web/11.0/graphhopper-web-11.0.jar
OSM_FIXTURE := /tmp/monaco.osm.gz
OSM_FIXTURE_URL := https://raw.githubusercontent.com/graphhopper/graphhopper/11.x/core/files/monaco.osm.gz

GH_CONFIG := /tmp/gh-config.yml
GO_CONFIG := /tmp/go-config.yml
GH_LOG := /tmp/gh.log
GO_LOG := /tmp/go.log

GH_PORT := 8090
GO_PORT := 8080

CONFORMANCE_CASES ?= testdata/conformance/route_cases.json

.PHONY: test conformance parity ci

test:
	mkdir -p "$(GOCACHE)"
	GOCACHE="$(GOCACHE)" $(GO) test ./...

build:
	$(GO) build -o gohopper ./cmd/gohopper

conformance:
	GO="$(GO)" \
	GH_JAR="$(GH_JAR)" \
	GH_JAR_URL="$(GH_JAR_URL)" \
	OSM_FIXTURE="$(OSM_FIXTURE)" \
	OSM_FIXTURE_URL="$(OSM_FIXTURE_URL)" \
	GH_CONFIG="$(GH_CONFIG)" \
	GO_CONFIG="$(GO_CONFIG)" \
	GH_LOG="$(GH_LOG)" \
	GO_LOG="$(GO_LOG)" \
	GH_PORT="$(GH_PORT)" \
	GO_PORT="$(GO_PORT)" \
	CONFORMANCE_CASES="$(CONFORMANCE_CASES)" \
	./scripts/conformance.sh

parity:
	cd tests/parity && docker compose up --build --abort-on-container-exit --exit-code-from test-runner
	cd tests/parity && docker compose down

ci: test conformance
