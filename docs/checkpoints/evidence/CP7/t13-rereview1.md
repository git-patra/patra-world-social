# T13 scoped rereview — correction round 1

Reviewed the complete 1,595-line `task-13-fix1.diff` against the unchanged staged candidate `b2f5394c98cd61ecdc6f2d13645798932e6cf88d`, the appended round-1 implementation report, the original R1–R4 review, and the brief/context including the approved R3 sequence-protocol ruling. Supporting reads checked current line locations and the component lifecycle harness. No tests were rerun and no source, index, HEAD, runtime or browser state was changed. This report is the only rereview output.

**Spec readiness:** the four implementation findings are addressed under the binding rulings. Full T13 acceptance remains pending the explicitly deferred external checks.

**Code readiness:** approve these corrections for coordinator integration. No new actionable regression was established within the fix diff. This is not a T13 task-done or CP7 PASS verdict.

**Counts:** 4 ADDRESSED, 0 NOT ADDRESSED; 0 new Critical, 0 new Important, 0 new Minor findings.

## Finding disposition

### R1 — ADDRESSED

`apps/server/internal/voice/reconcile.go:104–129` now installs an unknown identity as revoked and journal-pending before attempting storage. `cleanupRevocation` at `:195–221` preserves it across failed saves, explicit removals and journal deletions; listing absence cannot settle it. Pending entries continue to block readiness and token/admission issuance. Initial admission write failures also mark the identity journal-pending before revocation.

`TestUnknownIdentitySaveFailureSurvivesDisappearance` covers both failure before persistence and persistence followed by an error response, removes the participant from subsequent listings, verifies continued degradation and blocked mint/join, then exercises removal failure and successful recovery. `TestJournalFailureFailsClosed` now also models and asserts both actual persistence outcomes for initial admissions. This corrects the earlier coverage overstatement about an error merely named “commit outcome unknown.” Storage and provider work remain outside hub/voice state mutexes.

### R2 — ADDRESSED

`apps/server/internal/lobby/service.go:166–179` publishes the spectator mirror before acknowledging owner operations, while `:402–416` exposes combined player/spectator voice exclusion without changing player-only `Busy`. Application wiring supplies this predicate to the hub. `SetActivity`, authoritative voice position checks, and snapshot observations now use it.

The actual application/database integration regression `TestVoiceExcludesRunningMatchSpectatorsInBothOrders` establishes a running match, verifies spectator entry revokes an existing voice seat and denies its token/chat, rejects a later voice join/token, confirms player-only `Busy` remains false, and verifies exclusion releases on leaving. The new mirror reads take only the existing short state lock and do not call the hub/provider from the lobby owner.

### R3 — ADDRESSED

`apps/server/internal/voice/service.go` removes the 4,096-message quota and implements the approved admission-bound positive JS-safe sequence, exact-next acceptance, high-water mark, latest 32 ACK payloads, and latest 200 visible table messages. A processed sequence outside the ACK window produces `voice_retry_expired` without another post; gaps, noncanonical/range violations and wrong admission prefixes fail. Revocation clears the admission ACK payload and transport context. The journal remains identity/room only.

The Go/browser validators and schema documentation describe the new unreleased voice-only ID contract. The error code is carried through the existing command-error transport. `VoicePanel.tsx:19–30` synchronizes the current admission's counter monotonically, preserves a pending submission for retry, and treats an expired retry as terminal before state resynchronization and further explicit user intent.

Coverage includes 4,200 accepted messages, bounded retention, recent retry identity, evicted retry rejection, successful message 4,201, exact next sequence, canonical/gap/prefix/range rejection, stale-state non-rewind and retry-expired UI recovery without automatic repost. These resolve the availability defect within the approved bounded-memory design.

### R4 — ADDRESSED

`apps/web/src/features/voice/VoicePanel.tsx:18–30` captures a submission's immutable text/ID/sequence in a socket/admission-specific context. Completion checks context identity before changing UI state, and success clears only a still-matching draft. An edited next draft survives; retry resends the original submission rather than substituting the newer text. Socket/admission replacement invalidates obsolete successes, failures and finalizers.

The added tests execute the actual `VoicePanel` through the deterministic component lifecycle harness and cover edited draft preservation, stale state, explicit retry payload identity, expired retry recovery, and both success/error after socket replacement or same-socket admission replacement. The harness exercises component hooks/render output; it is not a DOM renderer, browser, microphone or provider test.

## Evidence and acceptance limits

The round-1 report records passing affected Go integration under `-race -tags=integration -p 1 -count=1` for voice/application/realtime/lobby, the strengthened uncertain-write tests under race, affected vet with and without integration tags, 75 frontend tests, lint and production build. Explicit voice production/test/E2E TypeScript includes were added and the report records a passing expanded typecheck plus file-list verification. Earlier narrow typecheck coverage and retained execution failures are corrected/disclosed rather than relabelled. I reviewed this evidence and test source; I did not independently rerun the commands.

Cloud audio isolation, initial/refreshed token replay, refresh/removal races, cutoff/retention behavior, never-connected or absent-room removal, authenticated browser workflows, and real desktop/mobile permission/device/network acceptance remain unexecuted. Local SDK/claims/wire tests and the component harness cannot establish those results. Root's prior guest-only unavailable preview remains separate evidence against the earlier API. No provisioning or credential/browser workaround is requested by this rereview.

Unrelated existing obligations and observations remain deferred; this rereview assesses R1–R4 and regressions introduced by their correction diff only.
