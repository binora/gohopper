# Conformance Tooling

Run route conformance comparisons between GraphHopper 11.0 and GoHopper:

```bash
gohopper conformance \
  --cases testdata/conformance/route_cases.json \
  --gh-url http://localhost:8989 \
  --go-url http://localhost:8080
```
