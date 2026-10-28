# T13 independent spec and code review

Reviewed base `5ac08783d06aa8202c664f6a01c311bc628a94b1`, packaged staged tree `b2f5394c98cd61ecdc6f2d13645798932e6cf88d`. Read the complete task brief, binding review context, implementation report and 3,728-line packaged diff. Supporting reads were limited to the actual realtime/lobby integration and installed LiveKit SDK behavior. No source/index/HEAD/runtime/browser changes or test-suite reruns were performed; this report is the only review output.

**Spec readiness: changes required; T13 completion/CP7 voice acceptance remains pending. Code readiness: changes required before accepting this candidate.** Four Important findings follow. No Critical finding was established. The missing Cloud and authenticated/device acceptance is an explicitly authorized deferral, not evidence of a failed provider test, and not a reason to provision a service now.

## Important findings

### R1 — Retain unknown identities when their journal write fails

**Location:** `apps/server/internal/voice/reconcile.go:125–128`, with readiness reset at `:171–172` and one-time journal loading at `:22–34`.

When reconciliation finds an unknown participant, it sets `healthy=false`, then tries to save its identity. A save error immediately continues without installing a revoked/pending entry. Trigger: discover an unknown identity in a managed room, make `journal.Save` fail (including a write which commits but returns an error), and have that participant disconnect before the following successful room listing. The next sweep finds no participant and no pending entry, resets `healthy=true`, and resumes admission/token issuance even though **no explicit removal/cutoff was ever requested for that identity**. A commit-unknown row also remains ignored until process restart because journal loading runs only during initialization. Disappearance from the participant list does not establish token revocation; the approved crash journal exists specifically because tokens can outlive visible connections.

Keep the identity in in-memory pending revocation state before attempting its journal write, retain it on every error, and retry the necessary durable save/removal/deletion stages. Readiness must remain degraded until explicit cleanup completes. Add a targeted case where an unknown participant disappears after a failed save, plus a save-that-persists-then-errors variant; verify no new seat/token is issued before successful cleanup. This finding concerns application control flow, not an assertion about untested Cloud replay behavior.

### R2 — The game fence omits spectators of an already-running match

**Changed locations:** `apps/server/internal/realtime/delivery.go:147–154`, `apps/server/internal/application/voice.go:27–28`, and `apps/server/internal/realtime/voice.go:22–26`.

**Supporting seam:** `apps/server/internal/application/lobby.go:183–193` passes `g.Busy(user)` to `SetActivity`; `apps/server/internal/lobby/service.go:307–325` appends spectators without calling `claim`, so `Busy` remains false.

Trigger: a user with an active voice seat joins an already-running match as spectator through the authenticated command interface. `SetActivity(false)` does not invoke `VoiceRevoke`, the avatar remains outside the `playing` fence, and subsequent voice state/token/chat calls continue to pass while the user remains at the table coordinates. The reverse order also works: a spectator can acquire a new voice seat. The browser's `match.started` listener does not repair the backend boundary, and joining an already-started match does not require another start event.

The brief requires game entry to release voice and says audio must not continue into a game; it contains no spectator exception. Add an authoritative voice-exclusion check covering spectator participation as well as player occupancy, and invoke local revocation on spectator entry. Keep its lifecycle distinct from player-only game controls if needed. Test existing-admission spectator entry, spectator late join/token, and release when spectating ends. If spectators are intentionally allowed to remain in table voice, obtain an explicit spec ruling rather than presenting the current player-only fence as complete.

### R3 — The 4,096-message cap permanently disables an occupied table's chat

**Location:** `apps/server/internal/voice/service.go:349–352`; reset behavior at `:216–219` and `apps/server/internal/voice/reconcile.go:174–177`. Disclosed in `docs/runbooks/voice.md:34`.

Every unique message consumes a dedupe entry for the whole table session. At 4,096 entries all further new messages fail, regardless of subsequent rate-limit recovery, until **every participant leaves for five minutes**. Eight admitted users sending at the permitted sustained rate can exhaust a table in approximately 8.5 minutes; one user can do so in approximately 68 minutes. Remaining seated keeps the denial in effect indefinitely. This is a product availability change beyond the specified per-user rate limit and ephemeral retention rule, not merely a memory implementation detail. It also undermines table chat as the fallback during media trouble.

A bounded visible window and bounded memory are reasonable, but preserve continued chat for an occupied table with a bounded, explicit retry/dedupe protocol, or obtain a concrete product ruling for a session quota and recovery behavior. Do not simply evict arbitrary dedupe IDs and accept old retries again. Add boundary/recovery coverage. The report accurately discloses the limit, but disclosure alone does not establish spec compliance.

