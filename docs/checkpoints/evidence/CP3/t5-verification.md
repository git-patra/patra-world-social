# T5 implementation evidence (in progress)

Resumed inherited untracked hub/runtime/movement and tests, preserving them. No applicable repository AGENTS.md was found; supplied Postman instruction retained. No browser credential actions performed. No staging/commits or PROGRESS ledger edits.

## Observed verification

- Initial `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -race ./internal/realtime ./internal/world`: both PASS (cached). This is observed initial state, not evidence of an uncached run.
- Added direct websocket admission/movement/revoke test; `go test -count=1 ./internal/realtime`: RED, `undefined: NewHandler`.
- Implemented transport; initial sandbox run could not bind httptest loopback (`operation not permitted`). Retried direct backend listeners with approved escalation.
- `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race ./internal/realtime ./internal/world`: PASS realtime 1.559s, world 1.291s. Covers concrete websocket handshake, evil origin and absent cookie rejection, authoritative movement snapshot, revoke policy-close; inherited concurrent 20-capacity/resume/isolation/fencing tests and fake-clock movement/collision tests.
- `go test -count=1 ./internal/world -run TestMapIntegrity`: RED `tampered map accepted`. Integrity implementation pending.

## Implemented so far

Single owner world runtime (20Hz), bounded request queue, server direction/path movement, collision, sequence/epoch fencing; hub admission/capacity/grace/takeover; direct websocket handler with exact origin, cookie/session/membership checks, 16KiB reader bound, rate limits, bounded command queue, 10Hz full state deltas, deadline writes/ping, ongoing authorization, immediate revoke.

## Remaining

Concrete shared services and lifecycle wiring; checksum validation/shared assets; frontend lifecycle/interpolation/reconciliation/status/explicit takeover; shared contracts; isolated real DB integration and 20-client load smoke; fuller security/backpressure/expiry tests; final full suite/build/typecheck.

Browser two-user/credential/offline/tab/reduced-motion acceptance: NOT_RUN per explicit user deferral. Synthetic direct backend integration remains authorized.

## Additional milestones

- Map integrity RED `tampered map accepted` → canonical SHA-256 validation in Go loaders and embedded shared map/rules copies with drift test → `go test -count=1 ./internal/world` PASS 0.468s.
- Concrete application wiring test RED `undefined: New`; implemented shared Application owning Auth, Worlds, Hub, routes and callback wiring, main graceful shutdown. First integration execution had missing DATABASE_URL; sourced ignored env without printing, then dedicated DB was stopped (`connection refused`). Coordinator restarted existing dedicated PG, unrelated to code.
- Frontend RED missing reconcile module → bounded SnapshotBuffer + validation/clock; frontend tests PASS 8 JS +12 TS.
- Frontend socket RED missing module → memory-only resume seat, epoch reset, explicit-only takeover, bounded send, reconnect/policy-close state; frontend PASS 8 JS +13 TS.
- Controller RED `world.setOnline is not a function` → separate saved solo state, authoritative self reconciliation, remote interpolation, no duplicate self, online intent dispatch, modal still receives snapshots, Reset cannot teleport → `npm test --workspace apps/web` PASS 8 JS +13 TS; `npm run typecheck --workspace apps/web` PASS.
- Added visible online UI + world picker join action, modal/visibility input cancellation, 100ms held-key input, websocket Vite proxy. Final verification pending.

