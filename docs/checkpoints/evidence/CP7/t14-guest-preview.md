# T14 guest browser verification

27 October 2026. Actual in-app browser, tab 2, http://127.0.0.1:4176/. Frontend used the T14 working tree through Vite; API remained the accepted T13 binary. Interactions used visible controls and real game clocks. No injected storage/session, credential entry, provider connection, or backend restart.

## Executed checks

| Check | Observed result |
|---|---|
| Initial history | Empty before the test |
| Guest settings | Saved synthetic name **Guest QA T14** and reduced motion; actual reload retained both and showed Motion off |
| Penalti | Played against BOT; finished **5–4**, 14 kicks including sudden death, 63 seconds; one local result at 19:03:38 |
| Duck Arcade | Completed 60-second session with one real Space input: score 0, hits 0/1; one local result at 19:08:09 |
| Duplicate protection | Reopened the finished Duck panel, then history; still exactly one Duck entry |
| Balapan | Guest plus four BOT; real Up input held for two seconds. BOT completed five laps in 92.30/93.97/95.87/97.87 seconds. Guest DNF at the normal 137.30-second cutoff; one local result at 19:11:08 |
| Three-game persistence | Actual reload retained exactly three completed entries with unchanged timestamps |
| Game-facing detail | Penalti outcome/score/kick count, Duck hits/shots/accuracy, and race standings/laps/DNF displayed correctly |
| Interrupted session | Started another Duck session and deliberately reloaded before completion; history still contained only the three finished sessions |
| Browser console | Warning/error query returned an empty array |
| Reset | User explicitly approved deletion of these three synthetic results and the Guest QA T14 preferences. Rechecked that scope, clicked **Konfirmasi hapus data perangkat**, and saw **Data perangkat dihapus** |
| Reset persistence | Actual reload showed empty history; settings returned to Guest, Labu, audio on, keyboard and touch, reduced motion off |

## Development interruption and corrections

An initial Duck run registered one pointer shot, then a concurrent frontend edit caused a Vite full reload with 28 seconds remaining. Debug logs at 12:06:01 UTC showed HistoryPage Fast Refresh invalidation followed by reconnect. This run was discarded and was not counted as a completed game or production failure. Subsequent game checks ran while frontend edits were frozen.

A targeted accessibility Space attempt timed out without sending input during that interruption. The successful frozen run used actual Space input at page focus. Initial name typing appended to the default; selected all and corrected it before the settings verification.

The first local-history detail showed technical version metadata only. The implementer replaced it with game-facing details and Indonesian game labels; the updated output was checked above. Version metadata remains stored in the record.

## Evidence

- `t14-guest-settings-reload.jpg`: name and reduced motion after reload.
- `t14-guest-penalty-history-reload.jpg`: first persisted result; before detail-copy refinement.
- `t14-guest-three-games-reload.jpg`: all three completed games after reload.
- `t14-guest-reset-ready.jpg`: final reset step prepared before action-time approval.
- `t14-guest-reset-confirmed.jpg`: empty history after the approved reset and reload.

## Limits

These checks cover three real browser results and reset, not the 100-record browser boundary. Backend account deletion/password/recovery routes were not running in this preview and no authenticated browser flow was attempted. No physical-mobile performance or LiveKit acceptance is inferred. CP7 remains IN_PROGRESS; no task completion claim.

Review note: M1 identifies a three-second early finish timestamp for race/Duck because countdown was omitted from the wall-time conversion. This evidence establishes the observed values survived reload, not exact finish-time accuracy. The repair is assigned to T16.

## Fix round 1 guest Motion regression
After the implementation was frozen, used the main Motion button while guest. Settings showed only reduced motion changed to on; name Guest, avatar Labu, audio on and keyboard/touch selection remained unchanged. Actual reload retained that state. Toggled Motion back to its default on/normal animation and verified reduced motion off plus unchanged other preferences. Console warnings/errors remained empty. No new history or deletion was performed. Screenshot: `t14-fix1-guest-motion-reload.jpg`. This is the device scope only; account scope is covered by component/integration tests, not an authenticated browser run.
