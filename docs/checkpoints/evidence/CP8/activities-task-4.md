# Task 4 — persistent scene/store/input integration

Implemented in the shared `feat/world-activities` checkout. No commit made; root coordinates Task 3/4 review and integration.

## Behavior and interfaces

- `activities/store.ts`: application-owned `ActivityStore`, singleton `activities`, and `mountActivities(world)`. The application owns one transport attachment/event/status listener set and a 50ms input/model projection timer. Closing every React menu subscriber leaves game snapshots, local engine/deadlines, and completion persistence alive. Local sessions load lazily and preserve existing `Completion`, seeds, deterministic physics, and durable local history.
- `ActivityState.games` retains race/penalty public snapshots independently by match ID. `active` identifies the authorized participant for HUD/results. `inputMode` separately identifies current spatial ownership: area OPEN/cleared match restores walking while retaining a pending durable result claim/HUD. This matches the revised Task 3 server `MovementBusy` seam.
- `ActivityState.areas` stores only `kind: area` metadata. `activity.snapshot {}` resync replaces generation/incarnation state without lifecycle effects; delayed old lifecycle generations/incarnations cannot reopen an area. Live `activity.lifecycle` is ordered and area closure metadata drives `world.setClosedZones`. Public game snapshots are requested from `lobby.list` match IDs and received continuously without any implicit `lobby.join`, spectator admission, input, or history write.
- `activities/input-router.ts`: `InputRouter` / `activityInput` owns walk, ride, race, penalty, pingpong, claw, blaster, fishing, modal modes. Keyboard and existing touch direction pads route through one owner. Modal/text/blur/hidden/lost capture/pointer cancel/disconnect release held intent. Neutral heartbeat never cancels an online click-to-walk route; only held walk intent repeats. Race inputs stop for finished/DNF actors and spatially released leases.
- `activities/scene.ts`: lazy race/penalty adapters return depth-sortable `SceneActor` projections. Public participant actors replace walking avatars/labels, share world depth order and viewport culling, and own the display camera. Race snapshots interpolate positions/shortest heading in the existing world RAF. The actor's display location/coordinates update the HUD without writing authoritative walking coordinates. OPEN/cleared leases remove old finished projections while results remain readable.
- `world/scenes/race.ts` and `penalty.ts`: race uses the exact uniform contract transform (scale 1/64, offsets 1488/1152); penalty renders kicker/keeper/ball, consuming revealed turns only. A late choosing snapshot never guesses ball movement or hidden choices.
- `ActivityHUD.tsx` / `activities.css`: persistent compact participant HUD, at most ten subscriber notifications per second; desktop/touch race and four-zone penalty controls, result return/rematch and menu/details. HUD measures the dock and sits 12px above it, has viewport-capped internal scrolling and a collapse control. Local practice has explicit safe exit, does not fabricate unfinished history, and permits another activity afterward.
- `world/controller.ts`: `setActivityState`, `setOnlineSampler`, one shared RAF, participant suppression/depth ordering/culling, display-only scene coordinates, local safe return, and hidden-tab RAF cancellation/resume. `online-ui.ts` interpolation now runs from the world frame sampler instead of a second RAF.
- `LobbyPanel`, Race/Duel controllers, app and PenaltyGame mounts: actual application wiring uses persistent store state. Race/penalty menu controllers no longer own game subscriptions, input timers, keyboard listeners, or separate render loops. Menus/details stay centered; legacy Duck remains its lazy popup.
- `guest/lifecycle.ts`: lazily registered session cleanup. Existing synchronous `GuestSession.start/clear` API remains in `guest/session.ts` for legacy/tests. Settings imports the cleanup registry so it does not eagerly pull every game into the initial bundle.

## Files

Created: `apps/web/src/activities/{store,input-router,scene}.ts`, `ActivityHUD.tsx`, `activities.css`; `apps/web/src/world/scenes/{race,penalty}.ts`; `apps/web/src/guest/lifecycle.ts`; `apps/web/tests/activities.test.ts`.