- Real application integration initially failed `development requires loopback HTTP`; corrected synthetic test Origin to `http://127.0.0.1:4176` to respect the existing guard (no production relaxation). Then `go test -count=1 -race -tags=integration -v ./internal/application` PASS TestLoadSmoke 15.38s/package17.039s: 20 seats, 21st denied, isolated second world, actual 15-second disconnect/resume, epoch/capacity, real logout callback.
- Expanded `make load-smoke` initially FAIL `ban did not close ... EOF`: synthetic idle clients were not reading, so server's ping timeout had already disconnected them. Revised smoke clients continuously read and send 10Hz movement, require live snapshots and seq>=50 per continuing client; rerun pending. This is a fixture correction, not removal of heartbeat.
- Shared snapshot schema/fixtures added. Go RED `undefined: ValidateSnapshot`; TS RED `Missing expected exception` for unknown field. Implemented strict Go/TS snapshot decoders. `go test -count=1 ./internal/realtime -run TestSharedSnapshotContract` PASS 0.598s; frontend PASS 8 JS +14 TS tests.
- Malformed transport frame test RED `malformed frame needs policy close ... EOF` → explicit policy-close reason communicated from reader to writer. Uncached `go test -count=1 -race ./internal/realtime ./internal/world` PASS realtime2.852s/world1.552s, including new fake-clock 15/30s grace/expired-session and malformed/cross-world input tests.

- Sustained `PATH=/opt/homebrew/bin:$PATH make load-smoke`: PASS TestLoadSmoke16.49s/package18.546s. 19 clients remain actively reading/ponging and sending 10Hz movement during the first client's actual15s outage; each continuing client's seq>=50 and 20 avatars; 21st rejection, world isolation, resume epoch/capacity, real logout+ban callback and periodic session-expiry enforcement pass. Extended membership-reread/archive assertions pending final integration rerun.
- Full `GOCACHE=... /opt/homebrew/bin/go test -count=1 -race ./...`: PASS all packages, realtime12.320s (includes ~8s heartbeat timeout test), world2.857s. Transport takeover/flood and unread-heartbeat tests pass. No integration-tagged tests included in this command.
- Root `npm test`, `npm run typecheck`, `npm run lint`, `npm run build`: PASS; 8 JS+14 bundled TS tests; Vite147 modules, JS315.45kB/gzip102.50kB, build221ms. TypeScript compiler printer expanded touched worlds UI control flow; removed unused selected state.

## Final implementation and verification

Ready for coordinator independent review; no commits/staging performed. Browser-specific acceptance remains NOT_RUN, so this is implementation/backend/controller evidence rather than a claim of completed browser acceptance.

### Interfaces and integration

- `realtime.NewHub(SessionChecker, MembershipChecker, MapDefinition, Clock)` owns admission across worlds; `Join(ctx, JoinContext)`, `Submit(ctx, JoinContext, Command)`, `Resume(ctx, JoinContext, lastEventID)`, `Disconnect(sessionID, reason)`, `Revoke(worldID,userID)`, `Validate`, `Snapshot`, `Done`, `Close`.
- Resume deliberately accepts the full authenticated JoinContext, including cookie token/world/opaque seat/epoch, instead of looking up a seat using a bare caller-supplied opaque ID. This preserves caller/session binding and revalidates real membership. Every resume is a full resync; durable event replay belongs later game/chat tasks.
- `application.New(pool, httpapi.Options)` constructs concrete shared Auth/Worlds/Hub, registers `/api/v1/realtime`, assigns revocation callbacks and validates embedded map/rules; main uses this application and SIGINT/SIGTERM shutdown. Existing auth/world REST routing retained.
- Logout callback is account-scoped and closes the online seat even if a different cookie owns it, while DB logout revokes only the requested cookie. Recovery DB-revokes all account sessions and closes its seat. Membership callbacks are post-commit; ongoing command and one-second periodic checks catch changes without callbacks.
- 20Hz server motion, 10Hz snapshots, 4.5 tiles/sec, 250ms direction expiry; server pathfinding/collision, integer destination validation, normalized direction, stale sequence and old epoch fencing. One active world seat per account, explicit takeover, opaque ID rotation, 30s disconnected reservation, capacity20 including reconnect.
- Exact Origin and cookie/member admission; bounded32 inbound commands, bounded128 runtime requests, no unbounded outbound queue, coalescing snapshot ticker, 2s writes, 5s ping/3s timeout, input60/control10 per second; malformed/flood policy close. Transport16KiB and stricter existing command8KiB limits documented.
- Frontend WorldSocket keeps resume credentials in memory, sends explicit takeover only from the visible button, detects incompatible versions/checksums and policy closure, reconnects for up to30s; Online UI exposes truthful live/reconnect/full/revoked/solo states. Saved solo state restored on leave; online self is reconciled directly and rendered once; remote interpolation has two snapshots/100ms delay/no extrapolation. Modal/background input cancels without stopping incoming authoritative state; animation preference does not change authority.
- Added strict public snapshot schema plus shared Go/TS valid/invalid fixtures, including unknown-field rejection and Unicode names. Bundled assets copied into Go embed directory are checksum-verified and byte-drift tested against root contracts. No remote map/rules asset downloading was added; remote snapshots must match bundled IDs/checksums/protocol.
- `make load-smoke` sources ignored DB env only when DATABASE_URL unset, creates/drops a random schema, uses transient HTTP server and synthetic websocket clients. No source-repo edits, browser credentials, previews stopped, remotes/push/deploy/purchases.

