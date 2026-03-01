#!/usr/bin/env bash
set -euo pipefail

GO=${GO:-go}
GH_JAR=${GH_JAR:-/tmp/graphhopper-web-11.0.jar}
GH_JAR_URL=${GH_JAR_URL:-https://repo1.maven.org/maven2/com/graphhopper/graphhopper-web/11.0/graphhopper-web-11.0.jar}
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OSM_FIXTURE=${OSM_FIXTURE:-$REPO_ROOT/testdata/conformance/monaco.osm.gz}
OSM_FIXTURE_URL=${OSM_FIXTURE_URL:-https://raw.githubusercontent.com/graphhopper/graphhopper/11.x/core/files/monaco.osm.gz}

GH_CONFIG=${GH_CONFIG:-/tmp/gh-config.yml}
GO_CONFIG=${GO_CONFIG:-/tmp/go-config.yml}
GH_LOG=${GH_LOG:-/tmp/gh.log}
GO_LOG=${GO_LOG:-/tmp/go.log}

GH_PORT=${GH_PORT:-8090}
GO_PORT=${GO_PORT:-8080}
CONFORMANCE_CASES=${CONFORMANCE_CASES:-testdata/conformance/route_cases.json}

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required"
  exit 1
fi

[ -f "$GH_JAR" ] || curl -fsSL -o "$GH_JAR" "$GH_JAR_URL"
[ -f "$OSM_FIXTURE" ] || curl -fsSL -o "$OSM_FIXTURE" "$OSM_FIXTURE_URL"

cat > "$GH_CONFIG" <<YAML
graphhopper:
  datareader.file: $OSM_FIXTURE
  graph.location: /tmp/gh-cache
  import.osm.ignored_highways: footway,construction,cycleway,path,steps
  profiles:
    - name: car
      custom_model_files: [car.json]
  profiles_ch: []
  profiles_lm: []
  graph.encoded_values: car_access, car_average_speed, road_access
  routing.snap_preventions_default: tunnel, bridge, ferry
  graph.dataaccess.default_type: RAM_STORE
server:
  application_connectors:
    - type: http
      port: 8090
      bind_host: localhost
  admin_connectors:
    - type: http
      port: 8091
      bind_host: localhost
logging:
  appenders:
    - type: console
YAML

cat > "$GO_CONFIG" <<YAML
graphhopper:
  datareader.file: $OSM_FIXTURE
  graph.location: /tmp/go-cache
  profiles:
    - name: car
      custom_model_files: [car.json]
  profiles_ch: []
  profiles_lm: []
  graph.encoded_values: car_access, car_average_speed, road_access
  routing.snap_preventions_default: tunnel, bridge, ferry
  graph.dataaccess.default_type: RAM_STORE
server:
  application_connectors:
    - type: http
      port: 8080
      bind_host: localhost
logging:
  appenders:
    - type: console
YAML

java -Ddw.graphhopper.datareader.file="$OSM_FIXTURE" \
  -jar "$GH_JAR" server "$GH_CONFIG" > "$GH_LOG" 2>&1 &
gh_pid=$!

$GO run ./cmd/gohopper server "$GO_CONFIG" > "$GO_LOG" 2>&1 &
go_pid=$!

cleanup() {
  kill "$gh_pid" "$go_pid" >/dev/null 2>&1 || true
}
trap cleanup EXIT

for _ in $(seq 1 180); do
  if curl -fsS "http://127.0.0.1:$GH_PORT/health" >/dev/null 2>&1 && \
    curl -fsS "http://127.0.0.1:$GO_PORT/health" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

if ! curl -fsS "http://127.0.0.1:$GH_PORT/health" >/dev/null 2>&1; then
  echo "GraphHopper did not become healthy"
  tail -n 200 "$GH_LOG" || true
  exit 1
fi

if ! curl -fsS "http://127.0.0.1:$GO_PORT/health" >/dev/null 2>&1; then
  echo "GoHopper did not become healthy"
  tail -n 200 "$GO_LOG" || true
  exit 1
fi

failed=0
while IFS= read -r case_json; do
  name=$(echo "$case_json" | jq -r '.name')
  method=$(echo "$case_json" | jq -r '.method // "GET"')
  path=$(echo "$case_json" | jq -r '.path')
  body=$(echo "$case_json" | jq -c '.body // empty')

  gh_headers=$(mktemp)
  gh_body=$(mktemp)
  go_headers=$(mktemp)
  go_body=$(mktemp)

  if [ -n "$body" ]; then
    gh_status=$(curl --globoff -sS -o "$gh_body" -D "$gh_headers" -w '%{http_code}' \
      -X "$method" -H 'Content-Type: application/json' \
      --data "$body" "http://127.0.0.1:$GH_PORT$path")
    go_status=$(curl --globoff -sS -o "$go_body" -D "$go_headers" -w '%{http_code}' \
      -X "$method" -H 'Content-Type: application/json' \
      --data "$body" "http://127.0.0.1:$GO_PORT$path")
  else
    gh_status=$(curl --globoff -sS -o "$gh_body" -D "$gh_headers" -w '%{http_code}' \
      -X "$method" "http://127.0.0.1:$GH_PORT$path")
    go_status=$(curl --globoff -sS -o "$go_body" -D "$go_headers" -w '%{http_code}' \
      -X "$method" "http://127.0.0.1:$GO_PORT$path")
  fi

  gh_norm=$(mktemp)
  go_norm=$(mktemp)
  jq -S 'del(.took, .info.took)' "$gh_body" > "$gh_norm" 2>/dev/null || cat "$gh_body" > "$gh_norm"
  jq -S 'del(.took, .info.took)' "$go_body" > "$go_norm" 2>/dev/null || cat "$go_body" > "$go_norm"

  status_ok=1
  body_ok=1
  if [ "$gh_status" != "$go_status" ]; then
    status_ok=0
  fi
  if ! diff -u "$gh_norm" "$go_norm" >/tmp/diff.out 2>&1; then
    body_ok=0
  fi

  if [ "$status_ok" -eq 1 ] && [ "$body_ok" -eq 1 ]; then
    echo "OK   $name"
  else
    echo "FAIL $name"
    echo "  status gh=$gh_status go=$go_status"
    echo "  --- GH body ---"
    cat "$gh_norm"
    echo "  --- GO body ---"
    cat "$go_norm"
    echo "  --- Diff ---"
    cat /tmp/diff.out || true
    failed=$((failed + 1))
  fi

  rm -f "$gh_headers" "$gh_body" "$go_headers" "$go_body" "$gh_norm" "$go_norm"
done < <(jq -c '.[]' "$CONFORMANCE_CASES")

if [ "$failed" -ne 0 ]; then
  echo "Conformance failed: $failed case(s)"
  exit 1
fi
