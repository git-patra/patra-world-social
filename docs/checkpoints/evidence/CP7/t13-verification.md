# T13 implementation report — review candidate

Status: **DONE_WITH_CONCERNS — implementation frozen for coordinator review.** Required local checks passed; provider/browser acceptance remains pending. This is not a T13 task-done or CP7 PASS claim.

## Scope and architecture

Implemented browser-only Warkop voice admission, provider integration, ephemeral table chat, lifecycle revocation and controls. No native app, LiveKit project provisioning, service purchase, preview/API restart, source-repository modification, commit, staging, push or deployment was performed. Root-owned PROGRESS/checkpoint/evidence files are not implementer changes.

- `voice.Service`: max eight reservations/seats per world/table; one seat/account across worlds; opaque 192-bit admission/media identity rotated per legal join; locally signed 60-second microphone-only JWT; authoritative session/member/ban/epoch/position/game fences.
- Reserve → journal insert outside state locks → reauthorize/recheck reservation → activate. A cancellation during the journal write wins; an insert failure/unknown commit never mints. Durable `voice_revocations` has no cascading user/world foreign keys and stores identity/room only. Startup revokes every retained identity before readiness, including a token never used to connect.
- Hub admission lease → hub mutex → voice mutex. SQL and provider network work stay outside hub/voice state mutexes. A dedicated reconciliation serialization mutex is worker-only. Local seat change/game-entry/position hooks synchronously revoke; the worker performs explicit provider removal/retries and room cleanup independently. Pending revocation globally blocks new voice admission/token issuance. World/game/chat owners never wait on provider network.
- `RemoveParticipant.RevokeTokenTs` is Unix seconds and rejects `nbf` strictly before cutoff, per the generated official protocol. Implementation passes local `now + 2s` explicitly, never the Cloud default. The cutoff covers same-second equality/small skew; refresh/in-flight races and actual Cloud rejection remain unverified. Opaque identities are never reminted after revocation. Successful removal precedes journal deletion. Failed cleanup remains durable/degraded. Existing audio may persist while provider removal is unavailable; no claim of instantaneous remote isolation is made.
- Only the explicitly configured application room prefix is swept. Room deletion has a reservation barrier against concurrent join. Deployment is one instance per database/provider namespace; distributed ownership is not implemented.
- Table chat uses actual active admission and authorized `voice.state` polling, with no world broadcast. Plain text, 500 code points, five-token bucket/one token per second, server/client IDs, last 200 visible messages and up to 4,096 session dedupe entries. At the dedupe safety bound, new messages fail rather than reaccept an evicted retry. Memory is deleted after five empty minutes, including before a late readmission exposes state. No durable chat/audio/token content.
- Owner/moderator table mute/kick is role-checked, same-table scoped and metadata-audited. Mute jobs are versioned; unmute is a participant action. Host follows oldest active admission.
- Shared separate table-zone contract binds to existing map ID/version/checksum; centers/radius are rendered and parity/bounds/reachability/nonoverlap are tested. Navigation map and game hashes remain unchanged.
- Browser lifecycle serializes leave before another join, cleans a possibly allocated seat after an ACK timeout, rejects late join/token/connect resurrection, starts listen-only, and preserves confirmed table chat on audio failure. Controls include mic permission only on explicit action, device selection, volume, deafen, keyboard/touch PTT, speakers, moderation and leave. Panel close, hidden tab, world disconnect and participant game entry disconnect audio/release admission. No provider configuration yields truthful unavailable UI.

## Changed implementation files

- Server dependencies: `apps/server/go.mod`, `apps/server/go.sum`.
- Voice package: `apps/server/internal/voice/service.go`, `reconcile.go`, `livekit.go`, `http.go`, `journal.go`, `zones.go`, `tables.v1.json`, and four test files (`service_test.go`, `livekit_test.go`, `http_test.go`, `zones_test.go`).
- Wiring: `apps/server/internal/application/application.go`, new `voice.go`, new `voice_integration_test.go`; `apps/server/internal/realtime/hub.go`, `delivery.go`, `connection.go`, `protocol.go`, new `voice.go`.
- Migration: `apps/server/internal/db/migrations/010_voice.sql`.
- Browser: `apps/web/src/features/voice/{VoicePanel.tsx,session.ts,microphone.ts,zones.ts,voice.css}`, `src/app.js`, `src/data.js`, `src/world.js`, `src/world/avatars.ts`, `tests/voice.test.ts`, authored `tests/voice.guest.e2e.spec.ts`, `scripts/test-map.mjs`, `package.json`, root `package-lock.json`.
- Contracts: `contracts/voice-tables.v1.json`, `contracts/protocol.schema.json`.
- Runbook: `docs/runbooks/voice.md`; this report.

## Dependency resolution and exact toolchain

