import { api, type User, type Friend, type World } from './net/api';
import './social.css';
const escape = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])!);
export function mountSocial() {
    const dialog = document.querySelector<HTMLDialogElement>('#social-dialog')!;
    const body = document.querySelector<HTMLElement>('#social-content')!;
    const badge = document.querySelector('#social-badge')!;
    let user: User | null = null, generation = 0;
    const message = (text: string) => { const el = body.querySelector('[role="status"]'); if (el)
        el.textContent = text; };
    async function poll() { const version = generation; if (!user) {
        badge.textContent = '';
        return;
    } try {
        const [friends, notes] = await Promise.all([api.friends(), api.notifications()]);
        if (version !== generation)
            return;
        const count = friends.filter(f => f.incoming && !f.accepted).length + notes.length;
        badge.textContent = count ? String(count) : '';
    }
    catch {
        if (version === generation)
            badge.textContent = '!';
    } }
    async function render() {
        const version = ++generation;
        if (!user) {
            body.innerHTML = '<h2 id="social-title">Teman dan undangan</h2><p>Masuk untuk berteman dan menerima undangan dunia privat.</p><button id="social-login" type="button">Masuk atau daftar</button>';
            body.querySelector<HTMLButtonElement>('#social-login')!.onclick = () => window.dispatchEvent(new Event('patra-login-request'));
            return;
        }
        body.innerHTML = '<h2 id="social-title">Teman dan undangan</h2><p role="status">Memuat teman dan undangan…</p>';
        try {
            const [friends, notes, blocks] = await Promise.all([api.friends(), api.notifications(), api.blocks()]);
            if (version !== generation || !user)
                return;
            body.innerHTML = `<h2 id="social-title">Teman dan undangan</h2><button id="social-refresh" type="button">Segarkan</button><p>Teman tidak otomatis menjadi anggota dunia. Presence hanya menunjukkan tersedia online; lokasi dunia privat tetap tersembunyi.</p><form id="friend-search"><label>Cari exact username<input name="username" required pattern="[A-Za-z0-9_]{3,24}" maxlength="24"></label><button type="submit">Cari</button></form><div id="friend-result"></div><h3>Undangan dunia (${notes.length})</h3><div id="notifications">${notes.map(n => `<p><span>${escape(n.worldName)} · dari @${escape(n.issuerUsername)}</span> <button type="button" data-accept-invite="${escape(n.id)}">Terima dan masuk</button></p>`).join('') || '<p>Belum ada undangan aktif.</p>'}</div><h3>Teman dan permintaan</h3><ul>${friends.map(friendRow).join('') || '<li>Belum ada teman.</li>'}</ul><h3>Akun diblokir</h3><ul>${blocks.map(f => `<li>@${escape(f.username)} <button type="button" data-unblock="${escape(f.id)}">Buka blokir</button></li>`).join('') || '<li>Tidak ada.</li>'}</ul><div id="friend-invite"></div><p role="status">Siap. Daftar presence diperbarui saat disegarkan.</p>`;
            body.querySelector<HTMLButtonElement>('#social-refresh')!.onclick = render;
            body.querySelector<HTMLFormElement>('#friend-search')!.onsubmit = async (e) => { e.preventDefault(); const form = e.currentTarget as HTMLFormElement; const username = (form.elements.namedItem('username') as HTMLInputElement).value; message('Mencari username…'); try {
                const person = await api.searchFriend(username);
                if (version !== generation)
                    return;
                const result = body.querySelector('#friend-result')!;
                result.innerHTML = `<p>@${escape(person.username)} <button id="send-friend" type="button">Kirim permintaan</button> <button id="block-search" type="button">Blokir</button></p>`;
                body.querySelector<HTMLButtonElement>('#send-friend')!.onclick = () => run(() => api.requestFriend(person.username));
                body.querySelector<HTMLButtonElement>('#block-search')!.onclick = () => run(() => api.friendAction(person.id, 'block'));
                message('Username ditemukan.');
            }
            catch (err) {
                if (version === generation)
                    message(errorText(err));
            } };
            body.querySelectorAll<HTMLButtonElement>('[data-respond]').forEach(b => b.onclick = () => run(() => api.respondFriend(b.dataset.respond!, b.dataset.accept === 'true')));
            for (const action of ['unfriend', 'block', 'unblock'] as const)
                body.querySelectorAll<HTMLButtonElement>(`[data-${action}]`).forEach(b => b.onclick = () => run(() => api.friendAction(b.dataset[action]!, action)));
            body.querySelectorAll<HTMLButtonElement>('[data-invite-friend]').forEach(b => b.onclick = () => invite(friends.find(f => f.id === b.dataset.inviteFriend)!));
            body.querySelectorAll<HTMLButtonElement>('[data-accept-invite]').forEach(b => b.onclick = async () => { b.disabled = true; message('Menerima undangan…'); try {
                const membership = await api.acceptNotification(b.dataset.acceptInvite!);
                if (version !== generation)
                    return;
                const world = await api.world(membership.worldId);
                if (version !== generation)
                    return;
                dialog.close();
                window.dispatchEvent(new CustomEvent('patra-world-join', { detail: { id: world.id, name: world.name } }));
                void poll();
            }
            catch (err) {
                if (version === generation) {
                    message(errorText(err));
                    b.disabled = false;
                }
            } });
            void poll();
        }
        catch (err) {
            if (version === generation)
                message(errorText(err));
        }
    }
    async function run(action: () => Promise<unknown>) { const version = generation; message('Menyimpan…'); try {
        await action();
        if (version === generation)
            await render();
    }
    catch (err) {
        if (version === generation)
            message(errorText(err));
    } }
    async function invite(friend: Friend) {
        const version = generation;
        message('Memuat dunia yang dapat kamu undang…');
        try {
            const worlds = await api.worlds();
            const allowed: World[] = [];
            for (const w of worlds) {
                const members = await api.members(w.id);
                const role = members.find(m => m.userId === user?.id)?.role;
                if (role === 'owner' || role === 'moderator')
                    allowed.push(w);
            }
            if (version !== generation)
                return;
            const host = body.querySelector('#friend-invite')!;
            host.innerHTML = `<h3>Undang @${escape(friend.username)}</h3>${allowed.length ? `<form id="invite-friend"><label>Dunia privat<select name="world">${allowed.map(w => `<option value="${escape(w.id)}">${escape(w.name)}</option>`).join('')}</select></label><button type="submit">Kirim undangan 24 jam</button></form>` : '<p>Buat dunia atau minta role moderator melalui panel Dunia privat untuk mengundang.</p>'}`;
            host.querySelector<HTMLFormElement>('form')?.addEventListener('submit', async (e) => { e.preventDefault(); const form = e.currentTarget as HTMLFormElement; const world = (form.elements.namedItem('world') as HTMLSelectElement).value; const button = form.querySelector<HTMLButtonElement>('button')!; button.disabled = true; try {
                await api.issueInvite(world, friend.username);
                if (version === generation)
                    message('Undangan dikirim ke notifikasi dalam aplikasi temanmu.');
            }
            catch (err) {
                if (version === generation)
                    message(errorText(err));
            }
            finally {
                button.disabled = false;
            } });
            message('Pilih dunia untuk undangan.');
        }
        catch (err) {
            if (version === generation)
                message(errorText(err));
        }
    }
    function friendRow(f: Friend) { const actions = f.accepted ? `<button data-invite-friend="${escape(f.id)}" type="button">Undang</button><button data-unfriend="${escape(f.id)}" type="button">Hapus teman</button>` : f.incoming ? `<button data-respond="${escape(f.requestId)}" data-accept="true" type="button">Terima</button><button data-respond="${escape(f.requestId)}" data-accept="false" type="button">Tolak</button>` : '<span>Menunggu jawaban</span>'; return `<li>@${escape(f.username)} ${f.accepted ? (f.online ? '· Online' : '· Offline') : ''} ${actions} <button data-block="${escape(f.id)}" type="button">Blokir</button></li>`; }
    document.querySelector<HTMLButtonElement>('#social-button')!.onclick = () => { if (!dialog.open)
        dialog.showModal(); void render(); };
    document.querySelector<HTMLButtonElement>('#close-social')!.onclick = () => dialog.close();
    window.addEventListener('patra-auth-change', e => { generation++; user = (e as CustomEvent<User | null>).detail; badge.textContent = ''; if (dialog.open)
        void render();
    else
        void poll(); });
    const timer = setInterval(() => void poll(), 10000);
    window.addEventListener('pagehide', () => clearInterval(timer));
}
const errorText = (e: unknown) => e instanceof Error ? e.message : 'Layanan teman belum tersedia.';