### Final actual commands and results

Run from `apps/server` after `set -a; source ../../.cache/database.env; set +a` without printing env:

```
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go test -count=1 -race -tags=integration ./...
```

PASS: cmd/server1.549s; application18.310s; auth8.491s; config2.352s; games/model2.897s; http4.027s; realtime12.529s; world4.874s; worlds4.640s. Application load now includes live reading/pong/movement, actual15s outage, count/epoch/isolation, logout/ban/archive callbacks, session-expiry and membership rereads without callbacks. All test schemas are cleaned up.

```
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go vet ./...
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go build -o ../../.cache/server-t5 ./cmd/server
/opt/homebrew/bin/go mod verify
```

PASS (vet/build silent; `all modules verified`). Coder WebSocket v1.8.15 was inherited in the installed/cache dependency state, retained/pinned and made a direct dependency by go mod tidy; no speculative version upgrade.

After the broad suites, shared Unicode24-emoji fixture exposed TS UTF16 counting RED (`Invalid avatar`) while Go accepted codepoints. Changed TS name counting to codepoints. Narrow Go `go test -count=1 ./internal/realtime -run TestSharedSnapshotContract` PASS0.752s.

Final UI seam review found global pointerup zero-input cancelled canvas destinations immediately. Added PointerHold regression RED (missing export), then implemented owned directional hold release and used it in app; canvas releases no longer send zero unless a directional hold was active. Non-movement key releases no longer cancel destinations; blur/visibility still cancel deliberately. No Go production changes after broad suites.

Final frontend commands (repository root): `npm test --workspace apps/web`, `npm run typecheck --workspace apps/web`, `npm run lint --workspace apps/web`, `npm run build --workspace apps/web`, `git diff --check`. Results recorded by final tool output; all PASS: 8 JS and15 bundled TS tests, typecheck/lint clean, Vite147 modules. Browser acceptance NOT_RUN.

### Remaining limits

- Two credentialed browsers seeing movement, browser offline/background/modal/reduced-motion UX, keyboard/mobile feel and visual quality are NOT_RUN by explicit user deferral; controller/protocol tests are separate evidence.
- Single-instance memory runtime; server restart loses presence and requires fresh join. No game/chat event persistence is invented. `world.delta` intentionally carries replaceable complete state.
- This is a basic local20-client smoke, not a production latency/CPU/mobile/render benchmark.
- Coordinator owns `docs/PROGRESS.md`, `docs/ACCEPTANCE.md`, independent review and commit. These files were not edited by this implementer.

## Fix round 1 — four Important review findings

Read task-5-review.md; retained original staged index for coordinator review (no staging/commits). Minor Reset copy remains coordinator-deferred to T15. Browser acceptance remains NOT_RUN.