Official docs checked: https://docs.livekit.io/frontends/reference/tokens-grants/ and https://github.com/livekit/server-sdk-go/blob/main/README.md . Current docs distinguish Cloud-only revocation, default cutoff buffer, refreshed tokens, ten-minute revocation retention and the need for explicit cutoff/replay testing.

Final pins: Go 1.26.8, LiveKit Go SDK v2.19.0, protocol v1.53.1, browser `livekit-client` 2.16.1. SDK 2.19 requires Go 1.26.8. CI already uses go-version-file; no workflow edit was required.

Approved task-only toolchain path: `/Users/patra/Project/patra-world-social/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go`. Verified official module checksum: `h1:dFjGhY6f8DTsY8/rLchUTDRRfjJP4pCrGCwxvHFT2Fg=`; go.mod checksum `h1:8wlg68NqwW7eMnI1aABk/C2pDYXj8mrMY4TyRfiLeS0=`. Observed `go version go1.26.8 darwin/arm64` from this explicit path and `go version go1.26.2 darwin/arm64` from unchanged `/opt/homebrew/bin/go`. `GOTOOLCHAIN=local` used for compatibility/final checks; global goenv/PATH untouched.

## RED → GREEN evidence and retained failures

1. Initial backend test file was accidentally created relative to `apps/server` with an extra `apps/server/` prefix. `GOCACHE=... /opt/homebrew/bin/go test ./internal/voice` failed `stat .../internal/voice: directory not found`. Moved the test into the correct directory and removed empty accidental directories. The next run failed compilation for missing implementation types (`Subject`, `JoinContext`, `Record`, etc.); this is scaffold RED, not a behavioral assertion pass.
2. One subsequent test command ran at repository root without `-C apps/server`: `go: cannot find main module`. Corrected all later commands to explicit module directory/`-C`.
3. Cached SDK 2.16.1/protocol 1.45.1 and downloaded protocol 1.45.6 lack `RevokeTokenTs`. Offline `go get` failed checksum DNS (`sum.golang.org: no such host`); npm offline install failed `ENOTCACHED` for registry metadata. Scoped approved downloads used the official Go/npm registries; no credentials/project were requested.
4. SDK v2.16.1 + protocol v1.53.1 failed compilation at `sipclient.go:314`; SDK v2.18.0 + protocol v1.53.1 failed at `sipclient.go:354`: `TransferSIPParticipant` returned `*livekit.TransferSIPParticipantResponse` where the older SDK expected `*emptypb.Empty`. Coordinator approved the task-scoped Go 1.26.8 + SDK 2.19 path. Earlier protocol v1.52.0/v1.51.0 downloads completed during investigation, but neither was selected or used as acceptance evidence.
5. Direct invocation of the downloaded Go module binary first failed `permission denied` because module extraction stripped executable bits. Applied executable permissions only to that project-local toolchain's bin/tools. No system toolchain update.
6. Dependency tests initially reported missing go.sum entries for `github.com/golang-jwt/jwt/v5` and `github.com/hashicorp/go-retryablehttp`; official `go mod tidy`/downloads resolved them. Final tidy first hit sandbox module-cache lock denial for `go-winio` and `otelhttp`; scoped approved tidy then exited 0.
7. First official SDK HTTP capture run hit sandbox `httptest: ... bind: operation not permitted`; reran with scoped loopback approval. The first capture assumed JSON and failed `invalid character '\x04'`/invalid protobuf response. Corrected the contract test to decode actual protobuf request and return protobuf response. The final contract verifies SDK admin JWT signature/scope and exact nonzero `RevokeTokenTs` on the wire; it does not assert Cloud enforcement.
8. Focused voice package then passed (`ok .../internal/voice 0.627s`). JWT claims verified actual signature, issuer, opaque subject, exact 60-second TTL, exact room, microphone-only publish, subscribe and explicit no data/metadata/admin grants. Webhook signature/tamper/duplicate, eight seats, isolation, reserve cancellation, journal failure, never-connected restart, removal retry, host transfer and scoped moderation are covered.
9. Focused actual application/database lifecycle test first failed at line 146 with `ErrOutside`: its reused account's asynchronous accepted lobby-return teleported after a test-only table teleport. Isolated the resume/logout scenario with a third synthetic account. Same focused command passed (`ok .../internal/application 0.772s`). No lobby production fix or preview account operation.
10. Two frontend behavioral regressions were written and observed RED: audio network failure incorrectly released authorized table chat; join ACK timeout did not send leave for a possibly allocated seat. `node --test .cache/voice-test.mjs` reported 4 pass/2 fail. Fixed lifecycle handling; repeated command reported 6 pass/0 fail. Existing cancellation, listen-only, release-before-switch and error distinction tests also pass.
11. Backend `TestChatExpiresBeforeReadmissionWithoutWaitingForWorker` observed RED with an old message resurrected after six empty minutes. Added activation-time expiry check; same focused command passed (`ok .../internal/voice 0.562s`). Also preserved session dedupe past the visible 200-message window and versioned repeated mute requests during self-review.

