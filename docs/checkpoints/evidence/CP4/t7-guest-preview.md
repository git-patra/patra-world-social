# T7 guest-only UI smoke

27 October 2026, local preview http://127.0.0.1:4176/, CUA IAB, 1280×720.

Clicked Race Park in the rendered destination list. The guest walked from the plaza to the destination and opened its dialog. It shows the game description, “Lobby dunia privat”, and “Sambungkan ke dunia privat untuk ikut lobby.” No join/start button is offered to the disconnected guest. Captured console warnings/errors: none.

Evidence: [guest lobby screenshot](t7-guest-lobby.png).

The Go preview was still the accepted T6 build; Vite rendered T7 frontend changes. This checks only the disconnected guest UI, not online lobby integration. No credentials or sessions were entered/injected. Authenticated browser acceptance remains NOT_RUN per user deferral.

Minor deferred to T15: disconnected guest also sees “Memuat ketersediaan…” indefinitely. Replace it with a truthful disconnected/locked state, and verify the final responsive dialog. This is not evidence of a server load or live lobby fetch.
