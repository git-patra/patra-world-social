# Task 0 — startup recovery versus initial Persist

Implemented the narrow reproduced prerequisite on `feat/world-activities`. No commit made by this worker.

## Root cause and RED evidence

`RecoverPending` enumerated candidates before taking any database fence. The existing per-match lock protected a known row against a settling Enqueue, but an initial `PGStore.Persist` could have inserted a match and claim inside an uncommitted transaction. Startup could not enumerate that ID and could return success before its commit, leaving an unowned running match and busy claim after admission reopened.

The new integration test uses the existing isolated PostgreSQL schema helper and the real `PGStore.Persist`. An `AFTER INSERT` trigger on `active_match_users` waits on a test advisory lock, after both new match and claim have been inserted but before COMMIT. The test observes actual PostgreSQL backend blocking; there is no Store mock. It exercises successful persistence followed by an injected acknowledgement error, and a trigger exception causing actual transaction rollback.

Before the fix, both subtests failed:

```text
startup made an orphan decision before invisible initial Persist settled:
returned=true old_status=aborted_server claims=0 visible_new=0 recovery_err=<nil>
```

The first attempted test invocation used an empty project module cache and could not resolve downloads in the restricted network. Retried with the already populated default module cache. A sandboxed PostgreSQL run could not connect to loopback; the authorized integration runs used execution escalation and the existing loopback PostgreSQL instance. These were environment failures, not the behavioral RED above. No environment file contents or secrets were printed.

## Change and lock ordering

- `apps/server/internal/lobby/store.go`: acquire a shared transaction advisory fence, keyed by `current_schema() || '.matches.startup'` with hash seed 21, before the match lock or account locks. Normal independent starts share the fence.
- `apps/server/internal/history/finalizer.go`: acquire the same fence exclusively before candidate enumeration. Hold it through all recovery decisions and commit. Enumeration and recovery use one READ COMMITTED transaction, avoiding a second pooled connection while the startup fence is held. Each known match still acquires its match fence and rereads the durable job state in a later statement, preserving settling-Enqueue behavior.
- `apps/server/internal/history/startup_persist_integration_test.go`: new committed/rollback and lock-order/cancellation regressions. Before settling, startup cannot abort the older orphan or release its claim. After settling, every committed orphan is `aborted_server` with no claims, while a rolled-back initial start has no row. Repeated recovery succeeds.

Order is **startup fence → match advisory 7 → sorted account advisory 12**. Startup also acquires known match IDs in sorted order. Inspection of account deletion, identity locks, history finalization, leaderboard projection and backfill found no path that takes a startup fence after holding a match/account lock. Account deletion takes account 12 without taking startup/match locks; finalization/backfill retain their existing match → account order.

An explicit regression holds the initial start's match lock and starts Persist. Startup must wait even before Persist obtains that match lock. Deliberately reversing Persist's order to match → startup produced RED:

```text
startup passed Persist before it acquired its match lock
```

Restoring startup → match produced GREEN. The same test cancels a waiting startup transaction, then confirms Persist and subsequent recovery can finish, checking that cancellation releases the exclusive fence.

## Commands and results

All Go commands used the pinned Go 1.26.8 binary, `GOTOOLCHAIN=local`, and project `GOCACHE`; integration commands quietly sourced `.cache/database.env` with `set -a` and `set +a`. From repository root, the command setup was:

```sh
set -a
source .cache/database.env
set +a
export GOTOOLCHAIN=local
export GOCACHE="$PWD/.cache/go-build"
cd apps/server
```

The binary below is `../../.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`.

| Command arguments | Result |
| --- | --- |
| `test -count=1 -tags=integration ./internal/history -run '^TestPostgresStartupWaitsForInitialPersist$' -v` before implementation | Expected behavioral RED, both subtests |
| `test -count=1 -race -tags=integration ./internal/history -run 'TestPostgresStartup(WaitsForInitialPersist\|FencesSettlingEnqueueAndRecoversOnce)' -v` | GREEN, including existing settling-Enqueue test |
| `test -count=1 -race -tags=integration ./...` | Exit 0; all 20 test-bearing packages passed, four packages had no tests; application 65.267 s, history 39.800 s, lobby 11.639 s |
| `test -count=1 -race -tags=integration ./internal/history -run '^TestPostgresInitialPersistFencesBeforeMatchLock$' -v` with deliberately reversed lock order | Expected mutation RED |
| `test -count=1 -race -tags=integration ./internal/history -run 'TestPostgres(Startup(WaitsForInitialPersist\|FencesSettlingEnqueueAndRecoversOnce)\|InitialPersistFencesBeforeMatchLock)' -v` after restoring correct order | Exit 0, history 2.486 s; both initial-Persist outcomes, prior Enqueue recovery, lock ordering/cancellation passed |
| `vet ./...` | Exit 0 |
| `build ./...` | Exit 0 |
| `git diff --check` | Exit 0 |

The full integration suite completed before the final lock-order test was added. Production code was restored exactly to its full-suite-tested version after mutation; the final focused race run includes the final new tests. Other workers' concurrent presence/UI/audio changes were present in the shared checkout and are outside this worker's ownership.

## Limits

The acknowledgement error is injected after the real Persist transaction commits; this is not a network-level COMMIT acknowledgement fault. The rollback branch is an actual PostgreSQL trigger error and rollback. This change adds no migration and does not claim browser/LiveKit acceptance, performance profiling, T16 completion, or general multi-process gameplay support. The fence adds one shared advisory-lock statement per Persist and holds startup match locks until startup recovery commits; it is used only during startup recovery. Frontend validation and broader Task 0 evidence remain with the coordinator.
