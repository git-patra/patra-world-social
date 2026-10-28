# T8 guest-only history UI smoke

27 October 2026, http://127.0.0.1:4176/, CUA IAB1280×720.

Clicked rendered History button in guest mode. A focused “History pribadi” dialog opens with “Masuk untuk melihat history. Permainan tamu tidak menghasilkan skor resmi.” No account result data or credential input was shown or submitted. Console warning/error capture was empty.

[Evidence screenshot](t8-guest-history.png).

Backend preview still uses accepted T7; Vite renders current T8 frontend. This verifies only the disconnected guest gate, not history API behavior or authenticated browser acceptance. Real backend integration evidence is separate. Authenticated browser tests remain NOT_RUN by user direction.