1. Scoped revocation: gated regression first failed all join/validate/resume for unrelated A revocation (`access revoked` / `connection fenced`). Added per-account/world/membership last-revocation generations; only relevant generations invalidate an authorization read. Same-subject admission race remains protected. `go test -count=1 -race ./internal/realtime -run 'TestUnrelatedRevocation|TestRevocationDuringAdmission'` PASS1.556s.
2. Admission cancellation: introduced a room interface to gate actual runtime calls in tests; RED reproduced unknown new seats on join/snapshot, takeover/snapshot, resume/join+snapshot and uncounted ghost on takeover/join. Admission now checks cancellation under lock and publishes/rotates the new opaque seat only after runtime join and full snapshot succeed. Any failed mutation cleans its avatar and affected old seat; empty rooms close. Cross-world old seat is retained until destination success. `go test -count=1 -race ./internal/realtime -run 'TestCancelledAdmission|TestUnrelatedRevocation|TestRevocationDuringAdmission|TestConcurrentSeats|TestTakeover'` PASS1.531s.
3. Explicit leave: deterministic test holds first command authorization, sends leave then closes the peer before the writer can process it. RED `explicit leave retained old seat: takeover required`. Reader now records received valid leave; after reader shutdown, deferred binding-aware Leave releases the matching seat even if acknowledgement could not be written. Stale IDs/epochs cannot release replacements. Targeted race test PASS1.540s. Client RED `leave must remain pending until server ack`; WorldSocket.close now resolves only after matching ack or reports unconfirmed after failure/timeout, and world picker switches are serialized behind acknowledged release. Frontend tests PASS8JS+16TS; typecheck PASS. Added actual acknowledged-leave→second-world protocol test for scoped verification.
4. HTTP shutdown: extracted actual serve lifecycle and reproduced RED `returned before handler drained: <nil>` and `drain timeout: <nil>`. Main now awaits both Serve and Shutdown completion before pool teardown, reports shutdown errors, and closes remaining HTTP connections on drain timeout. Focused handler/signal tests cover normal drain and timeout; target race run PASS1.644s.

All Go commands use `GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go` from apps/server. Transient loopback tests used approved escalation. Combined scoped verification is running; full broad suites are not repeated without need.

Fix-round scoped verification completed:

- `go test -count=1 -race ./internal/realtime ./internal/world ./cmd/server`: PASS realtime9.666s, world2.249s, cmd/server1.844s. Includes gated unrelated revocation, all six admission cancellation cases, received-leave+close, acknowledged leave→new websocket world, and drain/timeout regressions.
- With ignored DATABASE_URL sourced without printing, `go test -count=1 -race -tags=integration ./internal/application`: PASS18.000s (existing isolated-schema sustained20-client smoke across changed hub/transport).
- `go vet ./internal/realtime ./cmd/server` and `go build -o ../../.cache/server-t5 ./cmd/server`: PASS, silent.
- `npm test --workspace apps/web`: PASS8 JS+16 bundled TS tests, including matching-ack leave; `npm run typecheck --workspace apps/web`: PASS. `npm run lint --workspace apps/web`: PASS. `npm run build --workspace apps/web`: PASS147modules, JS317.06kB/gzip102.92kB,229ms. `git diff --check`: clean.
- Deliberate leave timeout/transport failure is explicitly `leave_unconfirmed`; a replacement join is not started on that result. UI offers reconnect to release/recover the seat, rather than claiming server release. Initial/transport disconnect grace remains30s. Simultaneous user leave and queued picker stop share the same ack promise; an already-cleared local socket is treated as successful release without wiping a newer socket.

