# Agent Instructions

## Migration Guidelines

- Aim for parity with GraphHopper.
- Preserve parity in structure, interfaces, tests, endpoints, CLI behavior, and graph cache format.

## Go Style

- Always write idiomatic Go.
- Use built-ins if possible.
- Do not hand-roll functionality that the standard library or an existing project dependency already provides.
- If there is no built-in or existing dependency for an important piece of functionality, ask the user before adding a custom implementation.

## Testing Guardrails

- Write tests that match the corresponding GraphHopper tests as closely as practical.
- Use Makefile commands for verification:
  - `make build`
  - `make lint`
  - `make test`

## Important

- After an important milestone, spin up a subagent that refactors the code without touching the tests.
  The goal is beautiful, idiomatic, and performant Go code needed to make that milestone successful.
- When in plan mode, do not write code as part of the plan.
- Before starting a new milestone, run `/parity-check <package>` on the target package to understand the gap.
- After completing a milestone, run `/parity-check <package>` again to verify coverage.
- After completing a milestone, update beads with `bd close <id> -r "reason"` based on the parity check results.
