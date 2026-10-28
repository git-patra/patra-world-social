import { api, type ChatMessage, type User } from './net/api';
import type { WorldSocket } from './net/socket';
import { mergeMessages, validMessage } from './chat-model';
import './social.css';
export function mountChat() {
    const panel = document.querySelector<HTMLElement>('#chat-panel')!;
    const button = document.querySelector<HTMLButtonElement>('#chat-button')!;
    const badge = document.querySelector('#chat-badge')!;
    const status = document.querySelector('#chat-status')!;
    const list = document.querySelector('#chat-messages')!;
    const form = document.querySelector<HTMLFormElement>('#chat-form')!;
    const input = form.querySelector<HTMLTextAreaElement>('textarea')!;
    const send = form.querySelector<HTMLButtonElement>('button')!;
    const older = document.querySelector<HTMLButtonElement>('#chat-older')!;
    const latest = document.querySelector<HTMLButtonElement>('#chat-latest')!;
    let socket: WorldSocket | null = null, user: User | null = null, messages: ChatMessage[] = [], generation = 0, unread = 0, canModerate = false;
    const deletedIDs = new Set<string>(), seenIDs = new Set<string>();
    let oldest = '', draft: {
        text: string;
        id: string;
    } | null = null, sending = false, loading = false, historyVersion = 0;
    const notice = (text: string) => { status.textContent = text; };
    function render() {
        list.replaceChildren();
        for (const m of messages) {
            const row = document.createElement('p'), text = document.createElement('span');
            text.textContent = `@${m.username}: ${m.deleted ? '[Pesan dihapus]' : m.text}`;
            row.append(text);
            if (canModerate && !m.deleted) {
                const remove = document.createElement('button');
                remove.type = 'button';
                remove.textContent = 'Hapus';
                remove.onclick = async () => { const current = socket; if (!current)
                    return; remove.disabled = true; try {
                    await api.moderate(current.worldId, { kind: 'delete', messageId: m.id });
                }
                catch (e) {
                    notice(errorText(e));
                    remove.disabled = false;
                } };
                row.append(remove);
            }
            list.append(row);
        }
        badge.textContent = unread ? String(unread) : '';
    }
    async function history(replace: boolean) {
        const current = socket, version = generation;
        if (!current || (loading && !replace))
            return;
        const request = ++historyVersion;
        loading = true;
        messages = [];
        render();
        older.disabled = true;
        notice('Memuat history 30 hari…');
        try {
            const page = await api.chatHistory(current.worldId, replace ? '' : oldest);
            if (current !== socket || version !== generation || request !== historyVersion)
                return;
            if (!page.every(m => validMessage(m, current.worldId)))
                throw Error('Format history tidak cocok.');
            // A live event can arrive while the request is in flight. Keep it during resync.
            messages = mergeMessages(messages, page, deletedIDs);
            page.forEach(m => seenIDs.add(m.id));
            oldest = page.at(-1)?.id || oldest;
            older.hidden = page.length < 50;
            render();
            notice('History 30 hari · pesan plain text.');
        }
        catch (e) {
            if (version === generation && request === historyVersion)
                notice(errorText(e));
        }
        finally {
            if (version === generation && request === historyVersion) {
                loading = false;
                older.disabled = false;
            }
        }
    }
    function attach(next: WorldSocket | null) {
        generation++;
        historyVersion++;
        socket = next;
        deletedIDs.clear();
        seenIDs.clear();
        messages = [];
        unread = 0;
        oldest = '';
        draft = null;
        input.value = '';
        sending = false;
        loading = false;
        canModerate = false;
        send.disabled = true;
        older.hidden = true;
        render();
        notice(next ? 'Menyambungkan chat…' : 'Masuk dunia privat untuk chat.');
        if (next) {
            const version = generation;
            void api.members(next.worldId).then(members => { if (version !== generation)
                return; const role = members.find(m => m.userId === user?.id)?.role; canModerate = role === 'owner' || role === 'moderator'; render(); }).catch(e => { if (version === generation)
                notice(errorText(e)); });
        }
    }
    button.onclick = () => { panel.hidden = !panel.hidden; if (!panel.hidden) {
        unread = 0;
        render();
        if (socket && messages.length === 0)
            void history(true);
        input.focus();
    } };
    document.querySelector<HTMLButtonElement>('#close-chat')!.onclick = () => { panel.hidden = true; };
    older.onclick = () => void history(false);
    latest.onclick = () => void history(true);
    form.onsubmit = async (e) => {
        e.preventDefault();
        const current = socket;
        if (!current || sending)
            return;
        const text = input.value;
        if (!text.trim() || Array.from(text).length > 500) {
            notice('Pesan harus 1–500 karakter.');
            return;
        }
        if (!draft || draft.text !== text)
            draft = { text, id: crypto.randomUUID() };
        const retry = draft, version = generation;
        sending = true;
        send.disabled = true;
        notice('Mengirim…');
        try {
            const result = await current.sendChat(retry.id, text);
            if (version !== generation)
                return;
            if (!validMessage(result, current.worldId))
                throw Error('Konfirmasi pesan tidak valid.');
            messages = mergeMessages(messages, [result], deletedIDs);
            render();
            if (input.value === text)
                input.value = '';
            draft = null;
            notice('Pesan tersimpan.');
        }
        catch (e) {
            if (version === generation)
                notice(errorText(e));
        }
        finally {
            if (version === generation) {
                sending = false;
                send.disabled = socket?.status !== 'online';
            }
        }
    };
    window.addEventListener('patra-auth-change', e => { user = (e as CustomEvent<User | null>).detail; if (!user)
        attach(null); });
    window.addEventListener('patra-world-attachment', e => attach((e as CustomEvent<WorldSocket | null>).detail));
    window.addEventListener('patra-world-status', e => { const detail = (e as CustomEvent).detail; if (detail.socket !== socket)
        return; send.disabled = sending || detail.state !== 'online'; if (detail.state !== 'online')
        notice('Chat menunggu koneksi dunia aktif.'); });
    window.addEventListener('patra-world-event', e => {
        const { socket: source, type, payload } = (e as CustomEvent).detail;
        if (source !== socket || !socket)
            return;
        if (type === 'world.snapshot') {
            void history(true);
            return;
        }
        if (type === 'chat.message' && validMessage(payload, socket.worldId)) {
            const fresh = !seenIDs.has(payload.id);
            seenIDs.add(payload.id);
            messages = mergeMessages(messages, [payload], deletedIDs);
            if (fresh && panel.hidden && payload.userId !== user?.id)
                unread++;
            render();
        }
        if (type === 'chat.deleted' && payload.worldId === socket.worldId) {
            deletedIDs.add(payload.id);
            messages = messages.map(m => m.id === payload.id ? { ...m, text: '', deleted: true } : m);
            render();
        }
    });
}
const errorText = (e: unknown) => e instanceof Error ? e.message : 'Chat belum tersedia.';