Modified: `apps/web/src/app.js`, `online-ui.ts`, `world/controller.ts`, `world/village-art.ts`, `features/lobby/LobbyPanel.tsx`, `features/race/{Race.tsx,session.ts}`, `features/penalty/{Duel.tsx,session.ts}`, `features/duck/session.ts`, `components/PenaltyGame.jsx`, `features/settings/SettingsPage.tsx`, `guest/session.ts`; narrow shell/mock integration in `tests/{app-shell,map}.test.ts`. Root registered all new production/test files in runner/tsconfig. Task 3 owns `activities/contracts.ts`, socket/validators/server/geometry contract files.

## Verification

The five required tests were written against empty behavior, run RED (all assertions failed), implemented, then run GREEN:

- `closing_menu_does_not_unsubscribe_game`
- `one_input_owner_only`
- `modal_and_text_neutralize_active_mode`
- `passive_spectator_has_no_input_or_history`
- `resync_swaps_generation_without_replaying_effects`

Additional real tests cover local→online→local, two independent public matches, local deadline progression without menu subscribers, late lazy-load attachment fences, passive finished snapshot against actual IndexedDB with zero history, exact race transform/walker replacement, installed keyboard/blur/pointer/hidden/text listener teardown, HUD 10Hz, explicit unfinished local exit/next game, spatial reopening with result retention, own finished kart input cessation, and reveal-only penalty ball animation. Actual app-shell regressions prove the shared hidden RAF stops/resumes, neutral heartbeat preserves online map routes, actor display coordinates do not mutate walking authority, and exit returns to `{45,49}`. Hidden RAF, neutral route, spatial release, finished input and reveal regressions were individually observed RED→GREEN.

Final run: `npm test` PASS (4 JS + 148 bundled tests); `npm run typecheck` PASS; `npm run lint` PASS; `npm run build` PASS. Latest build initial JS 131.54 KiB gzip; Race/Duel view chunks each <1 KiB gzip, adapters 0.56/0.63 KiB, session/model chunks remain lazy. These are build measurements, not a physical-device performance claim. Intermediate protocol-v1 fixture and test-double failures were resolved by their owners/narrow fixture updates; final suite has zero failures.

Root CUA has already checked actual local race Canvas/HUD, working keyboard control, live menu standings, and close/reopen preservation. Root requested and received dock overlap/display location/explicit exit fixes; final browser geometry recheck remains root-owned.

## Deviations and remaining validation

- Passive clients keep public race/penalty snapshots for their current world and cull Canvas actors/labels by viewport, instead of churning subscriptions by spectator radius. Re-entering a view uses current full snapshots and has no spectator admission/voice side effect. No client network subscription API exists for radius scopes in Task 3.
- Existing local fixed-step session timers are retained to preserve completion/deadline behavior, alongside the application projection/input timer. Neither adds a Canvas/RAF loop; there is one world RAF, stopped while hidden.
- Full multiplayer reconnect/two-match/physical touch/performance/20-cycle heap acceptance is root integration work. Unit tests exercise production lifetime/input/scene boundaries and two-match state; they do not claim physical-device or live multi-account acceptance.
- Free vehicles/football/pingpong/claw/fishing/blaster are deliberately not implemented in Task 4. Later stages extend `ActivityGame`, per-kind store handling, lazy scene adapters and `activityInput.register` modes. Public area payloads stay isolated from those future kinds.

## Task 3/4 review fix round 1 — independent pointer ownership

Finding: global pointerup/lostpointercapture released all inputs. Releasing a steering thumb canceled a still-held gas pointer, and an unrelated pointerup canceled held keyboard movement.

Changed only `activities/input-router.ts`, `ActivityHUD.tsx`, `tests/activities.test.ts` (plus this report). The router now keeps keyboard/directional intent, semantic button intent and `pointerId`-owned HUD controls separately, then combines them for the one active input owner. Pointerup/pointercancel/lostcapture remove only their matching pointer. Modal/text/blur/hidden/disconnect/owner change clear all sources and the installed held-key set, so delayed releases cannot resurrect a cleared input. HUD buttons pass pointer IDs through down/up/cancel/lostcapture and release only their own semantic keyboard control on keyup/blur.

