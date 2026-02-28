# Parity Check: GraphHopper Java → GoHopper Go

Audit a Go package against its corresponding GraphHopper Java package for structural and implementation parity.

## Input

The user provides a Go package path as the argument, e.g.:
- `core/storage`
- `core/util`
- `core/routing`
- `core/routing/ch`
- `web-api`
- `web-bundle/resources`

If no argument is given, ask the user which package to audit.

## Package Mapping

Map the Go package path to the Java source and test directories:

| Go package prefix | Java source base | Java test base |
|---|---|---|
| `core/` | `../graphhopper/core/src/main/java/com/graphhopper/` | `../graphhopper/core/src/test/java/com/graphhopper/` |
| `web-api/` | `../graphhopper/web-api/src/main/java/com/graphhopper/` | `../graphhopper/web-api/src/test/java/com/graphhopper/` |
| `web-bundle/` | `../graphhopper/web-bundle/src/main/java/com/graphhopper/` | `../graphhopper/web-bundle/src/test/java/com/graphhopper/` |

Subpackage mapping examples:
- `core/storage` → Java `storage/` under the base paths above
- `core/routing/ch` → Java `routing/ch/`
- `core/util` → Java `util/`
- `core/config` → Java `config/`
- `core/reader/osm` → Java `reader/osm/`

The Go project root is the current working directory. The Java reference is always at `../graphhopper` relative to the Go project root.

## Procedure

### Step 1: Discover Java side

Use Glob and Read to enumerate:
1. All `.java` source files in the mapped Java source directory (non-recursive first, then subdirectories)
2. All `.java` test files in the mapped Java test directory
3. For each Java file, extract: class/interface name, whether it's abstract, key public method signatures

### Step 2: Discover Go side

Use Glob and Read to enumerate:
1. All `.go` source files in the Go package directory (exclude `_test.go`)
2. All `_test.go` files
3. For each Go file, extract: type names (struct/interface), exported function/method signatures

### Step 3: Match and compare

For each Java source file:
1. Convert the class name to snake_case to find the expected Go filename (e.g., `BaseGraph.java` → `base_graph.go`)
2. Check if the Go file exists
3. If it exists, compare:
   - Does the Go file define the corresponding type?
   - Are key public methods present?
   - Is the implementation a stub (empty methods, TODO comments, panic("not implemented"))?
4. Classify as MATCHED, MISSING, or PARTIAL

For each Go source file:
1. Check if it corresponds to a Java file
2. If not, classify as EXTRA

For each Java test file:
1. Convert to expected Go test filename (e.g., `BaseGraphTest.java` → `base_graph_test.go`)
2. Check if the Go test file exists
3. If it exists, compare test function counts
4. Classify test gaps

### Step 4: Generate report

Output a structured parity report in this exact format:

```
# Parity Report: <package-name>

## Summary
- Java source files: N
- Go source files: N
- Matched: N | Missing: N | Partial: N | Extra: N
- Java test files: N
- Go test files: N
- Test coverage: N/N ported

## MATCHED (Go file exists with substantial implementation)
| Java File | Go File | Notes |
|---|---|---|
| ClassName.java | class_name.go | All key methods present |

## PARTIAL (Go file exists but incomplete)
| Java File | Go File | Missing |
|---|---|---|
| ClassName.java | class_name.go | Methods X, Y, Z not implemented |

## MISSING (No Go equivalent)
| Java File | Expected Go File | Priority |
|---|---|---|
| ClassName.java | class_name.go | High/Medium/Low |

## EXTRA (Go files with no Java counterpart)
| Go File | Notes |
|---|---|
| helper.go | Utility functions, may map to GHUtility.java |

## TEST GAPS
| Java Test | Expected Go Test | Status |
|---|---|---|
| ClassNameTest.java | class_name_test.go | Missing / Partial (N/M tests ported) |

## Recommended Next Steps
1. [Highest priority file to port]
2. [Second priority]
3. ...
```

## Priority Assignment

When marking MISSING files, assign priority based on:
- **High**: Core data structures, interfaces that other files depend on (e.g., DataAccess, Graph interface, RoutingAlgorithm)
- **Medium**: Implementations that are needed for the current milestone (check `plans/milestones.md`)
- **Low**: Utility classes, optional features, advanced algorithms not yet needed

## Important Notes

- Focus on the 11.x branch of GraphHopper (that's what `../graphhopper` contains)
- Java interfaces map to Go interfaces; Java abstract classes may map to Go interfaces + default implementation structs
- Java static utility methods typically become Go package-level functions
- Some Java files may legitimately not need a Go equivalent (e.g., Java-specific utilities like `Unzipper.java`)
- When comparing method signatures, account for Go's different error handling (returns error vs throws Exception)
- Flag any Go code that takes a fundamentally different approach from the Java implementation — parity means structural alignment
