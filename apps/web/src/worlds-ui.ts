import { api, type User, type World, type Member, type Invite } from './net/api.ts';
import './worlds.css';
const escape = (s: string) => s.replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])!);
export function mountWorlds() {
    const dialog = document.querySelector<HTMLDialogElement>('#worlds-dialog')!;
    const body = document.querySelector<HTMLElement>('#worlds-content')!;
    const openButton = document.querySelector<HTMLButtonElement>('#worlds-button')!;
    let user: User | null = null, pendingInvite: string | null = null;
    const match = /^#invite=([A-Za-z0-9_-]{43})$/.exec(location.hash);
    if (match) {
        pendingInvite = match[1];
        history.replaceState(null, '', location.pathname + location.search);
    }
    const message = (s: string) => {
        const el = body.querySelector<HTMLElement>('.worlds-message');
        if (el)
            el.textContent = s;
    };
    const show = () => {
        if (!dialog.open)
            dialog.showModal();
    };
    const login = () => { show(); body.innerHTML = '<h2 id="worlds-title">Masuk untuk menerima undangan</h2><p>Dunia privat hanya untuk akun yang diundang. Masuk, lalu terima undangan di sini.</p><button type="button" id="world-login">Masuk atau daftar</button>'; body.querySelector<HTMLButtonElement>('#world-login')!.onclick = () => window.dispatchEvent(new Event('patra-login-request')); };
    async function picker() {
        if (!user) {
            login();
            return;
        }
        show();
        body.innerHTML = '<h2 id="worlds-title">Dunia privatmu</h2><p class="worlds-message" role="status">Memuat dunia...</p>';
        try {
            const worlds = await api.worlds();
            body.innerHTML = `<h2 id="worlds-title">Dunia privatmu</h2><p>Pilih dunia yang kamu ikuti, atau buat tempat baru untuk teman. Berjalan bersama hingga 20 pemain. Voice sedang disiapkan.</p><div class="worlds-list">${worlds.map(w => `<button type="button" data-world="${escape(w.id)}">✦ ${escape(w.name)} <span>↗</span></button>`).join('') || '<p>Belum ada dunia. Buat duniamu di bawah.</p>'}</div><form id="create-world"><label>Nama dunia<input name="name" maxlength="64" required></label><button type="submit">Buat dunia</button></form><p class="worlds-message" role="status"></p>`;
            body.querySelectorAll<HTMLButtonElement>('[data-world]').forEach(b => b.onclick = () => detail(worlds.find(w => w.id === b.dataset.world)!));
            body.querySelector<HTMLFormElement>('#create-world')!.onsubmit = async (e) => {
                e.preventDefault();
                const f = e.currentTarget as HTMLFormElement;
                const button = f.querySelector<HTMLButtonElement>('button')!;
                button.disabled = true;
                message('Membuat dunia...');
                try {
                    await detail(await api.createWorld((f.elements.namedItem('name') as HTMLInputElement).value));
                }
                catch (err) {
                    message(errorText(err));
                    button.disabled = false;
                }
            };
        }
        catch (err) {
            message(errorText(err));
        }
    }
    async function detail(world: World) {
        if (!user)
            return login();
        show();
        body.innerHTML = `<h2 id="worlds-title">${escape(world.name)}</h2><p class="worlds-message" role="status">Memuat anggota...</p>`;
        try {
            const members = await api.members(world.id);
            const self = members.find(m => m.userId === user?.id);
            const canInvite = self?.role === 'owner' || self?.role === 'moderator';
            const owner = self?.role === 'owner';
            body.innerHTML = `<button type="button" id="world-back" class="worlds-back">← Dunia saya</button><h2 id="worlds-title">${escape(world.name)}</h2><p>Berjalan bersama anggota dunia ini. Mini-game bersama dan voice sedang disiapkan.</p><button type="button" id="join-world">Masuk dunia privat</button><h3>Anggota ${members.length}/50</h3><ul class="worlds-members">${members.map(m => memberRow(m, self?.role || 'member', user!.id)).join('')}</ul>${canInvite ? `<h3>Undang teman</h3><p>Username tujuan menerima notifikasi dalam aplikasi. Daftar teman juga menyediakan tombol Undang. Mute 15 menit/1 jam/24 jam adalah usulan default produk.</p><form id="issue-invite"><label>Username tujuan (opsional)<input name="username" pattern="[A-Za-z0-9_]{3,24}" placeholder="Kosongkan untuk tautan bebas"></label><button type="submit">Buat tautan 24 jam</button></form><div id="invite-link"></div><h3>Tautan yang dibuat</h3><div id="invite-list"></div><h3>Akun diblokir</h3><div id="ban-list"></div>` : ''}${owner ? '<button type="button" id="archive-world" class="worlds-danger">Arsipkan dunia</button>' : ''}<p class="worlds-message" role="status"></p>`;
            body.querySelector<HTMLButtonElement>('#world-back')!.onclick = picker;
            body.querySelector<HTMLButtonElement>('#join-world')!.onclick = () => { dialog.close(); window.dispatchEvent(new CustomEvent('patra-world-join', { detail: { id: world.id, name: world.name } })); };
            body.querySelectorAll<HTMLButtonElement>('[data-member-action]').forEach(b => b.onclick = async () => {
                if (!confirm(`Lanjutkan tindakan ${b.dataset.memberAction} untuk anggota ini?`))
                    return;
                try {
                    const action = b.dataset.memberAction!, target = b.dataset.member!;
                    if (action === 'mute') {
                        const select = body.querySelector<HTMLSelectElement>(`[data-mute-duration="${target}"]`)!;
                        await api.moderate(world.id, { kind: 'mute', userId: target, seconds: Number(select.value) });
                    }
                    else if (action === 'moderator' || action === 'member')
                        await api.changeRole(world.id, target, action);
                    else if (action === 'transfer')
                        await api.transferWorld(world.id, target);
                    else
                        await api.memberAction(world.id, target, action as 'kick' | 'ban');
                    await detail(world);
                }
                catch (err) {
                    message(errorText(err));
                }
            });
            if (canInvite) {
                body.querySelector<HTMLFormElement>('#issue-invite')!.onsubmit = async (e) => {
                    e.preventDefault();
                    const f = e.currentTarget as HTMLFormElement;
                    const button = f.querySelector<HTMLButtonElement>('button')!;
                    button.disabled = true;
                    message('Membuat undangan...');
                    try {
                        const invite = await api.issueInvite(world.id, (f.elements.namedItem('username') as HTMLInputElement).value);
                        const link = new URL(location.pathname + location.search, location.origin);
                        link.hash = `invite=${invite.token}`;
                        const host = body.querySelector<HTMLElement>('#invite-link')!;
                        host.innerHTML = '<label>Tautan sekali pakai<input readonly></label><button type="button">Salin tautan</button>';
                        host.querySelector<HTMLInputElement>('input')!.value = link.href;
                        host.querySelector<HTMLButtonElement>('button')!.onclick = async () => {
                            try {
                                await navigator.clipboard.writeText(link.href);
                                message('Tautan disalin.');
                            }
                            catch {
                                message('Salin tautan dari kotak di atas.');
                            }
                        };
                        message('Undangan dibuat. Hanya orang yang menerima tautan dapat menggunakannya.');
                        await loadInvites(world.id);
                    }
                    catch (err) {
                        message(errorText(err));
                    }
                    finally {
                        button.disabled = false;
                    }
                };
                await loadInvites(world.id);
                await loadBans(world.id);
            }
            if (owner)
                body.querySelector<HTMLButtonElement>('#archive-world')!.onclick = async () => {
                    if (!confirm('Arsipkan dunia ini untuk semua anggota?'))
                        return;
                    try {
                        await api.archiveWorld(world.id);
                        await picker();
                    }
                    catch (err) {
                        message(errorText(err));
                    }
                };
        }
        catch (err) {
            message(errorText(err));
        }
    }
    async function loadInvites(id: string) {
        try {
            const invites = await api.invites(id);
            const host = body.querySelector<HTMLElement>('#invite-list');
            if (!host)
                return;
            host.innerHTML = invites.map(i => inviteRow(i)).join('') || '<p>Belum ada tautan.</p>';
            host.querySelectorAll<HTMLButtonElement>('[data-revoke]').forEach(b => b.onclick = async () => {
                try {
                    await api.revokeInvite(id, b.dataset.revoke!);
                    await loadInvites(id);
                }
                catch (err) {
                    message(errorText(err));
                }
            });
        }
        catch (err) {
            message(errorText(err));
        }
    }
    async function loadBans(id: string) {
        try {
            const bans = await api.bans(id);
            const host = body.querySelector<HTMLElement>('#ban-list');
            if (!host)
                return;
            host.innerHTML = bans.map(m => `<div class="worlds-invite"><span>@${escape(m.username)}</span><button type="button" data-unban="${escape(m.userId)}">Batalkan ban</button></div>`).join('') || '<p>Tidak ada akun diblokir.</p>';
            host.querySelectorAll<HTMLButtonElement>('[data-unban]').forEach(b => b.onclick = async () => {
                try {
                    await api.memberAction(id, b.dataset.unban!, 'unban');
                    await loadBans(id);
                }
                catch (err) {
                    message(errorText(err));
                }
            });
        }
        catch (err) {
            message(errorText(err));
        }
    }
    function inviteRow(i: Invite) { const expired = new Date(i.expiresAt).getTime() <= Date.now(); return `<div class="worlds-invite"><span>${i.targetUsername ? `@${escape(i.targetUsername)}` : 'Tautan bebas'} · ${i.redeemed ? 'Dipakai' : i.revoked ? 'Dicabut' : expired ? 'Kedaluwarsa' : 'Aktif sampai ' + escape(new Date(i.expiresAt).toLocaleString('id-ID'))}</span>${!i.redeemed && !i.revoked && !expired ? `<button type="button" data-revoke="${escape(i.id)}">Cabut</button>` : ''}</div>`; }
    function memberRow(m: Member, role: string, selfID: string) { const own = m.userId === selfID; const canAct = !own && (role === 'owner' || role === 'moderator') && m.role !== 'owner'; return `<li><span>@${escape(m.username)} <small>${escape(m.role)}</small></span>${canAct ? `<span class="worlds-actions">${role === 'owner' ? `<button data-member-action="${m.role === 'moderator' ? 'member' : 'moderator'}" data-member="${escape(m.userId)}">${m.role === 'moderator' ? 'Turunkan' : 'Moderator'}</button><button data-member-action="transfer" data-member="${escape(m.userId)}">Transfer owner</button>` : ''}<select aria-label="Durasi mute usulan default" data-mute-duration="${escape(m.userId)}"><option value="900">15 menit</option><option value="3600">1 jam</option><option value="86400">24 jam</option></select><button data-member-action="mute" data-member="${escape(m.userId)}">Mute chat</button><button data-member-action="kick" data-member="${escape(m.userId)}">Keluarkan</button><button data-member-action="ban" data-member="${escape(m.userId)}">Ban</button></span>` : ''}</li>`; }
    async function accept() {
        if (!pendingInvite)
            return;
        if (!user) {
            login();
            return;
        }
        show();
        body.innerHTML = '<h2 id="worlds-title">Terima undangan</h2><p>Masuk ke dunia privat ini sebagai anggota?</p><button type="button" id="accept-invite">Terima undangan</button><button type="button" id="cancel-invite">Nanti</button><p class="worlds-message" role="status"></p>';
        body.querySelector<HTMLButtonElement>('#cancel-invite')!.onclick = () => { pendingInvite = null; dialog.close(); };
        body.querySelector<HTMLButtonElement>('#accept-invite')!.onclick = async () => {
            const b = body.querySelector<HTMLButtonElement>('#accept-invite')!;
            b.disabled = true;
            message('Menerima undangan...');
            try {
                const m = await api.redeemInvite(pendingInvite!);
                pendingInvite = null;
                await detail(await api.world(m.worldId));
            }
            catch (err) {
                message(errorText(err));
                b.disabled = false;
            }
        };
    }
    window.addEventListener('patra-auth-change', e => {
        user = (e as CustomEvent<User | null>).detail;
        if (pendingInvite) {
            accept();
        }
        else if (dialog.open) {
            picker();
        }
    });
    window.addEventListener('patra-worlds-request',()=>{document.querySelector<HTMLDialogElement>('#settings-dialog')?.close();if(!user){window.dispatchEvent(new Event('patra-login-request'));return}void picker()});
    openButton.onclick = () => pendingInvite ? accept() : picker();
    document.querySelector<HTMLButtonElement>('#close-worlds')!.onclick = () => dialog.close();
    if (pendingInvite)
        login();
}
function errorText(error: unknown) { return error instanceof Error ? error.message : 'Layanan dunia belum tersedia.'; }