## Exact verification commands and outputs

All commands ran from `/Users/patra/Project/patra-world-social` unless stated otherwise. Private database settings were sourced with `set -a; source .cache/database.env; set +a`, never printed. Shell logs below contain no provider token/secret.

Focused backend (last successful full voice package before final checks):

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/voice
# ok patra-world-social/server/internal/voice 0.627s
```

Focused application:

```sh
set -a
source .cache/database.env
set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run TestVoiceUsesActualAdmissionLifecycleAndJournal -count=1
# ok patra-world-social/server/internal/application 0.772s
```

Focused chat expiry:

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/voice -run TestChatExpiresBeforeReadmissionWithoutWaitingForWorker -count=1
# RED: expired messages resurrected; GREEN: ok .../internal/voice 0.562s
```

Focused browser model (run after writing the two regressions, then after fixing them):

```sh
node --input-type=module - <<'JS'
import {build} from 'esbuild';
await build({entryPoints:['apps/web/tests/voice.test.ts'],outfile:'.cache/voice-test.mjs',bundle:true,platform:'node',format:'esm'});
JS
node --test .cache/voice-test.mjs
# RED: tests 6, pass 4, fail 2; GREEN: tests 6, pass 6, fail 0
```

Final full Go race integration, vet/build and browser results are appended below. Serial package scheduling `-p 1` was explicitly authorized because T12 exposed existing default-parallel chat fixture sensitivity. This is disclosed scheduling, not a claim of default-parallel execution.

## Unexecuted acceptance and concerns

- No actual LiveKit Cloud project, media connection, two-table/two-world audio isolation, initial-token replay, refreshed-token replay, concurrent refresh/removal, or provider clock-skew check was executed.
- No authenticated browser account/invite/credential workflow, injected browser authentication, microphone permission exercise, real audio device check, mobile real-device check or network-block browser acceptance was executed by the implementer.
- `voice.guest.e2e.spec.ts` is authored but not CLI-executed. Root separately performed guest-only CUA against the existing accepted T12 API; root owns `docs/checkpoints/evidence/CP7/t13-guest-preview.md` and screenshots. That confirms only guest/unavailable presentation, not the T13 backend or provider.
- Async remote revocation can lag local seat invalidation and cannot be guaranteed during a provider outage. Degraded blocks new voice access and preserves recovery evidence; it does not retroactively prove active media stopped.
- Single application instance per provider/database namespace is required. A distributed admission/journal lease is not implemented.
- Session dedupe has a documented 4,096-message safety bound and the browser sees latest 200 messages. Tables must empty for five minutes to reset that bound. This is a deliberate memory bound, not durable history.
- Invalid/partial LiveKit configuration fails application startup; completely absent configuration permits the normal unavailable mode. Runtime provider failures stay independent from game/world chat.
- The official SDK raises the server patch minimum and expands the dependency graph. Both older incompatible SDK combinations and the scoped toolchain installation are retained above.

See `docs/runbooks/voice.md` for operational sequencing and the explicit mandatory pending Cloud/device/replay acceptance matrix. Root review and commit remain outstanding.

## Final verification results (frozen candidate)

Additional retained final-check findings:

- First `go vet ./...` exited 1 for four unkeyed `world.Point` literals at `internal/voice/zones_test.go:31`. `go build ./...` in that same batch exited 0. Replaced only the test literals with keyed X/Y fields. Corrected vet/build both exited 0 with no output. The full race rerun had already started before the coordinator said a targeted rerun was enough; it was allowed to finish and also passed. No further broad Go checks were launched.
- Final frontend self-review found device enumeration could fail after successful mic activation and falsely set displayed mic state to off. The actual UI handler now uses `updateMicrophone` from `microphone.ts`. A focused regression exercised real `VoiceSession` activation plus rejected enumeration: RED was 7 tests / 6 pass / 1 fail (`false !== true` for displayed mic); GREEN is 7/7, also proving the explicit mute action stops publication and updates the exposed state. Device discovery is best-effort and cannot overwrite truthful active publication state.

