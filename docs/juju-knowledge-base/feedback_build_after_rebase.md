---
name: build-verify-after-rebase-conflicts
description: Always run go build after resolving rebase/merge conflicts to catch semantic incompatibilities that survive textual merge
type: feedback
---

Always run `go build` (or equivalent compilation check) after resolving each conflict-heavy commit during a rebase, not just at the end.

**Why:** During the 001-k8s-deployment-types rebase onto upstream/main, two build errors were introduced by conflict resolution — `logger` used in a function whose signature no longer included it, and a missing `appUUID` argument in a call that auto-merged cleanly but had the wrong arity. Both were semantic incompatibilities that survived textual merge (no conflict markers, but types/signatures disagreed). A `go build` after each conflicted commit would have caught them immediately.

**How to apply:** After `git rebase --continue` succeeds for any commit that had conflicts, run `go build ./affected/packages/...` before moving to the next commit. If multiple packages were touched, build them all. Don't wait until the full rebase is done.
