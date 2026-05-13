# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

This repo is the upstream Juju source tree. The active development branch is
`4.0` (DQLite-based); `main` carries the 3.6.x line. Verify which branch you
are on before citing version-specific behavior.

## Always-loaded context

The files below are imported into every session. Treat them as authoritative:

@~/dev/juju-brain/JUJU.md
@./AGENTS.md
@./AGENTS.architecture-rules.md
@./AGENTS.core-domain-rules.md

Between them they cover: build/test commands, the strict layer hierarchy
(`cmd → api → apiserver → domain → core`, plus `internal/worker`), worker /
catacomb / dependency-engine patterns, watcher semantics, the Life state
machine, K8s/CAAS specifics, agent-binary lifecycle, the IAAS/CAAS dichotomy
invariants, and Juju 4.0 CLI differences from 3.x.

## Load on demand

These are NOT auto-imported. Read them when the task touches their domain:

- `AGENTS.documentation.rules.md` — Diataxis structure for `docs/` changes.
- `AGENTS.doc-dot-go-rules.md` — Rules for package-level `doc.go` files.
- `CODING.md` — Long-form rationale for worker/watcher/layering patterns
  ("EVERYTHING FAILS", `time.Now` discipline, facade attack-surface notes).
  Useful when the imports leave you uncertain *why* a rule exists.
- `STYLE.md` — Method/error documentation patterns (e.g. the
  `// The following errors may be returned:` convention).
- `CONTRIBUTING.md` — PR process, sign-off, commit-message expectations.

## Branch / workspace conventions

- Default working branch is `4.0`. Don't open PRs against `main` unless the
  change is genuinely 3.6-only.
- After any conflicted rebase, run `go build ./...` after each conflicted
  commit, not just at the end — semantic incompatibilities can survive a
  clean textual merge.
- After editing any `.go` file, run `gci` to fix import grouping before
  committing (see AGENTS.md for the exact invocation). CI's `gci` linter
  will fail otherwise.

## Spec Kit

<!-- SPECKIT START -->
Active feature: `specs/003-juju-advisor-cli/`.

Primary plan: [plan.md](specs/003-juju-advisor-cli/plan.md). Companion
artifacts in the same directory: `spec.md`, `research.md`,
`data-model.md`, `contracts/cli-contract.md`, `quickstart.md`. Read the
plan before implementing -- it defines six iterative milestones
(M0-M6) with explicit drop-out points for the time-boxed competition
window. The CLI contract in `contracts/cli-contract.md` is byte-locked
once M1 begins; do not iterate on visual format during implementation.
<!-- SPECKIT END -->
