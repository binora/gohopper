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



