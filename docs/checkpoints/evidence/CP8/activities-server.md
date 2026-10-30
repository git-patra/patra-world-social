# Activities server verification — 30 October 2026

Runtime: macOS26.5.2 arm64 / pinned Go1.26.8; AppleM1 from prior baseline (not reverified via denied sysctl). Uncommitted Task11 build based on c9fdec6. Actual authenticated test WebSockets and isolated PostgreSQL schemas; no injected browser authentication. All results below are application JSON bytes, excluding WS/TCP/TLS overhead.

The20-client A allocation is5race/2penalty/4Blaster/2Pingpong/1claw/4fishing/2spatial spectators. B replaces race/penalty with4free riders and3football users. A60 repeats A over3private worlds. Trusted Hub placement is a setup shortcut; WS commands govern admission, lobbies, starts, movement and casual intents. B sustains rider controls and all3football users kick during setup; fishing recasts/claw reacquisition are exercised. Pingpong paddles move with human serve pending; penalty times out under its real rules. These short windows do not prove continuous rallies or every-game functional acceptance.

Each steady window follows actual clearing/countdown and a separate2s quiet baseline after500ms admission drain. Initial11-snapshot/full-resync/metadata bursts lie outside that window. Activity totals include reliable lifecycle, lobby, match and game events; peak complete1s bins are reported without dropping reliable bursts.

| Workload | Duration(s) | World owner p95(ms)/samples | Game owner p95(ms)/samples | Max participant activity KiB/s | Max spectator activity KiB/s | Max spectator1s activity KiB | Max total KiB/s |
|---|---:|---:|---:|---:|---:|---:|---:|
|A20|8.110|0.010625/162|0.061291/113|33.094|7.576|11.635|61.724|
|B20|8.113|0.013042/162|0.032292/115|23.199|11.747|13.942|52.581|
|A60|8.079|0.008375/484|0.035958/142|34.639|7.892|10.650|63.447|

A20 representative rates: race=15.044Hz, Blaster participant=9.988Hz, Pingpong participant=9.988Hz; spatial Blaster/Pingpong spectators=3.329/3.329Hz. Endpoint/drain timing may add a small number of events/window; expected production cadence is15/10/10Hz, watchers capped4Hz. B race rate0 is intentional.
B20 representative rates: race=0.000Hz, Blaster participant=9.984Hz, Pingpong participant=9.984Hz; spatial Blaster/Pingpong spectators=3.328/3.328Hz. Endpoint/drain timing may add a small number of events/window; expected production cadence is15/10/10Hz, watchers capped4Hz. B race rate0 is intentional.
A60 representative rates: race=15.224Hz, Blaster participant=10.150Hz, Pingpong participant=10.026Hz; spatial Blaster/Pingpong spectators=3.342/3.342Hz. Endpoint/drain timing may add a small number of events/window; expected production cadence is15/10/10Hz, watchers capped4Hz. B race rate0 is intentional.

Actual world tick cost is measured only when a fixed step is due; game owner advance is sampled at most once per50ms window. Both exclude queue wait and use bounded nonblocking diagnostic channels, nil outside tests. Snapshot-owner queue p95 is separately available in JSON. The60-client owner p95≤25ms and average participant≤40/spectator≤12KiB/s assertions pass for this workload. Complete-second peaks exceed the nominal per-second budgets: B20spectator13.942KiB against12 and A60participant46.582KiB against40. No strict peak capPASS is claimed. B spectator maximum complete1s bin13.942KiB exceeds12 during reliable casual/lifecycle plus visual bursts; peak12KiB/s is NOT established. The capped single-Blaster probe is not a universal bound including arbitrary simultaneous casual/resync bursts. Inputs normally≈8Hz, within20Hz limit; this is not a max-rate sender stress test.

The original A20 baseline was213.783KiB/s total to an observer and215.541KiB/s to a race participant; it broadcast55casual snapshots/s, all race frames and≈20lobby metadata frames/s. Early baseline activity counter omitted lobby/lifecycle bytes; compare total bytes directly. Transport now separates visual wakeups from metadata changes, caches unchanged casual state, filters spatial/explicit spectators, and fairly limits replaceable visual bandwidth. New-match/interest snapshots and reliable terminal/lifecycle events remain available.

Independent capped Blaster codec probe:4human UUID actors with32HTML-escaped-name characters,8bots,48shots. Full floats/simulation fields cost7771B+256B envelope allowance=78.389KiB/s@10Hz. Final visual projection3365Bpayload,3490Bactual event=34.082KiB/s; conservative envelope allowance gives35.361KiB/s.0.001 rounding has≤0.0005normalized error. Private simulation fields stay internal; projectile tuples preserve id/x/y/owner. Full physics/shared goldens pass unchanged. Actual Go-event→WorldSocket→ActivityStore→61-entity scene and command/resync paths are tested.

Portable raw data: [before](activities-server-before.json), [final](activities-server-final.json). Browser frame/heap/audio/lifecycle, bundle figures and deferred gates are in [combined verification](activities-verification.md). OnlyCP1officialPASS; these≈8s load windows and the separate15min browser profile are not the required60minT16soak.

Pinned commands (quietly source ignored `.cache/database.env` first):

```sh
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -count=1 -race -tags=integration -p 1 ./...
ACTIVITIES_LOAD=1 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build .cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go -C apps/server test -tags=integration ./internal/application -run "^TestActivitiesCombinedLoad" -count=1 -v -timeout=180s
```

Verification status: final workload3/3 PASS (62.48s suite); focused rules/cache/interest/fairness/generation/personal-finish/codec tests PASS. Final frozen-source pinned Go full race/integration PASS (22packages with tests; application86.594s, realtime10.606s); vet/build PASS with empty output and verification process exit0. Root final frontend4JS+223TS/typecheck/lint/build PASS.