Exact final Go commands (with private environment sourcing as shown earlier):

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race -tags=integration -p 1 -count=1 ./...
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server vet ./...
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server build ./...
git diff --check
```

Results: all exit 0. Vet/build/diff check have no output. Exact full race output:

```text
?   	patra-world-social/server/cmd/e2edb	[no test files]
ok  	patra-world-social/server/cmd/server	1.553s
ok  	patra-world-social/server/internal/application	32.173s
ok  	patra-world-social/server/internal/auth	6.231s
ok  	patra-world-social/server/internal/chat	1.850s
ok  	patra-world-social/server/internal/config	1.240s
?   	patra-world-social/server/internal/contracts	[no test files]
?   	patra-world-social/server/internal/db	[no test files]
ok  	patra-world-social/server/internal/games/duck	1.251s
ok  	patra-world-social/server/internal/games/model	1.289s
ok  	patra-world-social/server/internal/games/penalty	1.303s
ok  	patra-world-social/server/internal/games/race	1.771s
ok  	patra-world-social/server/internal/history	16.045s
ok  	patra-world-social/server/internal/http	1.466s
ok  	patra-world-social/server/internal/leaderboard	3.263s
ok  	patra-world-social/server/internal/lobby	1.995s
ok  	patra-world-social/server/internal/moderation	1.645s
ok  	patra-world-social/server/internal/realtime	9.627s
ok  	patra-world-social/server/internal/social	1.492s
?   	patra-world-social/server/internal/testutil	[no test files]
ok  	patra-world-social/server/internal/voice	1.434s
ok  	patra-world-social/server/internal/world	1.279s
ok  	patra-world-social/server/internal/worlds	2.108s
```

Exact final frontend commands:

```sh
node --input-type=module - <<'JS'
import {build} from 'esbuild';
await build({entryPoints:['apps/web/tests/voice.test.ts'],outfile:'.cache/voice-test.mjs',bundle:true,platform:'node',format:'esm'});
JS
node --test .cache/voice-test.mjs
npm run lint
npm run typecheck
npm test
npm run build
```

Results: all exit 0. Lint checks `src/app.js` and `src/world.js`; typecheck is `tsc --noEmit`. Focused voice/mic model test output:

```text
✔ voice joins listen-only and releases old connection before next admission (3.035875ms)
✔ cancelled pending admission never connects and releases server seat (7.021459ms)
✔ permission, device and network failures stay distinct (0.266459ms)
✔ table geometry is separate and nonoverlapping (0.111042ms)
✔ audio network failure keeps the authorized table chat seat until explicit leave (0.310375ms)
✔ join acknowledgement timeout is cleaned up as a possibly allocated seat (0.140583ms)
✔ successful publication remains visibly on when device enumeration fails, and explicit mute still stops it (0.830708ms)
ℹ tests 7
ℹ suites 0
ℹ pass 7
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 119.534834
```

Full browser test output:

```text

> patra-world-social@0.1.0 test
> npm test --workspace apps/web


