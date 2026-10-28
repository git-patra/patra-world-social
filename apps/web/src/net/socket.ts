import {validateActivitySnapshot,type ActivitySnapshot} from '../activities/contracts';
import {ServerClock} from './clock';
import { validateEvent } from '../world/avatars';
import { validateSnapshot, type Snapshot } from './reconcile';
export type ConnectionStatus = 'solo' | 'connecting' | 'online' | 'reconnecting' | 'takeover_required' | 'world_full' | 'access_denied' | 'resume_expired' | 'incompatible' | 'leaving' | 'leave_unconfirmed';
/** Seat credentials are memory-only and distinct from the HttpOnly auth cookie. */
export class WorldSocket {
    private clock=new ServerClock();
 serverNow(){return this.clock.now()}
 private ws: WebSocket | null = null;
    private seat: Snapshot | null = null;
    private seq = 0;
    private stopped = false;
    private terminal = false;
    private retry: ReturnType<typeof setTimeout> | undefined;
    private offlineSince = 0;
    private leavePromise: Promise<boolean> | null = null;
    private leaveID: string | null = null;
    private completeLeave: ((released: boolean) => void) | null = null;
    onEvent: (type: string, payload: Record<string, unknown>) => void = () => { };
    private startCommands = new Map<string,string>();
    private pending = new Map<string, {
        resolve: (value: unknown) => void;
        reject: (error: Error) => void;
        timer: ReturnType<typeof setTimeout>;
    }>();
    status: ConnectionStatus = 'connecting';
    constructor(readonly worldId: string, private snapshot: (s: Snapshot) => void, private changed: (s: ConnectionStatus) => void, private origin = location.origin) { }
    private state(s: ConnectionStatus) { this.status = s; this.changed(s); }
    connect(takeover = false) {
        clearTimeout(this.retry);
        this.leavePromise = null;
        this.stopped = false;
        this.terminal = false;
        if (this.ws) {
            this.ws.onclose = null;
            this.ws.close();
        }
        if (takeover) {
            this.seat = null;
            this.seq = 0;
        }
        this.state(this.seat ? 'reconnecting' : 'connecting');
        const url = new URL('/api/v1/realtime', this.origin);
        url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
        url.searchParams.set('worldId', this.worldId);
 url.searchParams.set('protocolVersion','2');
        if (takeover)
            url.searchParams.set('takeover', 'true');
        if (this.seat) {
            url.searchParams.set('sessionId', this.seat.sessionId);
            url.searchParams.set('epoch', String(this.seat.epoch));
        }
        const ws = this.ws = new WebSocket(url.href);
        ws.onmessage = event => {
            if (ws !== this.ws || this.stopped)
                return;
            try {
                const e = validateEvent(JSON.parse(event.data));
 this.clock.observe(e.serverTimeMs);
                if (this.leaveID && e.type === 'command.ack' && e.payload.commandId === this.leaveID) {
                    this.completeLeave?.(true);
                    return;
                }
                if (this.completeLeave)
                    return;
                if (e.type === 'command.ack' || e.type === 'command.error') {
                    const id = String(e.payload.commandId), pending = this.pending.get(id);
                    if (pending) {
                        clearTimeout(pending.timer);
                        this.pending.delete(id);
                        if (e.type === 'command.error')
                            pending.reject(Object.assign(new Error(String(e.payload.message || 'Perintah ditolak atau belum tersedia.')),{code:String(e.payload.code||'invalid_or_unavailable')}));
                        else
                            pending.resolve(e.payload.result);
                    }
                }
                if(e.type==='activity.snapshot'||e.type==='activity.lifecycle'){if(e.payload.worldId===this.worldId)this.onEvent(e.type,e.payload);return;}
 if (['race.snapshot','race.finished','duck.snapshot','duck.finished','duck.shot_resolved'].includes(e.type)) {
 if(e.payload.worldId === this.worldId) this.onEvent(e.type,e.payload);
 return;
 }
 if (e.type==='pingpong.snapshot' || e.type.startsWith('penalty.') || e.type === 'chat.message' || e.type === 'chat.deleted' || e.type === 'lobby.updated' || e.type === 'match.started' || e.type === 'match.result_status' || e.type === 'leaderboard.changed') {
                    this.onEvent(e.type, e.payload);
                    return;
                }
                if (e.type === 'command.error') {
                    const code = String(e.payload.code);
                    if (['incompatible','takeover_required', 'world_full', 'access_denied', 'resume_expired'].includes(code)) {
                        this.terminal = true;
                        this.state(code as ConnectionStatus);
                    }
                    return;
                }
                if (e.type !== 'world.snapshot' && e.type !== 'world.delta')
                    return;
                const next = validateSnapshot(e.payload, this.worldId);
                if (this.seat && next.epoch < this.seat.epoch)
                    return;
                if (!this.seat || this.seat.epoch !== next.epoch)
                    this.seq = 0;
                this.seat = next;
                this.offlineSince = 0;
                this.state('online');
                this.snapshot(next);
                if (e.type === 'world.snapshot')
                    this.onEvent(e.type, e.payload);
            }
            catch {
                this.terminal = true;
                this.state('incompatible');
                ws.close();
            }
        };
        ws.onclose = event => {
            if (this.completeLeave) {
                this.completeLeave(false);
                return;
            }
            if (ws !== this.ws || this.stopped || this.terminal)
                return;
            if (event.code === 1008) {
                this.terminal = true;
                this.state('access_denied');
                return;
            }
            const now = Date.now();
            this.offlineSince ||= now;
            if (now - this.offlineSince >= 30000) {
                this.terminal = true;
                this.state('resume_expired');
                return;
            }
            this.state('reconnecting');
            this.retry = setTimeout(() => this.connect(), 1000);
        };
    }
    get selfId() { return this.seat?.selfId ?? ''; }
    private send(type: string, payload: Record<string, unknown>, commandId?: string) {
        if (this.status !== 'online' || this.ws?.readyState !== 1 || this.ws.bufferedAmount > 16384)
            return;
        this.seq++;
        const id = commandId ?? `${this.seat?.epoch ?? 1}-${this.seq}`;
        this.ws.send(JSON.stringify({ v: 2, commandId: id, seq: this.seq, worldId: this.worldId, type, payload }));
        return id;
    }
    sendChat(clientMessageId: string, text: string): Promise<unknown> {
        const id = this.send('chat.send', { channelId: 'world', clientMessageId, text });
        if (!id)
            return Promise.reject(new Error('Chat memerlukan koneksi dunia aktif.'));
        return new Promise((resolve, reject) => {
            const timer = setTimeout(() => { this.pending.delete(id); reject(new Error('Belum dikonfirmasi. Coba kirim ulang pesan yang sama.')); }, 5000);
            this.pending.set(id, { resolve, reject, timer });
        });
    }
    gameCommand(type: string, payload: Record<string, unknown>): Promise<unknown> {
        let commandId:string=crypto.randomUUID();
        if(type==='match.start'){const key=String(payload.lobbyId);commandId=this.startCommands.get(key)??commandId;this.startCommands.set(key,commandId);}
        const id = this.send(type, payload, commandId);
        if (!id) return Promise.reject(new Error('Lobby memerlukan koneksi dunia aktif.'));
        return new Promise((resolve,reject) => {
            const timer=setTimeout(()=>{this.pending.delete(id);reject(new Error('Belum dikonfirmasi. Sambungkan ulang untuk menyinkronkan lobby.'));},type==='match.start'?15000:5000);
            this.pending.set(id,{resolve,reject,timer});
        });
    }
    async activitySnapshots(): Promise<ActivitySnapshot[]> {
        const result = await this.gameCommand('activity.snapshot', {}) as { activities: unknown[] };
        if (!Array.isArray(result?.activities) || result.activities.length !== 6)
            throw Error('Invalid activity resync');
        const snapshots = result.activities.map(a => validateActivitySnapshot(a, this.worldId));
        if (new Set(snapshots.filter(a => a.kind === 'area').map(a => a.areaId)).size !== 4 || snapshots.filter(a => a.kind === 'vehicles').length !== 1 || snapshots.filter(a => a.kind === 'football').length !== 1)
            throw Error('Invalid activity resync');
        return snapshots;
    }
    input(dx: number, dy: number) { this.send('player.input', { dx, dy }); }
    destination(x: number, y: number) { this.send('player.destination', { x, y }); }
    /** A deliberate exit is complete only after the server releases its seat. */
    close(): Promise<boolean> {
        if (this.leavePromise)
            return this.leavePromise;
        clearTimeout(this.retry);
        const ws = this.ws;
        const alreadyReleased = !ws && !this.seat || ['solo', 'takeover_required', 'world_full', 'access_denied', 'resume_expired'].includes(this.status);
        if (alreadyReleased) {
            this.finishClose(true);
            return Promise.resolve(true);
        }
        if (ws?.readyState !== 1 || this.status !== 'online') {
            this.finishClose(false);
            return Promise.resolve(false);
        }
        this.leaveID = this.send('world.leave', {}) ?? null;
        if (!this.leaveID) {
            this.finishClose(false);
            return Promise.resolve(false);
        }
        this.state('leaving');
        this.terminal = true;
        this.leavePromise = new Promise(resolve => {
            const timeout = setTimeout(() => this.completeLeave?.(false), 3000);
            this.completeLeave = released => { clearTimeout(timeout); this.completeLeave = null; this.finishClose(released); resolve(released); };
        });
        return this.leavePromise;
    }
    private finishClose(released: boolean) {
        this.stopped = true;
        this.terminal = true;
        this.leaveID = null;
        for (const pending of this.pending.values()) {
            clearTimeout(pending.timer);
            pending.reject(new Error('Koneksi chat ditutup.'));
        }
        this.pending.clear();
        if (this.ws) {
            this.ws.onclose = null;
            this.ws.close();
        }
        this.ws = null;
        if (released)
            this.seat = null;
        this.state(released ? 'solo' : 'leave_unconfirmed');
    }
}
