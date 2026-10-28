import { WorldSocket, type ConnectionStatus } from './net/socket';
import { SnapshotBuffer, type Snapshot } from './net/reconcile';
import { ServerClock } from './net/clock';
import type { createWorld } from './world.js';

export function mountOnline(world: ReturnType<typeof createWorld>) {
    const panel = document.createElement('div');
    panel.className = 'online-status';
    panel.setAttribute('role', 'status');
    panel.innerHTML = '<span>Solo lokal · tidak tersambung</span><button type="button" hidden>Ambil alih kontrol</button><button type="button" hidden>Kembali ke solo</button><button type="button" hidden>Coba sambungkan</button>';
    document.querySelector('#world')!.parentElement!.append(panel);
    const label = panel.querySelector('span')!;
    const [takeover, leave, retry] = Array.from(panel.querySelectorAll('button'));
    let socket: WorldSocket | null = null;
    let attached = false;
    let authenticated = false;
    let intent = 0;
    let buffer = new SnapshotBuffer(), latest: Snapshot | null = null;
    let clock = new ServerClock(), name = '';
    const messages: Record<ConnectionStatus, string> = {
        leaving: 'Keluar dari dunia…',
        leave_unconfirmed: 'Keluar belum dikonfirmasi. Sambungkan kembali untuk melepas kursi.',
        solo: 'Solo lokal · tidak tersambung',
        connecting: 'Menyambungkan', online: 'Online',
        reconnecting: 'Koneksi terputus · menyambungkan ulang',
        takeover_required: 'Akun sedang mengendalikan avatar di tab lain. Ambil alih atau kembali ke solo.',
        world_full: 'Dunia penuh (20 pemain termasuk kursi reconnect).',
        access_denied: 'Sesi atau izin dunia berakhir. Masuk/pilih dunia lagi.',
        resume_expired: 'Waktu reconnect habis. Sambungkan lagi untuk kursi baru.',
        incompatible: 'Versi dunia tidak cocok. Muat ulang halaman.'
    };
    function status(state: ConnectionStatus) {
        const text = !attached && state !== 'solo'
            ? `Solo lokal · ${state === 'online' ? 'Melepas kursi dunia…' : messages[state]}`
            : `${messages[state]}${state === 'online' ? ` · ${name} · ${latest?.avatars.length ?? 0}/20` : ''}`;
        label.textContent = text;
        const summary = document.querySelector('#connection-summary');
        if (summary) summary.textContent = text;
        takeover.hidden = !authenticated || state !== 'takeover_required';
        leave.hidden = !attached;
        retry.hidden = !authenticated || !['resume_expired', 'world_full', 'leave_unconfirmed'].includes(state);
        world.setConnectionStatus(attached ? state : 'solo');
        if(attached&&socket)window.dispatchEvent(new CustomEvent('patra-world-status',{detail:{socket,state}}));
    }
    async function stop() {
        // Local mode never depends on server acknowledgement. Keep the socket
        // separately until its seat is confirmed released or explicitly recovered.
        if(attached){
            attached=false;
            world.setOnline(null);
            window.dispatchEvent(new CustomEvent('patra-world-attachment',{detail:null}));
        }
        latest = null;buffer=new SnapshotBuffer();clock=new ServerClock();
        const leaving = socket;
        const released = !leaving || await leaving.close();
        if (socket !== leaving) return socket === null;
        if (!released) {
            status('leave_unconfirmed');
            return false;
        }
        socket = null;
        status('solo');
        return true;
    }
    let switching = Promise.resolve();
    function join(id: string, title: string) {
        const requested = ++intent;
        switching = switching.then(async () => {
            if (!authenticated || requested !== intent) return;
            if (!await stop() || !authenticated || requested !== intent) return;
            name = title;
            buffer = new SnapshotBuffer();
            clock = new ServerClock();
            const next = new WorldSocket(id, snapshot => {
                if (socket !== next) return;
                if (!attached || !authenticated) {
                    // Recovery can release a pending seat without reattaching the
                    // user to online rendering/input or interrupting local play.
                    void stop();
                    return;
                }
                latest = snapshot;
                buffer.push(snapshot);
                clock.observe(snapshot.timeMs);
                world.setOnlineSnapshot(snapshot, buffer.at(clock.now() - 100));
                status('online');
            }, state => { if (socket === next) status(state); });
            socket = next;
            attached = true;
            world.setOnline(next);
            next.onEvent=(type,payload)=>{if(attached&&socket===next&&authenticated)window.dispatchEvent(new CustomEvent('patra-world-event',{detail:{socket:next,type,payload}}));};
            window.dispatchEvent(new CustomEvent('patra-world-attachment',{detail:next}));
            next.connect();
        });
    }
    window.addEventListener('patra-solo-request',()=>{intent++;void stop()});
    takeover.onclick = () => { if (authenticated) socket?.connect(true); };
    leave.onclick = () => { intent++; void stop(); };
    retry.onclick = () => {
        if (!authenticated || !socket) return;
        if (socket.status === 'leave_unconfirmed') {
            socket.connect();
            return;
        }
        join(socket.worldId, name);
    };
    window.addEventListener('patra-world-join', event => {
        const detail = (event as CustomEvent<{ id: string; name: string }>).detail;
        join(detail.id, detail.name);
    });
    window.addEventListener('patra-auth-change', event => {
        authenticated = Boolean((event as CustomEvent).detail);
        if (!authenticated) {
            intent++;
            void stop();
        } else if (socket) status(socket.status);
    });
    world.setOnlineSampler(() => {
        if (attached && socket?.status === 'online' && latest) {
            world.setOnlineSnapshot(latest, buffer.at(clock.now() - 100));
        }
    });
    window.addEventListener('pagehide', event => {
        if (!event.persisted) {
            intent++;
            authenticated = false;
            void stop();
            world.setOnlineSampler(undefined);
        }
    });
}