### R4 — A delayed chat ACK deletes the user's next draft

**Location:** `apps/web/src/features/voice/VoicePanel.tsx:54`.

Trigger: submit draft A, edit the still-enabled textarea to draft B while A's `gameCommand` awaits acknowledgement, then receive A's success. The handler unconditionally calls `setDraft('')` and clears `draftID`, deleting unsent B. The completion is also not scoped to the socket/admission, so an old request can clear a later draft after an attachment change within the same mounted panel. Existing world-chat coverage in the report explicitly protects this behavior, but the new table-chat handler does not reuse that protection.

Capture the submitted text, client ID, socket and admission generation; clear only the matching unchanged draft and ignore completions from an obsolete attachment/admission. Retain the same submitted ID for an explicit retry. Add a component-level delayed-ACK regression for an edited next draft and an attachment change, rather than only testing `VoiceSession` helpers.

## Minor observations and coverage precision

- The provider, admission and journal boundaries are sensibly separated. The room cleanup reservation barrier and opaque per-admission identities address the ordinary local join/deletion race; provider network work does not run under hub/voice state mutexes. Locally signed token creation under the fence is not a provider network request.
- The actual Go token test verifies the JWT signature and grants, and the local Twirp capture verifies the official SDK's explicit `RevokeTokenTs` wire field. The installed protocol describes Unix seconds and `nbf` strictly before the cutoff. These support the implementation mechanics only. They do not verify Cloud enforcement, future/in-flight refresh cutoffs, revocation retention, or removal of a never-connected identity whose room does not exist.
- `service_test.go:215–223` names an error “commit outcome unknown”, but its fake journal returns before adding a row. It does not model a committed write with an error response. The real application test covers ordinary successful journal lifecycle, not that uncertain-commit case. Strengthen this coverage with R1; do not count the error label as evidence that both outcomes were exercised.
- The new frontend tests execute lifecycle and microphone helpers. They do not render the actual `VoicePanel`, exercise real microphone permission, or prove asynchronous DOM/media cleanup. `voice.guest.e2e.spec.ts` is correctly described as authored and unexecuted.

## Compliance and operational assessment

Implemented local mechanisms include two eight-seat tables, one voice seat per account, authoritative session/member/ban/epoch/position checks, rotated opaque identities, journal-before-token issuance, 60-second microphone-only tokens, explicit removal cutoff/retry, signed webhook body validation, managed-room sweeping, default listen-only connection, explicit mic/PTT controls, deafen, device selection, host transfer, scoped audited moderation, and table-only memory chat with safe text rendering. No change to world-chat persistence or navigation/result map checksum was identified in this scope. R1 and R2 prevent treating the revocation boundary as complete; R3 is an unresolved spec deviation and R4 is a concrete UI data-loss defect.

The single application instance/database/provider namespace limit is consistent with the existing in-memory ownership architecture and is explicitly documented. Partial invalid voice configuration failing startup is an operational choice distinct from a runtime provider outage; deployment must validate it before replacing a working instance. The approved Go patch minimum increase and scoped toolchain are disclosed, with earlier failed SDK combinations retained. No system toolchain or provisioning acceptance is inferred.

The worker's asynchronous removal and bounded network calls cannot guarantee immediate cessation during a provider outage. The runbook correctly states this and keeps actual Cloud/device acceptance pending. `now+2s` is a proposed cutoff margin, not a proven skew bound. Startup/never-connected removal, refresh concurrent with removal, initial and refreshed token replay, two-table/two-world audio isolation, and real desktop/mobile permission/device/network/playback tests remain mandatory before beta/CP7 completion. Root's guest-only CUA preview against the prior accepted API establishes unavailable presentation only.

Reported verification was reviewed, not rerun: full Go integration under `-race -tags=integration -p 1 -count=1`, corrected vet/build, frontend lint/typecheck/build and 71 frontend tests passed according to the retained report. The earlier vet failure and explicitly serial Go scheduling remain visible. Those results are useful baseline evidence but do not cover the four interleavings/limits above or substitute for provider/browser acceptance. Existing T16 and T12 deferred obligations remain outside this review.

After repairs, run focused regressions for these findings and the required affected checks. Continue to label Cloud/authenticated/real-device acceptance pending under the user's existing constraints; do not mark T13 complete solely from local checks.