> @patra-world-social/web@0.2.0 test
> node --test tests/*.test.js && node scripts/test-map.mjs

✔ password form counts Unicode code points and keeps whitespace (0.474125ms)
✔ isometric projection round trips fractional coordinates (0.735ms)
✔ ground navigation walks around an occupied tile without leaving the island (0.275666ms)
✔ station navigation stops at the closest reachable entrance (0.960208ms)
ℹ tests 4
ℹ suites 0
ℹ pass 4
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 56.672834
✔ shared Duck spawn and collision fixtures (1.570834ms)
✔ local guest sixty seconds, cooldown, dead target and persistent counter (0.30925ms)
✔ online view aims through delivered geometry at200ms RTT and55ms after arrival (1.162958ms)
✔ actual online controller/view follows measured50/100/200+jitter and capped320ms profiles (5.245375ms)
✔ online presentation hides expired/dead/stale targets and resets calibration on epoch (1.631584ms)
✔ empty board has three empty ranks and no invented winners; summaries link only to world podium (20.871917ms)
✔ committed event invalidates once, wrong world/rules and temporary results cannot promote a score; reconnect reloads (0.287084ms)
✔ late board response cannot restore data after world switch/logout; retry replaces stale requests (0.17475ms)
✔ public invalidation protocol rejects expanded private payload (0.306333ms)
✔ production component cancels board and detail together on denial and fences late responses through explicit retry (20.267709ms)
✔ production loaded detail is removed when a board refresh returns401 or404 and only a valid online recheck restores it (10.446291ms)
✔ production detail denial fences pending board and ignores late transient failures after a fresh retry (3.216542ms)
✔ production deep link ignores denial from another world and replaces detail request on route change (2.707792ms)
✔ 64 square map, reachable entrances and collision-free spawn (7.925584ms)
✔ map decoder fails closed (0.2125ms)
✔ motion preference does not change movement duration or position (0.181291ms)
✔ protocol validates envelope and penalty zones (0.32525ms)
✔ actual world controller preserves route when motion is toggled (549.990375ms)
✔ shared wire and fixed-point golden fixtures (0.41475ms)
✔ public avatars and world objects share one depth order (4.512083ms)
✔ co-located avatar nameplates occupy separate lanes (8.712792ms)
✔ shared semantic invalid maps fail at load boundary (2.256583ms)
✔ shared events validate required envelope fields and type set (0.480375ms)
✔ nonfinite zoom cannot corrupt camera (0.06675ms)
✔ online interpolation is bounded and full resync resets old epochs (0.33125ms)
✔ socket only sends when online and reconnect uses rotated seat without auto takeover (9.999ms)
✔ shared snapshot fixtures reject forged versions, unknown fields, missing self and duplicate avatars (0.209291ms)
✔ canvas pointer release preserves destination; only a directional hold sends stop (0.141416ms)
✔ explicit socket leave waits for matching acknowledgement before replacement join (0.287833ms)
✔ world picker serializes the real controller switch behind leave acknowledgement (0.7725ms)
✔ disconnected leave restores solo while pending release fences replacement (0.732291ms)
✔ disconnected logout restores solo while pending release fences replacement (0.263833ms)
✔ logout fences a queued world switch even when its old leave is later acknowledged (0.390667ms)
✔ chat resync merges exact durable IDs without duplicates or deletion resurrection (0.182875ms)
✔ chat send succeeds only after matching stored-message ack and failure preserves explicit error (0.6135ms)
✔ a delete received before its live message prevents later text resurrection (0.050208ms)
✔ actual chat form preserves an edited next draft while the submitted message awaits its ACK (0.83925ms)
✔ actual chat form retains failed draft and durable client ID for an explicit retry (0.146916ms)
✔ actual chat form ignores old ACKs after switching attachments while a new draft is pending (0.132834ms)
✔ production lobby view truthfully disables Start without a concrete engine (4.72775ms)
✔ lobby wire commands reject identity spoofing and socket dispatches resync events (0.380334ms)
✔ history does not invent a score or duration for an unknown-start aborted match (8.93575ms)
✔ actual socket dispatch forwards race and duck snapshots while fencing replaced sockets and worlds (0.370041ms)
✔ penalty local matches all 16 shared fixture outcomes (2.748125ms)
✔ penalty local locks first timely input and hides it until deadline (1.471167ms)
✔ penalty local equal sudden death pairs and maximum draw survive hidden tab catchup (0.554583ms)
✔ penalty local early winner and BOT blindness (0.260375ms)
✔ penalty guest session survives controller unmount and catches up on reopen (0.25425ms)
✔ penalty shared timelines and boundaries match Go fixture exactly (0.717166ms)
✔ penalty public view uses server time and locks chosen/spectator controls (9.566833ms)
✔ shared checkpoint gates reject reverse and shortcut laps (0.605166ms)
✔ guest countdown, stale input expiry and full bot race match server semantics (40.071875ms)
✔ bounded catchup never replays expired throttle (0.169459ms)
✔ fixed point acceleration steering and brake shared vectors (0.521042ms)
✔ prepared grid is stable and permutes identities (0.142958ms)
✔ guest session keeps race identity on reopen and never extends absolute deadline (0.33675ms)
✔ race controls are unavailable to spectators and saved result claims need server confirmation (10.996417ms)
✔ race rules cannot omit a physics coefficient (0.204167ms)
✔ finished race clock freezes at authoritative finish (1.969208ms)
✔ remounted guest controller accepts its first controls after a high prior sequence (6.702541ms)
✔ voice joins listen-only and releases old connection before next admission (0.719042ms)
✔ cancelled pending admission never connects and releases server seat (3.945291ms)
✔ permission, device and network failures stay distinct (0.631875ms)
✔ table geometry is separate and nonoverlapping (0.241833ms)
✔ audio network failure keeps the authorized table chat seat until explicit leave (0.753667ms)
✔ join acknowledgement timeout is cleaned up as a possibly allocated seat (0.3495ms)
✔ successful publication remains visibly on when device enumeration fails, and explicit mute still stops it (0.858208ms)
ℹ tests 67
ℹ suites 0
ℹ pass 67
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 743.18375
```

Production build output:

```text

> patra-world-social@0.1.0 build
> npm run build --workspace apps/web


> @patra-world-social/web@0.2.0 build
> vite build

vite v8.3.4 building client environment for production...
transforming...
✓ 76 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                               7.59 kB │ gzip:   2.64 kB
dist/assets/index-Cps2Rda3.css               64.10 kB │ gzip:  15.16 kB
dist/assets/index-7k7gXYOq.js               371.59 kB │ gzip: 120.15 kB
dist/assets/livekit-client.esm-_2wpeQoP.js  398.26 kB │ gzip: 102.58 kB

✓ built in 211ms
```

Final local logs are `.cache/t13-go-race-final.log`, `.cache/t13-go-vet-final.log`, `.cache/t13-go-build-final.log`, `.cache/t13-voice-model-final.log`, `.cache/t13-web-{lint,types,test,build}-final.log`. Earlier full race output remains `.cache/t13-go-race.log`; the failed vet output remains `.cache/t13-go-vet.log`. These are implementation evidence only. Authenticated browser/provider/device/replay acceptance remains unexecuted as listed above; root review and commit are still required.

## Dependency command ledger

Dependency commands below were executed from `apps/server` unless a different working directory is stated. They are not provider acceptance checks.

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOPROXY=off /opt/homebrew/bin/go get github.com/livekit/server-sdk-go/v2@v2.16.1
# FAIL: sum.golang.org DNS unavailable in sandbox.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go get github.com/livekit/server-sdk-go/v2@v2.16.1 github.com/livekit/protocol@v1.45.6
# exit 0, official dependency download; protocol lacks required field.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go list -m -json github.com/livekit/protocol@latest github.com/livekit/server-sdk-go/v2@latest
# protocol v1.53.1 / Go1.26.0; SDK v2.19.0 / Go1.26.8.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build /opt/homebrew/bin/go get github.com/livekit/protocol@v1.53.1
# exit 0; generated request exposes RevokeTokenTs int64.
```

From repository root:

```sh
npm install --workspace apps/web --offline --ignore-scripts --cache /Users/patra/.npm livekit-client@2.16.1
# FAIL: ENOTCACHED registry metadata; no log write permission in global npm cache.
npm install --workspace apps/web --ignore-scripts --cache .cache/npm livekit-client@2.16.1
# exit 0: added 14 packages, audited 140, found 0 vulnerabilities.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local /opt/homebrew/bin/go -C apps/server list -m -json github.com/livekit/server-sdk-go/v2@v2.18.0 github.com/livekit/server-sdk-go/v2@v2.17.0
# both declare Go1.26; v2.18 tested but incompatible with protocol1.53.1.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local /opt/homebrew/bin/go -C apps/server get github.com/livekit/server-sdk-go/v2@v2.18.0
# exit 0; subsequent compile failed as retained above.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local /opt/homebrew/bin/go -C apps/server mod download -json github.com/livekit/protocol@v1.52.0 github.com/livekit/protocol@v1.51.0
# exit 0; investigation only, neither selected.
```

Approved toolchain module download from `apps/server`:

```sh
GOMODCACHE=/Users/patra/Project/patra-world-social/.cache/toolchains GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local /opt/homebrew/bin/go mod download -json golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64
# exit 0; official module checksums/path recorded above.
```

From repository root after the initial non-executable-binary failure:

```sh
chmod u+x .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/gofmt .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/pkg/tool/darwin_arm64/*
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server get github.com/livekit/server-sdk-go/v2@v2.19.0
# exit 0; go minimum 1.26.2 -> 1.26.8, SDK 2.18.0 -> 2.19.0.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server mod tidy
# first sandbox run failed cache locks; approved retry exit 0.
```

Older-toolchain `go mod tidy` runs with the same GOCACHE/GOTOOLCHAIN flags resolved the initially missing JWT/webhook checksum entries before the patch upgrade. All dependency downloads used scoped tool approval after actual sandbox DNS/cache restrictions; none configured or contacted a LiveKit media project.

Final freeze: no implementation edits after the last reported checks. Local code/tests/runbook are ready for independent coordinator review. **Cloud audio/replay and authenticated/real-device browser acceptance remain pending.**

## Review correction round 1 — R1–R4 (2026-10-27)

This section supersedes the original 4,096-message quota and original frontend coverage statements above. Historical results remain as recorded; they do not establish coverage for these repairs. Read all four Important findings in `task-13-review.md`. The coordinator approved the bounded sequence protocol before implementation. All round-1 fixes are unstaged. No index/HEAD change, process restart, browser action, service provisioning, credential workflow, or remote operation was performed by this implementer.

### Repairs and rationale

- **R1:** Install unknown provider identities as revoked, journal-pending entries before Save. Retain them on a precommit failure or a write which persists then returns an error. Cleanup retries Save, explicit provider removal/cutoff, then journal deletion; participant-list disappearance never establishes revocation. New admission and token issuance remain gated until all stages succeed. The new journal fixture actually writes its row before `afterSave` returns an error, and asserts whether the row exists. Both unknown-participant and initial-admission uncertain writes cover both outcomes. Initial uncertain writes never activate a seat or mint a token. No provider/SQL calls were moved under hub/voice state locks.
- **R2:** Publish a spectator membership mirror from the lobby owner before acknowledging its operations. `VoiceExcluded` includes players and spectators while `Busy` remains player-only. Hub admission/observation and `SetActivity` use the wider voice exclusion; entering a running game as spectator revokes an existing seat, rejects a later voice join/token, and releases exclusion after leaving. The application regression uses actual application/hub/lobby/database wiring with an isolated test clock and provider fixture. It does not exercise Cloud media.
- **R3:** Remove the 4,096-message lifetime quota. A canonical `<currentAdmissionId>:<positive JS-safe integer>` is the only voice chat ID. Each admission stores a monotonic high-water mark and its last **32 ACK payloads**. Only the exact next sequence creates; recent retries return the original ACK; an evicted processed sequence returns stable code **`voice_retry_expired`** and never reposts. Gaps, wrong admission, leading zeros/signs/exponents/zero and values above 9007199254740991 are rejected. Go/browser envelope validation and schema documentation reflect this contract; current-admission and exact-next checks remain authoritative service semantics. `voice.state.nextMessageSeq` synchronizes the client monotonically within that admission. Latest **200** table messages remain visible; ACK payloads clear on revocation, visible table messages expire after five empty minutes. The journal remains identity/room only.
- **R4:** The actual `VoicePanel` retains one pending submission's text/ID and scopes its completion to a specific socket/admission context object. A success clears only the still-matching submitted draft. Failure retains the exact prior submission for an explicitly labeled retry while preserving an edited next draft. Obsolete success/failure callbacks cannot change replacement draft, status or pending state. A retry-expired response clears the old pending submission, explains it was already processed and resynchronizes before a new user submission; it never automatically reposts old text. This permits continued chat without table-empty/rejoin recovery.
- **Typecheck correction:** The prior `tsconfig.json` include list covered only imported voice geometry, not VoicePanel/session/microphone/voice tests. Earlier typecheck PASS claims therefore had narrower coverage. Explicit T13 production TS/TSX and both test/E2E includes are now added; actual typecheck passes and `--listFilesOnly` confirms all six voice files. Compiling the guest E2E source is not execution of that E2E.

### Files changed in this correction round

`apps/server/internal/application/lobby.go`, `voice_integration_test.go`; `internal/lobby/service.go`; `internal/realtime/connection.go`, `delivery.go`, `hub.go`, `protocol.go`, `protocol_test.go`, `voice.go`; `internal/voice/reconcile.go`, `service.go`, `service_test.go`; `apps/web/src/features/voice/VoicePanel.tsx`, `src/net/socket.ts`, `src/world/avatars.ts`, `tests/voice.test.ts`, `tsconfig.json`; `contracts/protocol.schema.json`; `docs/runbooks/voice.md`; this report. Server paths after the first prefix are under `apps/server/`. Root-owned PROGRESS/checkpoint/evidence changes were not authored or modified by this implementer.

### RED → GREEN and retained execution failures

- R1 focused test initially failed both real fixture outcomes with `disappearing participant incorrectly restored readiness`; after retaining the pending entry it passed (`ok .../internal/voice 0.587s`). The final uncertain-admission fixture was also strengthened after the affected suite and rerun under race below.
- R2 actual application test initially failed `spectator entry kept existing voice admission`; after the wider voice fence it passed (`ok .../internal/application 0.769s`). Final affected race integration includes this regression.
- R3 focused command passed `TestChatContinuesBeyond4096WithBoundedRetryWindow`, canonical sequence/gap/current-admission cases and the visible-window retry case (`ok .../internal/voice 0.396s`). It creates 4,200 messages while simulating rate-bucket recovery, checks recent/evicted retries, then successfully creates message 4,201 with bounded retention and state next sequence 4,202. The preceding focused RED invocation's result was truncated during context rollover; no specific RED assertion/pass is claimed from that missing output.
- R4 real component lifecycle regressions initially produced **67 pass / 3 fail**: ACK deleted edited `draft B`; obsolete ACK deleted replacement `new`; explicit retry changed ID and text from A to B. After the fix the component cases passed, including both success and error after either socket replacement or same-socket admission replacement. Stale state cannot rewind sequence; retry-expired recovery preserves edited B and emits no automatic chat request. Final tests include these cases.
- Execution mistakes retained: `npm run test:map --workspace apps/web` failed because no `test:map` script exists. A read of `apps/web/package.json` from the `apps/web` working directory failed `No such file or directory`. An early test tree traversal used a default argument during recursion, causing three `RangeError: Maximum call stack size exceeded` errors; the test harness traversal was fixed before the meaningful behavioral RED run. `node scripts/test-map.mjs` from repository root failed `MODULE_NOT_FOUND`; reran from `apps/web`. Some exploratory commands printed a tail after the failing test and therefore returned the tail's exit code; the retained logs/counts, not those wrapper exit codes, establish RED. Final affected Go wrapper preserves the actual test exit code, and final frontend logs show zero failures.

### Executed verification commands and outputs

Working directory is repository root unless stated otherwise. No whole-repository Go suite was repeated for this fix round. Go is the previously approved project-scoped official1.26.8 toolchain; system Go and global environment remain unchanged.

```sh
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/voice -run TestUnknownIdentitySaveFailureSurvivesDisappearance -count=1
# RED in both committed=false/true: disappearing participant incorrectly restored readiness
# GREEN: ok .../internal/voice 0.587s

# Private database environment sourced with set -a / set +a, never printed.
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run TestVoiceExcludesRunningMatchSpectatorsInBothOrders -count=1
# RED: spectator entry kept existing voice admission
# GREEN: ok .../internal/application 0.769s

GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test ./internal/voice -run 'TestChatContinuesBeyond4096|TestChatRequiresCanonical|TestChatDedupe' -count=1
# ok patra-world-social/server/internal/voice 0.396s

# From apps/web; meaningful R4 RED and initial GREEN:
node scripts/test-map.mjs > ../../.cache/t13-round1-frontend-red.log 2>&1
# tests70 pass67 fail3 (behavioral RED after fixing the test traversal)
node scripts/test-map.mjs > ../../.cache/t13-round1-frontend-green.log 2>&1
# tests70 pass70 fail0

# Final affected Go integration; approved scoped database/loopback execution:
set -a; source .cache/database.env; set +a
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race -tags=integration -p 1 ./internal/voice ./internal/application ./internal/realtime ./internal/lobby -count=1 > .cache/t13-round1-go-race.log 2>&1
# Wrapper saved result=$?, printed log, exited $result: exit0.
```
ok  	patra-world-social/server/internal/voice	1.794s
ok  	patra-world-social/server/internal/application	30.735s
ok  	patra-world-social/server/internal/realtime	9.928s
ok  	patra-world-social/server/internal/lobby	2.721s

```sh
# After strengthening the initial-admission fixture to actual persisted/not-persisted outcomes:
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -race ./internal/voice -run 'TestJournalFailureFailsClosed|TestUnknownIdentitySaveFailureSurvivesDisappearance' -count=1
# ok patra-world-social/server/internal/voice 1.680s

GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server vet ./internal/voice ./internal/application ./internal/realtime ./internal/lobby
# exit0; no output
GOCACHE=/Users/patra/Project/patra-world-social/.cache/go-build GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server vet -tags=integration ./internal/voice ./internal/application ./internal/realtime ./internal/lobby
# exit0; no output (includes integration test source)

npm test --workspace apps/web > .cache/t13-round1-frontend-tests.log 2>&1
# tests4 pass4 fail0; bundled tests71 pass71 fail0; total75 passed
npm run lint --workspace apps/web
# exit0; node --check src/app.js && node --check src/world.js
npm run build --workspace apps/web > .cache/t13-round1-frontend-build.log 2>&1
# exit0, Vite8.3.4: 76 modules, built in388ms
npm run typecheck --workspace apps/web
# exit0; this earlier invocation still had the narrower include list
npm run typecheck --workspace apps/web > .cache/t13-round1-voice-typecheck.log 2>&1
# exit0 AFTER explicit T13 include expansion; tsc --noEmit

# From apps/web:
node ../../node_modules/typescript/bin/tsc --listFilesOnly | rg 'features/voice|tests/voice'
# .../src/features/voice/zones.ts
# .../src/features/voice/session.ts
# .../src/features/voice/microphone.ts
# .../src/features/voice/VoicePanel.tsx
# .../tests/voice.test.ts
# .../tests/voice.guest.e2e.spec.ts
```

### Self-review and limitations after repairs

Reviewed the actual pending-entry lifecycle, absence-versus-revocation distinction, game/spectator mirror publication and lock direction, admission-scoped sequence contract, bounded ACK/chat retention, explicit retry error transport, and UI completion fences. Component tests execute the real VoicePanel hooks/render output through the existing deterministic Node lifecycle harness. They are **not DOM, browser, permission or media tests**.

No actual Cloud/audio/replay/device/authenticated browser acceptance was executed. The official SDK/token/wire tests remain mechanism evidence only. Cloud cutoff enforcement and retention, never-connected/absent-room removal, refresh concurrent with cutoff, audio isolation, real desktop/mobile device/permission/network behavior, and beta/CP7 voice acceptance remain pending under the existing authorized deferral. No fake/local provider is treated as Cloud acceptance. Root's earlier guest-only preview remains separate evidence against the prior accepted API.

Round-1 code frozen for coordinator review after the checks above. No T13 completion or CP7 PASS claim.

Final read-only integrity check: `git diff --check` exited0 with no output. `git diff --cached --binary | shasum -a 256` matched the saved pre-fix value exactly: `96f25bb7cc39e70fd51e80377c12e6e7f4fce18cd9f8be35a080697728a583d2`. Reviewed staged candidate remains unchanged; all fixes remain unstaged.
