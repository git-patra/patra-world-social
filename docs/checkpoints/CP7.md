# CP7 — Warkop, guest persistence dan account settings

Status: **IN_PROGRESS — bukan PASS**. Kode T13 disetujui; T14 round2 diimplementasikan dan diuji, tetapi scoped re-review belum dilakukan. Pekerjaan dijeda atas permintaan pengguna.

- Produk adalah web responsif, termasuk browser mobile.
- LiveKit Cloud staging belum tersedia. SDK/token/admission/revocation tests dapat dikerjakan, tetapi actualaudioisolasi dan replaytokenawal/refreshed belum diuji.
- Authenticated browser credentials/account workflows ditunda pengguna.
- Voice harus mulai mute, izin mic hanya setelah tombol pengguna, scope satu meja, dan gagal secara jelas tanpa memutus game/chat.
- Dua meja dengan8kursi, admission/revocation, ephemeraltablechat dan moderatorcontrol mengikuti spec.
- T14 owns guest reload persistence dan fullaccountsettings/deletion orchestration, termasuk acceptedT12 leaderboard removal seam.

Tidak ada checkpointPASS yang disimpulkan dari fakeprovider, tokenclaimtest, atau viewportcheck. Bukti akan ditambahkan setelah implementation/review/verification dilakukan.

## Rancangan verifikasi T13 — belum merupakan hasil tes

| Area | Bukti kode/integration yang direncanakan | Bukti eksternal yang tetap diperlukan |
|---|---|---|
| Admission |8kursi, satuuser, proximityauthoritative, epoch/membership/game-entry fences | Browser multiakun dan perpindahan meja |
| Token | OfficialSDKclaims60s/mic-only/room/opaqueidentity; actualSDKwire explicitcutoff | Replaytokenawal danrefreshed setelahkick padaCloudstaging |
| Revocation | Localinvalidate segera, durablejournal, retry/degradedgate, restartneverconnected | Provider removes activeaudio/rejoin |
| Tablechat | Admission-only ephemeral, dedup/rate,5minemptycleanup, no world/tableleak | Browser dua meja/dua dunia |
| UI | Muteawal, useractionpermission, denied/no-device/networkstates, staleattachmentcleanup | Desktop/mobilebrowser audio dan devicepermissions |
| Deletion | T14 composes session/mediarevocation; pendingcleanupjournal survivesidentityremoval | User-authorized accountworkflow |

Independent shared table-zone contract references existing map identity/checksum; navigation and recorded game result versions are preserved. New voice admission remains unavailable when provider is unconfigured or cleanup cannot establish isolation. No microphone permission is requested merely on page load.

## Kandidat T13 siap review

[Implementation verification](evidence/CP7/t13-verification.md) records full serial-package Go1.26.8 race/integration,vet/build and71frontendtests/lint/typecheck/buildPASS. OfficialSDK protobuf capture verifies explicitUnixsecondcutoff; actualJWTclaims verifies60s/mic-only/room/opaqueidentity. Actualapplication/PGjournal tests exercise walkout/gameentry/ban/epoch/logout and crash-safe cleanup. These are independentcode/integration checks, not Cloudmediaenforcement.

[Guest preview](evidence/CP7/t13-guest-preview.md) and screenshots show unavailablevoice, guestgate, two8seat tables, disabledjoin and noobservedpermissionprompt onopen. The liveAPI was acceptedT12 duringtheseCUAchecks; T13guestE2E isauthored/NOT_RUN. RealT13readiness andCloudproviderbehavior are not inferred.

Code awaitsindependentreview/commit. NewprojectGo minimum1.26.8 isinstalled onlyunderproject.cache; systemGo1.26.2/globalenv unchanged. NoLiveKitCloudproject wascreated.

## Review T13

[Independent review](evidence/CP7/t13-review.md) requires four fixes beforecodeacceptance: unknownparticipantcleanup afterfailed/uncertainjournalwrite; spectatorgameentryvoicefence; continuedtablechat beyond4096messages withbounded safe dedupe; lateACKdraftpreservation scopedtoattachment/admission. Originalimplementerfixround1 underway. No T13commit/CP7PASS.

## Perbaikan T13 — menunggu scoped review

Empat temuan telah diperbaiki dan diuji: cleanupjournal tetap pending setelah write gagal/uncertain, spectator fence, bounded sequence retry protocol tanpa kuota4096, dan pendingchat/draft generation fence. AffectedGo-race/integration/vetPASS;75frontendtests/lint/buildPASS. Typecheck kini secara eksplisit mencakup VoicePanel/session/microphone dan kedua filetest; hasiltypecheck awal hanya mencakup geometry. Lihat bagian correctionround1 pada verificationreport untuk perintah/output dan batas bukti.

Scoped rereview belum selesai. T13/CP7 belum selesai; Cloudmedia, replaytoken, authenticatedbrowser dan realdevice acceptance tetap pending.

## Hasil scoped review T13

[Scoped rereview](evidence/CP7/t13-rereview1.md): empat temuan ADDRESSED, nol temuan baru. Kode siap commit parsial C13. Seluruh batas bukti eksternal di atas tetap berlaku; CP7 IN_PROGRESS.

T13 partial commit: `b6e74fbf12ff0a2913e09888b81c1e3d767566b6`. ActualAPIreadiness checked afterrestart: healthOK, voiceunavailable(twozones) withoutproviderconfiguration. This is configuration/readiness evidence only.

## T14 candidate — independent review pending

[Implementation verification](evidence/CP7/t14-verification.md) records full serial Go race/integration, vet/build, 83 frontend tests, lint/typecheck/build. Storage tests execute the actual GuestHistory transaction code with a maintained Node IndexedDB implementation. Actual 100-record browser boundary remains NOT_RUN.

[Guest browser verification](evidence/CP7/t14-guest-preview.md) covers real completion of all three local games, history/settings reload, duplicate protection, unfinished reload and explicitly approved reset. Account credential/login/world/logout/deletion browser acceptance remains deferred. No task completion or CP7 PASS is inferred.

## T14 review

[Independent review](evidence/CP7/t14-review.md) requires four fixes before code acceptance: concurrent REST writes after account deletion, tombstone username collision, account/device preference scope, and stale Settings/response-cookie races. Fix round1 is underway. The minor countdown timestamp offset is carried to T16. CP7 remains IN_PROGRESS.

T14 fix round1 is ready for scoped re-review. The report records canonical ordered account fences, reserved tombstone migration, account/device preference isolation, generation-fenced Settings callbacks, independent response-cookie repair, and field-only Motion writes. Final affected Go checks and 94 frontend tests passed. Guest Motion/reload smoke preserved unrelated settings and restored defaults afterward. External acceptance remains pending; no commit/PASS claim yet.

[Scoped re-review round1](evidence/CP7/t14-rereview1.md) marks the original four findings addressed, but identifies two new Important Motion regressions. Round2 will separate profile/preference notifications from actual session changes and prevent obsolete requests from dispatching across a delayed CSRF lookup. No T14 acceptance/commit yet.

## Pause requested by user

T14 round2 repairs Motion session teardown and obsolete post-CSRF dispatch. Coordinator reran 97 frontend tests, typecheck, JS syntax lint and build successfully. Independent re-review is still pending; [round2-only diff](../handoff/t14-round2-review.diff) is preserved for the next conversation. Current work is committed as a WIP snapshot at user request. No CP7 PASS or T14 acceptance is claimed. [Handoff](../HANDOFF.md).
