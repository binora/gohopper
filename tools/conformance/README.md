# Conformance Tooling

Run route conformance comparisons between GraphHopper 11.0 and GoHopper:

```bash
gohopper conformance \
  --cases testdata/conformance/route_cases.json \
  --allowlist testdata/conformance/allowlist.json \
  --gh-url http://localhost:8989 \
  --go-url http://localhost:8080 \
  --report-out /tmp/conformance-report.json
```

The allowlist is explicit and minimal (`json_paths` and `header_keys`) and should
only include known nondeterministic fields.
