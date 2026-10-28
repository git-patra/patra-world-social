# T6 guest-only UI smoke

27 October 2026. CUA inspected local Vite4176 at1280×720 after T6 UI hot reload. Main API still runs accepted T5; authenticated T6 flow was not attempted.

- Teman button opens Teman dan undangan with login/register gate and no friend/invite data. Closing returns focus to the trigger.
- Chat dunia opens local panel with “Masuk dunia privat untuk chat.” and disabled Kirim. No message sent.
- Entered non-sensitive local test draft “wasd” and pressed Right in textarea. Player remained at32:32; input focus did not move the guest avatar. Cleared the draft and closed the panel afterward.
- Console warning/error list empty. Saved t6-guest-friends-gate.png and t6-guest-chat-gate.png. Preview left on game village.

This verifies only guest gates and local input focus. Registration/login, accepted friends, invitation redemption, authenticated chat/world isolation/reconnect/moderation remain AUTH_BROWSER_NOT_RUN by user deferral. Screenshots and Node/backend tests do not replace those acceptance items.