Meaningful RED command (after bundling the new actual EventTarget regression):

```sh
node --test --test-name-pattern=installed_pointer_owners /tmp/task4-pointer.test.mjs
```

Exact relevant RED output:

```text
✖ installed_pointer_owners_keep_other_thumb_and_keyboard_holds_until_their_own_release (1.720041ms)
ℹ tests 1
ℹ suites 0
ℹ pass 0
ℹ fail 1
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 66.029875
AssertionError [ERR_ASSERTION]: releasing steering must preserve the gas thumb
0 !== 1
```

GREEN commands (from repository root; test bundling uses the same esbuild options as the registered test runner):

```sh
node --input-type=module -e "import {build} from 'esbuild';await build({entryPoints:['apps/web/tests/activities.test.ts'],outfile:'/tmp/task4-pointer.test.mjs',bundle:true,platform:'node',format:'esm',banner:{js: \"import {createRequire} from 'node:module';const require=createRequire(import.meta.url);\"},define:{'import.meta.env.DEV':'false'}})"
node --test /tmp/task4-pointer.test.mjs
npm run typecheck
```

Exact final GREEN output:

```text
✔ closing_menu_does_not_unsubscribe_game (2.065708ms)
✔ one_input_owner_only (0.613459ms)
✔ modal_and_text_neutralize_active_mode (0.15625ms)
✔ passive_spectator_has_no_input_or_history (0.254625ms)
✔ resync_swaps_generation_without_replaying_effects (0.18ms)
✔ local_online_local_clears_ownership_and_two_public_matches_are_independent (3.095ms)
✔ real_local_match_runs_while_all_menu_subscribers_are_closed (1.486916ms)
✔ late_local_lazy_load_cannot_restore_a_session_after_attachment_reset (0.106875ms)
✔ passive_public_finished_snapshot_does_not_write_device_history (7.876542ms)
✔ public_scene_replaces_walking_actor_and_uses_the_exact_physics_art_transform (0.814375ms)
✔ installed_input_owner_releases_keyboard_on_blur_visibility_and_text (1.057333ms)
✔ HUD_projection_is_limited_to_ten_notifications_per_second (6.903083ms)
✔ explicit_guest_exit_neutralizes_unfinished_match_without_history_and_allows_next_game (3.180666ms)
✔ spatial_reopen_removes_finished_projection_while_result_claim_stays_explicit (0.775709ms)
✔ finished_participant_stops_sending_controls_while_other_karts_continue (0.287583ms)
✔ penalty_scene_never_animates_a_guessed_ball_before_authoritative_reveal (0.177291ms)
✔ installed_pointer_owners_keep_other_thumb_and_keyboard_holds_until_their_own_release (0.462667ms)
ℹ tests 17
ℹ suites 0
ℹ pass 17
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 92.701959

> patra-world-social@0.1.0 typecheck
> npm run typecheck --workspace apps/web

> @patra-world-social/web@0.2.0 typecheck
> tsc --noEmit
```

The new event test covers two actual independent pointer IDs (11 gas, 22 steering), up/cancel/lostcapture affecting only their owner, keyboard W surviving unrelated pointerup and subsequent D key transitions, and complete modal/disconnect/hidden/blur resets without ghost reactivation. Focused suite and typecheck only were rerun as requested; no commit or full-suite/build during the frozen review.

### Round 1 continuation — shell producer and authoritative ordering (R1/R2/R3)

R1 also needed the existing shell producer fix: `app.js` now captures the movement pad's pointer ID, ignores other pointer releases, releases only pad direction on its own up/cancel/lostcapture, and preserves the router's separate keyboard source. The full-input reset remains on context boundaries. `tests/app-shell.test.ts` exercises the actual app handlers/world movement with W plus pad pointer 7, unrelated pointer 99, owned pointer release, and W keyup. The new test fixture sends the actual key event type; after correcting that fixture field, a temporary probe using the frozen pre-fix router and reconstructed old shell producer confirmed the same RED assertion, without changing production files.

