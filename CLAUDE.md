# migration guidline

* we're aiming to achieve parity with Graphhopper
* parity in structure, interfaces, tests, endpoints, cli, graph cache format

# code style

* Always write idiomatic Go.
* Use built-ins if possible.
* Do not hand-roll stuff.
* If there is no built-in, ask the user.


# testing guardrails
* make sure you write exactly the same tests as Graphhopper tests.
* use make commands


# important
* After an important milestone, spin up a subagent that refactors the code without touching the tests.
The goal: beautiful, idiomatic, and performant Go code needed to make that milestone successful.
* when in plan mode, do not write code as part of the plan
* Before starting a new milestone, run `/parity-check <package>` on the target package to understand the gap
* After completing a milestone, run `/parity-check <package>` again to verify coverage
* After completing a milestone, update beads (`bd close <id> -r "reason"`) based on the parity check results



