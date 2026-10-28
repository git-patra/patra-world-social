# Activity area authority and world scenes — integration evidence

Base bb1869a. Implemented on feat/world-activities. Details: activities-task-3.md, activities-task-4.md and activities-world-scenes-browser.md in this directory.

- Task3 Go1.26.8 affected world/lobby/realtime/application integration + race detector: PASS. Exact package durations in task report.
- Task4 final review fixes: 42 focused production activity/shell tests PASS, typecheck PASS; explicit RED→GREEN probes for pointer ownership, same-tick lifecycle, delayed resync, stale penalty turn/reveal.
- Root combined frontend verification after fixes: 4 JS +154 bundled tests PASS. Lint, production build, diff whitespace check PASS.
- Entry JS415.20kB /132.44kB gzip, CSS70.56kB /16.55kB gzip. LiveKit398.26kB /102.58kB gzip stays lazy. Build size is measured; physical-device framerate/heap/network acceptance is not established by this build.

Fresh scoped re-review: R1, R2 and R3 ADDRESSED; Tasks3/4 approved. See activities-stage2-review.md. Browser/LiveKit/physical-device gates remain explicit; only CP1 is officially PASS.