R2 now compares lease generation and monotonic phase progression as well as tick, so RESERVED → ACTIVE within one owner tick is accepted and an older phase cannot return on a later publish. Full resync merges with newer accepted area metadata without lifecycle effects. Reloads are fenced by attachment epoch and the latest reload request; per-area accepted revisions prevent an in-flight read from introducing another runtime over later live state. Once a fresh current resync introduces an incarnation, retired incarnations cannot return through stale reads or events. Lobby reads are also fenced against later streamed lobby updates.

R3 now orders penalty public snapshots by turn, then choosing → animation → results within that turn, and merges same-turn chosen flags monotonically. Per-match async game read IDs and the active reload epoch reject obsolete snapshot read completions. Scores/deadlines from an older turn therefore cannot replace a newer turn. These async/ownership methods and the router are formatted into readable statements, addressing the review's related minor note.

Only the additional necessary R1 shell producer and its focused regression extend the initially named files; no unrelated production files changed. The following four store tests were observed RED before the implementation:

```sh
node --test --test-name-pattern='ordered_lifecycle|slow_resync|newer_resync|penalty_public_progress' /tmp/task4-pointer.test.mjs
```

Exact RED output (assertion excerpts follow the summary):

```text
✖ ordered_lifecycle_accepts_reserved_to_active_in_the_same_owner_tick (2.2425ms)
✖ slow_resync_merges_against_newer_live_generation_without_replaying_effects (0.419708ms)
✖ newer_resync_incarnation_fences_older_pending_reads_and_old_runtime_events (0.300834ms)
✖ penalty_public_progress_never_rewinds_turn_reveal_or_locked_chosen_flags (1.632792ms)
ℹ tests 4
ℹ suites 0
ℹ pass 0
ℹ fail 4
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 74.681
```

The assertions reported RESERVED instead of ACTIVE; generation 1 instead of 2; old `runtime` instead of `runtime-new`; and `[false,false]` instead of locked `[true,false]` respectively.

Confirmed shell RED command/output using the frozen producer probe:

```sh
node --test --test-name-pattern=world_pad_release /tmp/task4-shell-confirmed-red.mjs
```

```text
✖ world_pad_release_preserves_held_keyboard_and_ignores_unrelated_pointerup (13.857042ms)
ℹ tests 1
ℹ suites 0
ℹ pass 0
ℹ fail 1
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 88.730333
AssertionError [ERR_ASSERTION]: unrelated pointer does not end either owned input
```

Final focused GREEN commands (bundle from repository root, execute the shell fixture from the web workspace, then typecheck from root):

```sh
node --input-type=module -e "import {build} from 'esbuild';await build({entryPoints:['apps/web/tests/activities.test.ts','apps/web/tests/app-shell.test.ts'],outdir:'/tmp/task4-round1',bundle:true,platform:'node',format:'esm',outExtension:{'.js':'.mjs'},banner:{js: \"import {createRequire} from 'node:module';const require=createRequire(import.meta.url);\"},define:{'import.meta.env.DEV':'false'}})"
cd apps/web
node --test /tmp/task4-round1/activities.test.mjs /tmp/task4-round1/app-shell.test.mjs
cd ../..
npm run typecheck
```

Exact final focused GREEN output:

