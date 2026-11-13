# JUJU-9604: Kubernetes SSH review sequence

This branch contains six implementation commits, followed by this review-only
note. Review commits **1 through 6 individually** using the links below. Each
link shows only that commit's change relative to its parent. The complete
branch diff is cumulative and is useful only as a final cross-check.

The six commits are the proposed future PR boundaries. Each includes the
relevant regression tests and builds on the preceding commits. No PR has been
opened; GitHub CI for future PR heads still needs to run.

Base: `42f19c2ea9e8b2ef99e4c26f5af142f9e75c47e1` in `juju/juju`.
Implementation tip: `65e9dffc1e9bdde89470b8f218f54dbfdef92645`.

| Step | Incremental diff | Scope | Review focus |
| --- | --- | --- | --- |
| 1 | [51cab23261](https://github.com/sinanawad/juju/commit/51cab232616bc214a77b38f91fe83cb938fe1fba) | Test setup and teardown | Shared suites, deterministic waits and local test conventions. No production behaviour changes. |
| 2 | [44c0c028ae](https://github.com/sinanawad/juju/commit/44c0c028ae2b15e33f6d09a547d28694fc45b19c) | Primary stream and terminal teardown | Cancel and join the primary exec stream; release terminal-size readers and writers on stop. |
| 3 | [411ecf1f9f](https://github.com/sinanawad/juju/commit/411ecf1f9fa532e45c09b5bdd4129ba9550cce8a) | Shell fallback and replay | Classify missing executables, retry sh/bash, preserve stdin/resize events and forward Bash signals. |
| 4 | [bdfd0e8663](https://github.com/sinanawad/juju/commit/bdfd0e86638d3e004d8eefc0c3ab0f0b3a6c24bd) | Direct and raw session intent | Adopt executor shell intent in direct CLI and raw SSH; preserve explicit commands and default shell preferences. |
| 5 | [5db431736a](https://github.com/sinanawad/juju/commit/5db431736ab5581cfb203c067512b5e8d9f3577b) | Negotiated CLI login shells | Ship SSHClient v7, the final-hop convention, client emission and server handling together; preserve older controllers. |
| 6 | [65e9dffc1e](https://github.com/sinanawad/juju/commit/65e9dffc1e9bdde89470b8f218f54dbfdef92645) | Integration coverage and documentation | Real PTYs and four workload types; opt-in Bash-only fixture, safe client-key reuse and compatibility docs. |

The seventh commit only adds this guide. Exclude it from the implementation
PRs. Local agent working notes and temporary validation artifacts are not part
of this branch.

## Intended behaviour

Kubernetes workload images containing only Bash should support `juju ssh`.
Images with neither `sh` nor `bash` should return a clear error and non-zero
status. Ordinary shell selection, login profiles, explicit commands, stdin,
terminal sizes and signal delivery must remain intact.

Explicit commands, raw SSH shell requests and direct sessions without a PTY
prefer `sh`. Commandless CLI sessions through a capable controller, and direct
sessions with a PTY, prefer a Bash login shell. Each tries the other shell only
when the preferred executable is absent. Command failures, permission errors,
invalid working directories and signal-delivery failures must not cause a retry.

Step 2 joins the primary stream; it does not claim to join client-go's internal
stdin/resize readers. Step 3 isolates those readers across retries. At step 3,
explicit-command fallback is usable, but commandless direct/raw callers still
pass their legacy explicit shells. Steps 4 and 5 complete the user-facing paths.

For step 3, read `errors.go` and `exec.go` first, then the stdin and resize
adapters with their corresponding tests. Most of this step is regression
coverage for retry boundaries and concurrent readers.

Keep step 5 atomic. The facade must not advertise the convention before the
SSH handler understands it. The final-hop username selects a shell; the outer
hop still authenticates and authorises the actual Juju user.

## Review locally after fetching

From an existing Juju checkout:

```sh
git fetch https://github.com/sinanawad/juju.git review/JUJU-9604-kubernetes-ssh
git worktree add --detach ../juju-9604-review FETCH_HEAD
cd ../juju-9604-review
```

The fetched history is available offline. Review one incremental change with:

```sh
git show 51cab232616bc214a77b38f91fe83cb938fe1fba
```

Use any SHA from the table to inspect another step. To run tests at that step
in this separate review worktree:

```sh
git switch --detach 51cab232616bc214a77b38f91fe83cb938fe1fba
go test -race ./internal/provider/kubernetes/exec ./cmd/juju/ssh ./internal/worker/sshserver/handlers/k8s
```

For a cumulative check of the implementation only:

```sh
git diff 42f19c2ea9e8b2ef99e4c26f5af142f9e75c47e1 65e9dffc1e9bdde89470b8f218f54dbfdef92645
```

Record findings against a step number and file/function. When these become PRs,
base each later PR on its predecessor until that dependency merges; comparing
every branch directly with main would include all preceding changes again.

## Validation

The final implementation's 39 changed/new code, test and product-documentation
files were compared byte-for-byte and by executable mode against the completed
working copy. Splitting introduces no final behaviour or source changes.

Checks run while assembling this sequence:

| Step | Result |
| --- | --- |
| 1 | Executor, SSH CLI and Kubernetes SSH handler package race tests passed. |
| 2 | Executor race tests passed; 112 bounded race stress runs, zero failures. |
| 3 | Executor, CLI and handler race tests passed; 108 executor stress runs, zero failures; scoped lint passed. |
| 4 | CLI and handler race tests passed; 80 CLI and 57 handler stress runs, zero failures. |
| 5 | API, SSH facade, core SSH, CLI and handler race tests passed. |
| 6 | Bash syntax, ShellCheck, shfmt, Python compilation, diff checks and exact final-file comparison passed. |

All touched Go files were formatted with gci. Final scoped golangci-lint across
all changed Go packages reported zero issues. `make pre-check` passed; its
optional full-repository static analysis was disabled, with the scoped lint
run separately. Each stress run was bounded by a 31-second outer timeout;
reported counts are completed runs from the final progress report.

The completed implementation was also exercised on a local Kubernetes cluster:

- Juju 4.2 client to 3.6.29 controller: the direct SSH matrix passed.
- Juju 4.2 client to 4.1-beta3 controller: legacy login/profile behaviour,
  terminal handling, commands and raw SSH defaults passed.
- Official 4.1-beta3 client to the rebuilt 4.2 controller: legacy login,
  profiles, PTY/resize, explicit commands and exit status passed.
- Rebuilt 4.2 client and controller: the full shell matrix passed. After the
  test-key and opt-in fixes, the latest rerun completed in 157 seconds.
- Controller api-server login loaded the real `juju_engine_report` function.
- The latest test archive contained no shell-key/client-key files or private-key
  PEM markers. Simulated raw SSH failure also produced no new key artifact.

A 3.6 client against a 4.2 controller was **not live-tested**; do not treat the
reverse-direction test above as evidence for that combination. No claim of
blanket cross-major compatibility is made.

The live tests used locally built controller and Bash-only fixture images.
See `tests/README.md` for the opt-in shell matrix command. A reachable image
with Bash and no sh is required; the test verifies that condition before it
exercises fallback. The full Juju build/native dependency environment is still
required when running the relevant package checks.
