# T1 implementation report

## Scope and provenance

Implemented baseline migration only. No commits made. Coordinator owns progress/plan checkboxes, source clone preparation, screenshots and final commit. Source repository was read only: HEAD `8d969212d91804e06225f5b1a75eed49e8c25307`; status contained preexisting untracked `.idea/`. New repository has no remote.

Every inherited file under `src`, `public`, and `tests` was compared byte-for-byte against Git HEAD after migration to `apps/web`; all match. Procedural art, movement and game rules are unchanged. Existing building demo stories remain during T1; local solo status and social-world title/metadata replace main portfolio branding without implying working multiplayer.

## Files and behavior

- Root npm workspace, lockfile, `.node-version`, Makefile: Node 25.2.1 / npm 11.6.2, frontend scripts and Go checks/build. Go cache and binary use ignored `.cache/`.
- `apps/web`: migrated assets/source/tests, Vite TypeScript config, theme entrypoint, package and incremental typecheck config. Preserved resume generation script under `apps/web/scripts`.
- `apps/server`: Go 1.26.2 module; configurable `SERVER_ADDR` (default loopback 8080), validated port, GET health route, bounded read-header timeout; config and HTTP handler tests.
- `.env.example`: documented local placeholder/default only, no secrets.
- `.github/workflows/ci.yml`: pinned runtime versions from files, clean npm install and complete `make check`.
- README documents local startup, provenance, limitations and no deployment.
- Removed obsolete executable `scripts/deploy.sh`, deployment Caddy config and superseded Node build/server scripts. Historical deployment documentation moved to `docs/history/DEPLOYMENT.md` and explicitly labeled historical. No active deploy entrypoint/destination.

## Verification

- `make check`: PASS. Web syntax lint, Go vet, TypeScript check, all 7 inherited web unit tests, Go tests (health response/405/404 and eight configuration cases), Vite production build and Go binary build.
- Vite build: 129 modules; HTML 23.56 kB, CSS 68.20 kB, JS 302.80 kB before gzip. No build warnings.
- `make test` repeated after mutation checks: PASS (7 web tests, both Go packages).
- Test mutation verification: temporarily changed health route to `/health/broken`; `go test ./cmd/server` correctly failed (GET expected 200 got 404; POST expected 405 got 404). Restored file.
- Temporarily widened max port to 99999; `go test ./internal/config` correctly failed for `localhost:65536`. Restored file. These are mutation red/green evidence, not a claim that tests preceded implementation.
- `git diff --check`: PASS.
- Live Go curl (outside sandbox): HTTP 200, `Content-Type: application/json`, body `{"status":"ok"}`.
- `npm install --ignore-scripts` completed with 0 vulnerabilities. Clean `npm ci --ignore-scripts && make check` final result appended below.

## Sandbox and remaining work

Initial network install stalled inside sandbox; interrupted and repeated with escalation. Initial Vite launch failed with EPERM on loopback; escalated launch succeeded. Sandboxed curl cannot reach the escalated process (exit 7); escalated curl succeeded. No failure was hidden with stub checks.

Running local servers for coordinator captures: Vite http://127.0.0.1:4176 (tool session 41504), Go http://127.0.0.1:8080 (session 27304). Coordinator will capture actual desktop/mobile screenshots and compare source references. Browser visual verification remains coordinator-owned and must happen before T1 completion.

Typechecking intentionally covers Vite plus `world-math.js` and `penalty.js`; remaining legacy UI JS is not fully typechecked. No integration/E2E scripts were added because those suites do not exist yet; README states this explicitly. No deployment performed; multiplayer/social shell and server-authoritative rules remain future tasks.

Final clean verification: `npm ci --ignore-scripts && make check` exited 0. npm installed 121 packages with 0 vulnerabilities; all checks passed. Final production CSS 68.24 kB (15.93 kB gzip), JS 302.80 kB (98.84 kB gzip), HTML 23.56 kB (7.07 kB gzip).

Coordinator visual verification completed: actual browser desktop1440×900 and mobile390×844; screenshots saved next to this file; console warn/error list empty.
