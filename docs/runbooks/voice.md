# Warkop voice operations and acceptance

Voice is implemented for browser web (including responsive mobile), with LiveKit Cloud as the media provider. No Cloud project or staging credentials were available during T13. Local fixtures, JWT verification and SDK HTTP capture are implementation checks, not proof of real audio isolation or provider token replay protection.

## Runtime and configuration

The server requires Go **1.26.8** for `github.com/livekit/server-sdk-go/v2 v2.19.0`; the explicit protocol dependency is `github.com/livekit/protocol v1.53.1`. Browser SDK: `livekit-client 2.16.1`. CI already reads the version from `apps/server/go.mod`.

For the approved local task only, the official toolchain was downloaded as `golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64` under `.cache/toolchains`. Module checksum: `h1:dFjGhY6f8DTsY8/rLchUTDRRfjJP4pCrGCwxvHFT2Fg=`. System Go and global Go environment were not changed. Reproduce with:

```sh
GOMODCACHE="$PWD/.cache/toolchains" GOTOOLCHAIN=local /opt/homebrew/bin/go mod download -json golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64
chmod u+x .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/* .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/pkg/tool/darwin_arm64/*
GOTOOLCHAIN=local .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go version
```

Configure these server-only environment variables using the deployment's secret manager. Never put secrets or issued tokens in logs, screenshots, repository files, frontend variables or browser storage:

- `LIVEKIT_URL`: `wss://<project>.livekit.cloud` (Cloud only).
- `LIVEKIT_API_KEY` and `LIVEKIT_API_SECRET`.
- `LIVEKIT_ROOM_PREFIX`: a unique deployment namespace matching `patra-<8–40 lowercase letters/digits/hyphens>-`. Never share a namespace between independent applications or databases.
- Provider signed webhook URL: `https://<app>/api/v1/voice/webhook`.

Run migrations before startup. `010_voice.sql` adds only the durable revocation journal, without cascading account/world foreign keys. The current ephemeral hub/table implementation requires **one application instance per database/provider namespace**. Do not run two writers against this journal. Multi-instance seat ownership and a distributed lease are outside this implementation.

When all LiveKit settings are absent, the application runs with voice `unavailable`; it issues no admission or token. Invalid partial/unsafe configuration fails startup with a configuration error. Provider outages after startup degrade voice while game and world chat continue. `GET /api/v1/voice/status` reports `unavailable`, `degraded`, or `ready`; it exposes no credentials. Provider initialization/reconciliation runs independently after result recovery and does not consume the T12 ten-second recovery budget.

## Admission, geometry and chat

The independently versioned `contracts/voice-tables.v1.json` binds its interaction geometry to the exact existing map ID/version/checksum. Its Go embedded copy and browser renderer are checked for parity. The navigation map and game result map checksum are unchanged. Meja Depan and Meja Pojok each allow eight seats; one account may reserve only one voice seat across worlds/tables.

The websocket command boundary uses the existing exact-origin, cookie/session, membership, ban, epoch and input-limit checks. `voice.join`, `voice.token`, `voice.state`, `voice.leave`, `voice.chat`, `voice.moderate` and `voice.unmute` are closed payload contracts. No client position/user/room is accepted. Join, token and chat recheck the authoritative table position and game occupancy, including spectators of an already-running match. Player-only game controls retain their separate occupancy semantics. `voice.moderate` accepts only `mute` or `kick`, requires the current world owner/moderator in the same table, and writes metadata-only moderation audit rows. Server mute never issues a server unmute; the participant must press the mic control.

Chat is authorized against the actual table admission and read via `voice.state`, not broadcast to world subscribers. Text is plain text, maximum 500 code points. A per-account bucket allows five initial messages and replenishes one per second. The state response contains the latest 200 messages and the current admission's `nextMessageSeq`.

`clientMessageId` is `<currentAdmissionId>:<sequence>`, where sequence is a canonical positive JavaScript-safe integer (no zero, leading zero, sign, exponent or value above 9007199254740991). Each admission retains a high-water sequence and its latest 32 acknowledgements. Only the exact next sequence creates a message; a recent retry returns the original acknowledgement even if its supplied text changed. A processed sequence whose acknowledgement aged out returns `voice_retry_expired` without posting again. Gaps and foreign admission prefixes are rejected. There is no 4,096-message lifetime quota.

The browser synchronizes its sequence monotonically within one admission. It retains one pending submission's original text and ID for explicit retry, while edits to the next draft remain separate. A delayed response cannot clear an edited draft or affect a replacement socket/admission. An expired retry is terminal: explain that the old submission was already processed, resynchronize state, then allow a new user submission. Never automatically repost the old text with a new ID.

Acknowledgement payloads are cleared on admission revocation. Visible table messages are memory-only and deleted five minutes after the last seat leaves; a new join after that deadline also clears them before exposing state. Process restart drops all chat. The durable journal remains identity/room only.

## Revocation and failure handling

Lock order is hub admission lease → hub state mutex → voice state mutex. Journal SQL and provider network calls run outside hub and voice state mutexes. A dedicated worker serialization mutex orders reconciliations only; it is never acquired by join, movement, lifecycle invalidation, token minting or chat.

