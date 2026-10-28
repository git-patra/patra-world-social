# Game-only follow-up review

What this change does: Commit `b448cf4f2979ada581e12c6118a31af1d40d95e4` removes inherited portfolio documents, the source deployment guide, and 14 legacy screenshots. It updates README startup/check commands and records the cleanup without changing runtime code.

**Verdict: Ship the scoped documentation cleanup.** No new actionable findings in this commit. The product copy, metadata, location dialogs, and public assets inspected at this committed tree focus on the world/game. There are no runtime CV, LinkedIn, resume download, career timeline, or business showcase links/content left in the inspected source. The autumn/Halloween procedural art remains.

## Known runtime follow-up (existing, nonblocking)

- `apps/web/src/world.js:247` still draws `CONNECT` on Balai Desa, and `:308–309` draws its `P` coin emblem. These are visible inherited financial-facade details, rather than internal identifiers. They were already recorded as Minor in `docs/checkpoints/evidence/CP1/t2-review.md:21`, explicitly deferred to T15 in `docs/PROGRESS.md:96`, and disclosed in `docs/ACCEPTANCE.md:7`. Replace the lettering/emblem with village details during that tracked work while retaining shape/palette. This commit does not introduce or hide them; do not describe the final facade cleanup as complete yet. Priority P3, existing accepted follow-up.

## Provenance and internal names

- README:62 describes the clone source and removal history; checkpoint/spec/progress records describe earlier CV cleanup. These are provenance, not career marketing surfaces.
- `docs/checkpoints/evidence/CP1/t1-desktop.png` visibly records the earlier career/business UI. It remains historical T1 evidence, not a current product/public asset. Retaining it and related checkpoint records is consistent with the explicit retained-history requirement; no history rewrite is warranted.
- Map IDs such as `fintech`, `erp`, `pos`, and `jajaneun`, and UI implementation names such as `project-dialog`, are internal compatibility names. Current visible station names and descriptions are village/game content. Network `Resume`/`session.resume` refers to reconnecting sessions, not CVs.
- Runtime public inventory contains pumpkin SVG and bundled fonts/licenses; the inherited PDF, resume generator, promotional images, and business logos are absent from the committed tree.

## README and isolation checks

Read README against committed Makefile, root/web package scripts, Vite config, Go config/startup and integration harness. `npm ci`, `make dev`, `make server`, `make check`, all-package integration command, and `make load-smoke` match those definitions. Vite binds 127.0.0.1:4176 and proxies HTTP/WebSocket `/api` to Go8080; Go consumes exported environment variables. `make test-integration` is correctly described as auth-only. Load smoke uses real application/DB and direct WebSocket clients with isolated schema cleanup.

The committed CI workflow only installs/builds/checks; no deployment job or source-hosting destination is present. `git remote -v` returned no remotes. No source repository files were changed by this review.

Not checked: no application suites, browser/account actions, deployment, or physical-device acceptance were run. Existing passing suites were not repeated. T6 uncommitted changes were excluded by reading `b448cf4` blobs; this report is the only file written.


Publication note (28 October 2026): the obsolete `t1-desktop.png` and `t1-mobile.png` portfolio screenshots mentioned above are retained only in local inherited Git history. They are omitted from the game-only public snapshot. Later game-only CP1 screenshots remain available; historical test claims are unchanged.