```text
✔ closing_menu_does_not_unsubscribe_game (2.067625ms)
✔ one_input_owner_only (0.6675ms)
✔ modal_and_text_neutralize_active_mode (0.166916ms)
✔ passive_spectator_has_no_input_or_history (0.295958ms)
✔ resync_swaps_generation_without_replaying_effects (0.265458ms)
✔ local_online_local_clears_ownership_and_two_public_matches_are_independent (4.542209ms)
✔ real_local_match_runs_while_all_menu_subscribers_are_closed (1.265333ms)
✔ late_local_lazy_load_cannot_restore_a_session_after_attachment_reset (0.093959ms)
✔ passive_public_finished_snapshot_does_not_write_device_history (9.284792ms)
✔ public_scene_replaces_walking_actor_and_uses_the_exact_physics_art_transform (0.781666ms)
✔ installed_input_owner_releases_keyboard_on_blur_visibility_and_text (0.498667ms)
✔ HUD_projection_is_limited_to_ten_notifications_per_second (6.623125ms)
✔ explicit_guest_exit_neutralizes_unfinished_match_without_history_and_allows_next_game (3.219958ms)
✔ spatial_reopen_removes_finished_projection_while_result_claim_stays_explicit (0.299ms)
✔ finished_participant_stops_sending_controls_while_other_karts_continue (0.112917ms)
✔ penalty_scene_never_animates_a_guessed_ball_before_authoritative_reveal (0.134125ms)
✔ installed_pointer_owners_keep_other_thumb_and_keyboard_holds_until_their_own_release (0.421125ms)
✔ ordered_lifecycle_accepts_reserved_to_active_in_the_same_owner_tick (0.090416ms)
✔ slow_resync_merges_against_newer_live_generation_without_replaying_effects (0.135458ms)
✔ newer_resync_incarnation_fences_older_pending_reads_and_old_runtime_events (0.105166ms)
✔ penalty_public_progress_never_rewinds_turn_reveal_or_locked_chosen_flags (0.157583ms)
✔ actual Map selection retains its online destination after dialog close cleanup (14.232625ms)
✔ semantic direction activation with "Enter" is bounded after release (11.071333ms)
✔ semantic direction activation with " " is bounded after release (9.512458ms)
✔ approved layout solids match runtime art and top-down projection round-trips (6.443916ms)
✔ minimap_click_never_routes_ground_behind_it (5.181667ms)
✔ dialog_close_cleanup_precedes_destination (5.374917ms)
✔ map_click_respects_letterbox_and_dynamic_closed_zone (4.753792ms)
✔ closeDialogThen waits for close cleanup before scheduling the destination frame (4.778667ms)
✔ full map click uses CSS geometry at DPR 1, zoom 0.65 (7.266375ms)
✔ full map click uses CSS geometry at DPR 1, zoom 1 (5.931416ms)
✔ full map click uses CSS geometry at DPR 1, zoom 1.5 (5.072625ms)
✔ full map click uses CSS geometry at DPR 2, zoom 0.65 (5.480875ms)
✔ full map click uses CSS geometry at DPR 2, zoom 1 (4.754125ms)
✔ full map click uses CSS geometry at DPR 2, zoom 1.5 (4.427125ms)
✔ menu child #settings-button returns focus to visible Menu without stealing from the child (3.907958ms)
✔ menu child #history-button returns focus to visible Menu without stealing from the child (2.783958ms)
✔ menu child #quick-play-duck returns focus to visible Menu without stealing from the child (3.199791ms)
✔ hidden world cancels its only frame and resumes without jumping (4.177208ms)
✔ neutral input heartbeat cannot cancel an online map route (4.268167ms)
✔ activity_location_display_tracks_own_actor_without_changing_walking_authority (7.343167ms)
✔ world_pad_release_preserves_held_keyboard_and_ignores_unrelated_pointerup (4.815792ms)
ℹ tests 42
ℹ suites 0
ℹ pass 42
ℹ fail 0
ℹ cancelled 0
ℹ skipped 0
ℹ todo 0
ℹ duration_ms 216.812666

> patra-world-social@0.1.0 typecheck
> npm run typecheck --workspace apps/web

> @patra-world-social/web@0.2.0 typecheck
> tsc --noEmit
```

No full-suite run, build, commit or subagent was performed during this frozen fix round. The scoped activity + actual shell suites contain 42 passing tests; the final typecheck exited 0.