1. Reserve an opaque identity under the authoritative admission fence.
2. Persist identity/room (no token) with a three-second database deadline.
3. Recheck the reservation, session/epoch, membership, position and game state before activation.
4. Mint a locally signed 60-second JWT only for that active admission. Grants are room-specific: subscribe and microphone publication only; no camera, screen, data, metadata editing or room administration.
5. Logout, ban/kick, disconnect, epoch rotation, world exit, game entry and authoritative table exit invalidate the local admission immediately. Provider removal is asynchronous; the pending removal blocks all new voice admissions/tokens until cleanup succeeds. Existing game/world chat stays independent.
6. The worker invokes official SDK `RemoveParticipant` with explicit `RevokeTokenTs = time.Now().Add(2*time.Second).Unix()`. The generated protocol field documents Unix seconds and rejection of tokens whose `nbf` is **before** the cutoff. The two-second margin covers whole-second equality and small clock skew; it is not a claim that arbitrary provider clock skew or in-flight refresh races have been proven safe.
7. On failure, retain the journal row and retry on the next worker cycle. Delete it only after a successful explicit provider removal. A commit-unknown insert never results in a token.
8. At process startup every journal row is treated as revoked before readiness, including minted identities that never connected. Sweep only the configured application room prefix; retain unknown identities as pending before attempting their journal write, remove them explicitly, and delete empty managed rooms. A failed or commit-unknown write remains pending even if the participant disappears from the next listing; disappearance never proves token revocation. Room cleanup reserves a cleanup barrier so it cannot race a new local seat reservation. Shutdown attempts bounded cleanup; failed rows remain for restart.

The worker runs once a second with bounded provider calls. Membership/position validation and media removal are not instantaneous network guarantees: local invalidation is immediate, provider removal starts on the worker cycle and may take up to the request deadline. An active remote participant can remain connected during a provider outage; **degraded means isolation is not yet confirmed**, and new voice access stays blocked. Do not clear journal rows manually to hide this state.

Signed webhook verification uses the official verifier, validates the body digest, and caps request size at 64 KiB. Webhooks do not create or resurrect admissions. They are acknowledged as hints; the periodic authoritative sweep handles delivery loss, duplication and reordering. No audio content is recorded or audited.

## Browser behavior

Joining connects in listen-only mode. Microphone capture is requested only after the mic button or explicit PTT gesture. Permission denial keeps table chat and listening available. A network connection failure keeps a confirmed authorized chat seat, with audio disconnected; leave/rejoin is explicit. Closing the panel, hiding the tab, losing the world connection or entering a game disconnects media and requests seat release. Switching tables waits for the prior leave acknowledgement. Delayed join/token/connect completions cannot resurrect a cancelled session. A join acknowledgement timeout is treated as a possibly allocated seat and cleaned up.

Controls include mic toggle, device selector after permission, volume, deafen (also turns off mic), optional keyboard/touch PTT, active speaker indicators, moderator mute/kick and leave. Keyboard PTT ignores typing and interactive controls. No native application, video, screen sharing, recording or transcription is included. Production microphone access requires HTTPS/secure context.

## Required staging acceptance — NOT EXECUTED

Use a dedicated, authorized LiveKit Cloud project and HTTPS deployment. No fake/local provider can satisfy these checks. Real authenticated browser acceptance is also deferred until the user authorizes the account/invite lifecycle workflow.

- Two different tables in one world, then a second private world: verify audio and chat cannot cross either boundary. Use distinguishable spoken phrases and independent real devices/accounts.
- Eight seats succeed; the ninth, guest, outsider, banned account, wrong epoch and out-of-zone caller fail. Verify one seat per account.
- Connect with initial mic muted and no prompt before the explicit mic control. Verify denied permission, no input device, device change, blocked microphone, blocked WebRTC/network and playback restrictions. Check desktop and real mobile browsers, touch PTT, keyboard typing, deafen, volume and speaker indication.
- Exercise walk-out, lobby/game entry, world exit, logout, session expiry, ban/kick, websocket disconnect/resume and tab hiding. Measure actual remote audio cessation; confirm game/world chat continue during voice failures.
- Privately capture the initial JWT and a provider-refreshed JWT using the staging test harness. Never print or store them in evidence. Kick the admission and confirm the explicit cutoff request succeeded; replay **both** tokens in fresh clients immediately, near the default one-minute window, and around the ten-minute revocation/refresh boundary. Neither may restore membership or audio. Exercise a refresh concurrent with removal and controlled clock skew. These are mandatory acceptance tests because local token claims and HTTP request capture do not prove Cloud enforcement.
- Force removal/listing failure: verify degraded status, no new admission/token, retained journal, retry success and readiness recovery. Restart after mint but before connect and prove the old token cannot join. Restart after an ambiguous journal write and verify no unjournaled mint occurred.
- Duplicate and reorder signed webhooks; invalid signature/body tampering must fail. Confirm old events cannot affect a later opaque admission. Verify automatic host transfer and five-minute chat deletion.
- Record provider version/project, UTC timestamps, exact executed cases, observed outcomes and redacted request metadata. Keep failed evidence. Do not mark CP7 voice acceptance complete until these checks pass.

Official references checked during implementation: [LiveKit token lifecycle/revocation](https://docs.livekit.io/frontends/reference/tokens-grants/), [official Go SDK](https://github.com/livekit/server-sdk-go/blob/main/README.md). Cloud documentation notes automatic refresh, ten-minute refresh-token lifetime/revocation retention and the default cutoff buffer. An explicit cutoff and short initial TTL are necessary; they do not substitute for the real replay tests above.