Final fix-round handoff: ready for scoped re-review of the four Important findings. Added a Node-only test of the actual mountOnline controller (synthetic EventTarget/DOM/WebSocket, no browser or credentials): selecting world B while world A leave is unacknowledged creates no second socket; matching ack then creates B. Final `npm test --workspace apps/web` PASS8JS+17TS, `npm run typecheck --workspace apps/web` PASS. Final production build after the idempotent controller stop guard PASS147modules, JS317.07kB/gzip102.92kB,209ms. Final `git diff --check` clean. No production edits after those checks. Original staged snapshot retained; coordinator-owned PROGRESS/ACCEPTANCE and review files untouched. No remaining implementation gaps known for these four findings; browser acceptance remains explicitly NOT_RUN.

## Fix round 2 — local solo independent of server release

Read task-5-rereview1.md. Its remaining Important finding was: **“[P2] Returning to solo is now blocked by unconfirmed server release.”** Required distinction retained verbatim: **“detach from authoritative rendering/input and restore local solo when the user leaves or authentication becomes null, while separately retaining truthful pending-release/recovery state.”** New authenticated joins must still wait for confirmed release or valid recovery/explicit takeover.

Regression evidence (repository root): `npm test --workspace apps/web` RED: 8 JS PASS; bundled TS17 PASS/2 FAIL. Both `disconnected leave restores solo while pending release fences replacement` and `disconnected logout restores solo while pending release fences replacement` failed `local solo must detach without a release ack`, with retained WorldSocket adapter instead of null.

Frontend-only implementation:

- Controller tracks local `attached` separately from its retained socket/seat-release state. stop() immediately calls setOnline(null), clears authoritative latest state and restores local movement before awaiting any server ack. A failed close retains the recovery handle and shows `Solo lokal · Keluar belum dikonfirmasi...`.
- Replacement joins remain serialized and require acknowledged release. Pending release cannot start another world socket. Explicit recovery resumes the old seat solely to send leave; it never reattaches authoritative rendering/input or interrupts local solo. After its matching ack, a subsequent authenticated join is allowed.
- Authentication defaults to unknown/false until auth-change supplies a user. Auth-null immediately detaches, disables retry/takeover, and increments an intent generation. Queued joins recheck authentication and generation after awaited release, preventing a late acknowledgement from reconnecting a logged-out user.
- Self-review covered overlapping user leave/picker switches, logout during pending ack, recovery callbacks while locally detached, and retry after logout. Socket identity guards prevent old callbacks from updating a replacement. Browser timers/credentials were not used; actual mountOnline controller tests run with synthetic Node EventTarget/DOM/WebSocket fixtures.

First GREEN: `npm test --workspace apps/web` PASS8JS+19TS; `npm run typecheck --workspace apps/web` PASS. Expanded the disconnected-leave test to verify resume remains solo, sends world.leave on recovered snapshot, waits for ack, then permits later world B. Added a distinct logout-during-pending-switch regression; final frontend check output recorded below.

No backend edits, Go reruns, browser credential workflows, subagents, staging/commits, or coordinator ledger edits. Original round1 staged review index remains unchanged. Coordinator preview processes8080/4176 were not touched. Browser acceptance remains NOT_RUN.

Final round2 commands and actual results (repository root):

- `npm test --workspace apps/web`: PASS8 JavaScript tests and20 bundled TypeScript tests,0 failures. Includes preserved ack-switch, disconnected local leave, disconnected auth-null, honest pending/no replacement, recovery-without-reattachment→ack→later join, and logout fencing a queued switch despite later old ack.
- `npm run typecheck --workspace apps/web`: PASS, no diagnostics.
- `npm run lint --workspace apps/web`: PASS, no diagnostics.
- `npm run build --workspace apps/web`: PASS,147modules; JS317.33kB/gzip103.04kB;394ms.
- `git diff --check`: clean.

Ready for scoped round2 re-review. Limitation intentionally preserved: if server release cannot be confirmed, the local solo world is immediately usable but new authenticated joins remain gated until recovery/ack or an explicit valid takeover path. No automatic assumption that a server seat disappeared was added. Browser acceptance still NOT_RUN; tests above are synthetic controller evidence only.
