# Reproduce activities verification locally

Use the dedicated `patra-world-social` checkout, pinned toolchain and dedicated PostgreSQL runtime described in [Activities resume](../handoff/ACTIVITIES-RESUME.md). No public deployment is part of this workflow. Preserve existing history and the separate source repository.

## Automated checks

From the repository root:

```sh
npm test
npm run typecheck
npm run lint
npm run build

set -a
. .cache/database.env
set +a
export GOTOOLCHAIN=local
export GOCACHE="$PWD/.cache/go-build"
PATRA_GO="$PWD/.cache/toolchains/golang.org/toolchain@v0.0.1-go1.26.8.darwin-arm64/bin/go"
"$PATRA_GO" -C apps/server test -count=1 -race -tags=integration -p 1 ./...
"$PATRA_GO" -C apps/server vet ./...
"$PATRA_GO" -C apps/server build ./...
ACTIVITIES_LOAD=1 "$PATRA_GO" -C apps/server test -tags=integration ./internal/application -run '^TestActivitiesCombinedLoad' -count=1 -v -timeout=180s -cpuprofile="$PWD/.cache/activities-load.pprof" -o "$PWD/.cache/activities-load.test"
```

The opt-in load test uses isolated schemas and actual local WS/HTTP transports. It measures A20, B20 and A60 across three private worlds, with approximately eight-second steady windows after setup/countdown. It is not the older T16 60-minute soak. Trusted Hub placement is a disclosed setup shortcut; admissions and game/casual commands use the actual transport. JSON byte totals exclude WS/TCP/TLS framing. Keep average, complete-second peak and startup/resync bursts distinct when reading results.

## Browser profile

Build the combined fixture separately so HMR and application rebuilds cannot reset a long run:

```sh
node --input-type=module <<'JS'
import {build} from 'vite';
import path from 'node:path';
await build({
  root:path.resolve('apps/web'),
  configFile:path.resolve('apps/web/vite.config.ts'),
  build:{
    outDir:path.resolve('.cache/activities-profile-dist'),
    emptyOutDir:true,
    rolldownOptions:{input:path.resolve('apps/web/tests/activities-combined.html')}
  }
});
JS
python3 -m http.server 4182 --bind 127.0.0.1 --directory .cache/activities-profile-dist
```

Open `http://127.0.0.1:4182/tests/activities-combined.html`. Prime any browser automation locator helper **before** clicking Start, so its listeners are outside the instrumentation window. Keep the tab visible. Use the explicit smoke or15-minute/20-cycle button; save the displayed JSON only when `done:true` and no `failed` field is present. A separate build directory preserves the exact measured artifact when later fixes are tested.

The fixture exercises production world rendering, ActivityStore subscriptions/timer and AudioService disposal with synthetic public snapshots. It does not authenticate browsers or create multiplayer/history records. Record device, browser, DPR, canvas size, build hash, duration, sample count and concurrent host workload. CPU callback measurements exclude browser compositing. Raw heap without forced GC does not establish retained-heap acceptance.

## Evidence and release boundaries

Use [combined verification](../checkpoints/evidence/CP8/activities-verification.md), [server measurements](../checkpoints/evidence/CP8/activities-server.md), and the associated raw JSON files. The authored browser E2E source is not proof that it ran. Guest CUA observations and actual backend transports are separate evidence types.

Only CP1 is an official PASS. Authenticated-browser acceptance, LiveKit/provider/media, physical mobile, post-GC retained heap, sustained physical60fps and older T16 soak/backup/restore/readiness remain separately gated. Preserve any measured bandwidth peak gaps in reports; do not silently raise budgets or drop reliable lifecycle/result events to hide them.
