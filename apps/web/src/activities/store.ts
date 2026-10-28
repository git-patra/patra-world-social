import { activityContract } from './contracts';
import type { ActivitySnapshot, AreaSnapshot, VehiclesSnapshot, Vehicle } from './contracts';
import { LocalVehicles, nearbyVehicle } from './vehicles';
import { audio } from '../audio/service';
import type { RaceSnapshot, Controls } from '../features/race/local';
import type { PublicPenalty } from '../features/penalty/local';
import { activityInput } from './input-router';
export type ActivityKind = 'race' | 'penalty';
export type ActivityLobby = {
    id: string;
    arenaId: string;
    gameType: string;
    worldId: string;
    state: string;
    matchId?: string;
    players: {
        userId: string;
        actorId: string;
    }[];
    spectators: string[];
};
export type ActivityGame = {
    kind: ActivityKind;
    matchId: string;
    snapshot: RaceSnapshot | PublicPenalty;
};
type Transport = {
    worldId: string;
    selfId: string;
    status: string;
    serverNow: () => number;
    activitySnapshots?: () => Promise<ActivitySnapshot[]>;
    gameCommand: (type: string, payload: Record<string, unknown>) => Promise<unknown>;
};
type LocalSession = {
    snapshot: (at: number) => RaceSnapshot | PublicPenalty | null;
    clear: () => void;
    input?: (controls: Controls, at: number) => boolean;
    choose?: (turn: string, zone: number, at: number) => boolean;
};
export type ActivityState = {
    games: ActivityGame[];
    areas: AreaSnapshot[];
    vehicles: Vehicle[];
    nearbyRide: Vehicle | null;
    active: ActivityGame | null;
    inputMode: 'walk' | 'ride' | ActivityKind;
    lobbies: ActivityLobby[];
    selfId: string;
    online: boolean;
    connected: boolean;
    now: number;
    message: string;
};
const neutral = (): Controls => ({ throttle: 0, brake: 0, steer: 0 });
const phaseOrder = { CLEARING: 1, RESERVED: 2, ACTIVE: 3, RESETTING: 4, OPEN: 5 };
function newerArea(next: AreaSnapshot, previous: AreaSnapshot) {
    if (next.generation !== previous.generation)
        return next.generation > previous.generation;
    if (next.tick < previous.tick)
        return false;
    const nextPhase = phaseOrder[next.payload.phase], oldPhase = phaseOrder[previous.payload.phase];
    return nextPhase > oldPhase || nextPhase === oldPhase && next.tick > previous.tick;
}
const penaltyPhase = { choosing: 0, animation: 1, results: 2 };
export class ActivityStore {
    private localVehicles = new LocalVehicles();
    private vehicleState: VehiclesSnapshot | null = null;
    private position = { x: 32, y: 32 };
    private rideHeld = { x: 0, y: 0 };
    private rideSending = false;
    private ridePending = false;
    private localStarting = false;
    private socket: Transport | null = null;
    private epoch = 0;
    private local: LocalSession | null = null;
    private localKind: ActivityKind | null = null;
    private resyncRequest = 0;
    private lobbyRevision = 0;
    private readSerial = 0;
    private areaRevisions = new Map<string, number>();
    private retiredIncarnations = new Map<string, Set<string>>();
    private gameReads = new Map<string, number>();
    private games = new Map<string, ActivityGame>();
    private areas = new Map<string, AreaSnapshot>();
    private lobbies: ActivityLobby[] = [];
    private listeners = new Set<() => void>();
    private sceneListeners = new Set<() => void>();
    private held = neutral();
    private sending = false;
    private lastNotify = -Infinity;
    private timer: ReturnType<typeof setInterval> | null = null;
    private state: ActivityState = { games: [], areas: [], vehicles: this.localVehicles.vehicles, nearbyRide: null, lobbies: [], active: null, inputMode: 'walk', selfId: 'guest', online: false, connected: true, now: 0, message: '' };
    constructor(private effect: (snapshot: ActivitySnapshot) => void = () => {
    }) {
    }
    snapshot = () => this.state;
    subscribe = (fn: () => void) => {
        this.listeners.add(fn);
        return () => {
            this.listeners.delete(fn);
        };
    };
    subscribeScene(fn: () => void) {
        this.sceneListeners.add(fn);
        return () => {
            this.sceneListeners.delete(fn);
        };
    }
    private publish(at = this.state.now, force = false) {
        const games = [...this.games.values()], selfId = this.socket?.selfId ?? 'guest';
        const active = games.find(g => this.localKind === g.kind && !this.socket || this.lobbies.some(l => l.matchId === g.matchId && l.players.some(p => p.userId === selfId))) ?? null;
        const area = active ? this.areas.get(active.kind) : undefined, gameMode = active && (!this.socket || !area || area.payload.phase !== 'OPEN' && area.payload.matchId === active.matchId) ? active.kind : 'walk';
        const vehicles = this.socket ? this.vehicleState?.payload.vehicles ?? [] : this.localVehicles.vehicles;
        const raceClosed = this.socket ? this.areas.get('race')?.payload.phase !== 'OPEN' : this.localStarting || this.localKind === 'race';
        const inputMode = gameMode !== 'walk' ? gameMode : !raceClosed && vehicles.some(v => v.riderId === selfId) ? 'ride' : 'walk';
        if (this.state.inputMode === 'ride' && inputMode !== 'ride')
            this.rideHeld = { x: 0, y: 0 };
        const nearbyRide = gameMode === 'walk' && !raceClosed ? nearbyVehicle(vehicles, this.position) ?? null : null;
        this.state = { ...this.state, vehicles, nearbyRide, games, areas: [...this.areas.values()], lobbies: this.lobbies, active, inputMode, selfId, online: Boolean(this.socket), connected: !this.socket || this.socket.status === 'online', now: at };
        for (const fn of this.sceneListeners)
            fn();
        if (force || at - this.lastNotify >= 100) {
            this.lastNotify = at;
            for (const fn of this.listeners)
                fn();
        }
    }
    async attach(socket: Transport | null) {
        this.controls(neutral());
        this.epoch++;
        this.resyncRequest++;
        this.local?.clear();
        this.local = null;
        this.localKind = null;
        this.socket = socket;
        this.games.clear();
        this.localVehicles.reset();
        this.vehicleState = null;
        this.rideHeld = { x: 0, y: 0 };
        this.rideSending = false;
        this.ridePending = false;
        this.localStarting = false;
        this.areas.clear();
        this.areaRevisions.clear();
        this.retiredIncarnations.clear();
        this.gameReads.clear();
        this.lobbies = [];
        this.sending = false;
        this.state = { ...this.state, message: '' };
        this.lastNotify = -Infinity;
        this.publish(socket?.serverNow() ?? performance.now(), true);
        if (socket?.status === 'online')
            await this.reload();
    }
    async reload() {
        const socket = this.socket, epoch = this.epoch;
        if (!socket || socket.status !== 'online')
            return;
        const request = ++this.resyncRequest;
        const revisions = new Map(this.areaRevisions), lobbyRevision = this.lobbyRevision;
        try {
            const [metadata, lobbies] = await Promise.all([
                socket.activitySnapshots ? socket.activitySnapshots().then(activities => ({ activities })) : socket.gameCommand('activity.snapshot', {}), socket.gameCommand('lobby.list', {})
            ]);
            if (epoch !== this.epoch || socket !== this.socket || request !== this.resyncRequest)
                return;
            this.resync((metadata as {
                activities: ActivitySnapshot[];
            }).activities, revisions);
            if (lobbyRevision === this.lobbyRevision)
                this.acceptLobbies(lobbies as {
                    worldId: string;
                    lobbies: ActivityLobby[];
                });
            await this.loadGames();
        }
        catch (e) {
            if (epoch === this.epoch && request === this.resyncRequest) {
                this.state = { ...this.state, message: (e as Error).message };
                this.publish();
            }
        }
    }
    private acceptLobbies(payload: {
        worldId?: string;
        lobbies: ActivityLobby[];
    }) {
        if (!this.socket || payload.worldId && payload.worldId !== this.socket.worldId || !Array.isArray(payload.lobbies))
            return;
        this.lobbyRevision++;
        this.lobbies = payload.lobbies;
        for (const [id] of this.games)
            if (!this.lobbies.some(l => l.matchId === id))
                this.games.delete(id);
        this.publish();
    }
    private async loadGames() {
        const socket = this.socket, epoch = this.epoch, resync = this.resyncRequest;
        if (!socket || socket.status !== 'online')
            return;
        const lobbies = this.lobbies.filter(l => (l.gameType === 'race' || l.gameType === 'penalty') && l.matchId);
        await Promise.all(lobbies.map(async (lobby) => {
            const read = ++this.readSerial;
            this.gameReads.set(lobby.matchId!, read);
            try {
                const snapshot = await socket.gameCommand(`${lobby.gameType}.snapshot`, { matchId: lobby.matchId });
                if (epoch === this.epoch && socket === this.socket && resync === this.resyncRequest && this.gameReads.get(lobby.matchId!) === read) {
                    this.acceptGame(lobby.gameType as ActivityKind, snapshot as RaceSnapshot | PublicPenalty);
                }
            }
            catch {
            }
        }));
    }
    private acceptGame(kind: ActivityKind, snapshot: RaceSnapshot | PublicPenalty) {
        if (!snapshot?.matchId || !this.lobbies.some(l => l.matchId === snapshot.matchId && l.gameType === kind))
            return;
        const previous = this.games.get(snapshot.matchId);
        if (kind === 'race' && previous && 'frame' in previous.snapshot && 'frame' in snapshot && snapshot.frame < previous.snapshot.frame)
            return;
        if (kind === 'penalty' && previous && 'turn' in previous.snapshot && 'turn' in snapshot) {
            const old = previous.snapshot;
            if (snapshot.turn < old.turn)
                return;
            if (snapshot.turn === old.turn) {
                if (penaltyPhase[snapshot.phase] < penaltyPhase[old.phase])
                    return;
                snapshot = { ...snapshot, chosen: snapshot.chosen.map((chosen, index) => chosen || old.chosen[index]) };
            }
        }
        this.games.set(snapshot.matchId, { kind, matchId: snapshot.matchId, snapshot });
        this.publish(this.socket?.serverNow());
    }
    private mergeArea(snapshot: AreaSnapshot, revisions?: Map<string, number>) {
        const previous = this.areas.get(snapshot.areaId);
        if (previous) {
            if (snapshot.incarnation === previous.incarnation) {
                if (!newerArea(snapshot, previous))
                    return false;
            }
            else {
                if (this.retiredIncarnations.get(snapshot.areaId)?.has(snapshot.incarnation))
                    return false;
                if (revisions && (revisions.get(snapshot.areaId) ?? 0) !== (this.areaRevisions.get(snapshot.areaId) ?? 0))
                    return false;
                const retired = this.retiredIncarnations.get(snapshot.areaId) ?? new Set<string>();
                retired.add(previous.incarnation);
                this.retiredIncarnations.set(snapshot.areaId, retired);
            }
        }
        this.areas.set(snapshot.areaId, snapshot);
        this.areaRevisions.set(snapshot.areaId, (this.areaRevisions.get(snapshot.areaId) ?? 0) + 1);
        return true;
    }
    receive(type: string, payload: any) {
        if (type === 'lobby.updated') {
            this.acceptLobbies(payload);
            void this.loadGames();
            return;
        }
        if (type === 'activity.snapshot' || type === 'activity.lifecycle') {
            if (!this.socket || payload.worldId !== this.socket.worldId)
                return;
            if (payload.kind === 'vehicles') {
                if (this.vehicleState && payload.incarnation !== this.vehicleState.incarnation)
                    return;
                const previous = this.vehicleState;
                this.mergeVehicles(payload);
                if (type === 'activity.lifecycle' && previous && this.vehicleState !== previous && payload.payload.vehicles.some((v: Vehicle) => v.riderId === this.socket?.selfId))
                    audio.play({ kind: 'ride', mode: 'ride', eventId: `ride:${payload.incarnation}:${payload.generation}` });
                this.publish(this.socket.serverNow());
                return;
            }
            if (payload.kind !== 'area')
                return;
            const previous = this.areas.get(payload.areaId);
            // A different runtime must be introduced by the current full resync request.
            if (previous && payload.incarnation !== previous.incarnation)
                return;
            if (!this.mergeArea(payload))
                return;
            if (type === 'activity.lifecycle' && previous)
                this.effect(payload);
            this.publish(this.socket.serverNow());
            return;
        }
        if (type.startsWith('race.') && payload?.karts)
            this.acceptGame('race', payload);
        if (type.startsWith('penalty.') && payload?.actors)
            this.acceptGame('penalty', payload);
        if (type === 'match.result_status') {
            this.state = { ...this.state, message: payload.status === 'completed' ? 'Hasil tersimpan. Buka History untuk detail.' : 'Menunggu konfirmasi penyimpanan hasil.' };
            this.publish();
        }
    }
    resync(areas: ActivitySnapshot[], revisions?: Map<string, number>) {
        if (!Array.isArray(areas))
            return;
        for (const area of areas) {
            if (area.worldId !== this.socket?.worldId)
                continue;
            if (area.kind === 'area')
                this.mergeArea(area, revisions);
            else if (area.kind === 'vehicles')
                this.mergeVehicles(area, revisions);
        }
        this.publish(this.socket?.serverNow());
    }
    private mergeVehicles(snapshot: VehiclesSnapshot, revisions?: Map<string, number>) {
        const previous = this.vehicleState, key = 'vehicles';
        if (previous) {
            if (snapshot.incarnation === previous.incarnation) {
                if (snapshot.generation < previous.generation || snapshot.generation === previous.generation && snapshot.tick <= previous.tick)
                    return;
            }
            else {
                if (this.retiredIncarnations.get(key)?.has(snapshot.incarnation) || revisions && (revisions.get(key) ?? 0) !== (this.areaRevisions.get(key) ?? 0))
                    return;
                const retired = this.retiredIncarnations.get(key) ?? new Set<string>();
                retired.add(previous.incarnation);
                this.retiredIncarnations.set(key, retired);
            }
        }
        this.vehicleState = snapshot;
        this.areaRevisions.set(key, (this.areaRevisions.get(key) ?? 0) + 1);
        const old = previous?.payload.vehicles.find(v => v.riderId === this.socket?.selfId), own = snapshot.payload.vehicles.find(v => v.riderId === this.socket?.selfId);
        if (old && !own)
            this.rideHeld = { x: 0, y: 0 };
    }
    setPosition(position: {
        x: number;
        y: number;
    }) {
        if (Math.hypot(position.x - this.position.x, position.y - this.position.y) < .001)
            return;
        this.position = { ...position };
        this.publish();
    }
    interactRide(position = this.position) {
        this.position = { ...position };
        if (this.state.inputMode === 'ride') {
            void this.rideAction('dismount');
            return true;
        }
        if (this.state.inputMode !== 'walk' || this.localStarting || this.localKind === 'race' || this.state.online && this.areas.get('race')?.payload.phase !== 'OPEN' || !this.state.connected || this.ridePending)
            return false;
        const vehicle = nearbyVehicle(this.state.vehicles, position);
        if (!vehicle)
            return false;
        void this.rideAction('mount', vehicle);
        return true;
    }
    async rideAction(action: 'mount' | 'dismount', vehicle?: Vehicle) {
        const v = vehicle ?? this.state.vehicles.find(v => v.riderId === this.state.selfId);
        if (!v || this.ridePending)
            return;
        this.rideHeld = { x: 0, y: 0 };
        if (!this.socket) {
            if (action === 'mount')
                this.localVehicles.mount(v.id, this.position, this.state.now);
            else
                this.localVehicles.dismount();
            this.publish(undefined, true);
            audio.play({ kind: 'ride', mode: 'ride', eventId: `ride:local:${v.id}:${v.generation}` });
            return;
        }
        if (!this.vehicleState || this.socket.status !== 'online')
            return;
        this.ridePending = true;
        const epoch = this.epoch;
        try {
            await this.socket.gameCommand('activity.ride', this.rideIntent(v, action));
            if (epoch === this.epoch)
                await this.reload();
        }
        catch (e) {
            if (epoch === this.epoch) {
                this.state = { ...this.state, message: (e as Error).message };
                this.publish(undefined, true);
            }
        }
        finally {
            if (epoch === this.epoch)
                this.ridePending = false;
        }
    }
    private rideIntent(v: Vehicle, action: 'mount' | 'dismount' | 'drive'): Record<string, unknown> {
        return { incarnation: this.vehicleState!.incarnation, areaGeneration: this.areas.get('race')?.generation ?? 0, vehicleId: v.id, generation: v.generation, action, ...(action === 'drive' ? { dx: this.rideHeld.x, dy: this.rideHeld.y } : {}) };
    }
    rideControls(input: {
        x: number;
        y: number;
        steer?: number;
        throttle?: number;
        brake?: number;
    }) {
        this.rideHeld = { x: input.x || input.steer || 0, y: input.y || (input.brake ?? 0) - (input.throttle ?? 0) };
        if (!this.socket)
            this.localVehicles.controls(this.rideHeld.x, this.rideHeld.y, this.state.now);
        if (!this.rideHeld.x && !this.rideHeld.y)
            this.sendRideInput();
    }
    private sendRideInput() {
        if (this.state.inputMode !== 'ride' || this.ridePending)
            return;
        if (!this.socket) {
            this.localVehicles.controls(this.rideHeld.x, this.rideHeld.y, this.state.now);
            return;
        }
        const v = this.state.vehicles.find(v => v.riderId === this.state.selfId);
        if (!v || !this.vehicleState || this.socket.status !== 'online' || this.rideSending)
            return;
        this.rideSending = true;
        const epoch = this.epoch;
        void this.socket.gameCommand('activity.ride', this.rideIntent(v, 'drive')).catch(() => {
            if (epoch === this.epoch)
                this.rideHeld = { x: 0, y: 0 };
        }).finally(() => {
            if (epoch === this.epoch)
                this.rideSending = false;
        });
    }
    async startLocal(kind: ActivityKind, at = performance.now(), seed?: number, id?: string) {
        if (this.socket)
            return;
        this.localVehicles.reset();
        this.rideHeld = { x: 0, y: 0 };
        this.localStarting = kind === 'race';
        this.publish(at, true);
        const epoch = ++this.epoch;
        this.local?.clear();
        this.games.clear();
        this.local = null;
        this.localKind = null;
        const session = kind === 'race' ? (await import('../features/race/session')).guestRace : (await import('../features/penalty/session')).guestSession;
        if (epoch !== this.epoch || this.socket)
            return;
        const snapshot = kind === 'race' ? (session as typeof import('../features/race/session').guestRace).start(at, seed, id) : (session as typeof import('../features/penalty/session').guestSession).start(at, undefined, id);
        this.localStarting = false;
        this.local = session;
        this.localKind = kind;
        this.games.set(snapshot.matchId, { kind, matchId: snapshot.matchId, snapshot });
        this.publish(at, true);
    }
    controls(controls: Controls) {
        const changed = this.held.throttle !== controls.throttle || this.held.brake !== controls.brake || this.held.steer !== controls.steer;
        this.held = { throttle: controls.throttle, brake: controls.brake, steer: controls.steer };
        if (changed && !controls.throttle && !controls.brake && !controls.steer)
            this.sendInput();
    }
    private sendInput() {
        const active = this.state.active;
        if (!active || this.state.inputMode !== 'race' || active.kind !== 'race' || !(active.snapshot as RaceSnapshot).karts.some(k => k.userId === this.state.selfId && !k.finished && !k.dnf))
            return;
        if (this.local) {
            this.local.input?.(this.held, this.state.now);
            return;
        }
        const socket = this.socket;
        if (!socket || socket.status !== 'online' || this.sending || !['countdown', 'racing'].includes(active.snapshot.phase))
            return;
        this.sending = true;
        const epoch = this.epoch;
        void socket.gameCommand('race.input', { matchId: active.matchId, ...this.held }).catch(() => {
            if (epoch === this.epoch)
                this.state = { ...this.state, message: 'Kontrol belum diterima. Menunggu sinkronisasi server.' };
        }).finally(() => {
            if (epoch === this.epoch)
                this.sending = false;
        });
    }
    tick(at = this.socket?.serverNow() ?? performance.now()) {
        this.state = { ...this.state, now: at };
        if (!this.socket)
            this.localVehicles.tick(at);
        this.sendRideInput();
        this.sendInput();
        if (this.local && this.localKind) {
            const snapshot = this.local.snapshot(at);
            if (snapshot)
                this.games.set(snapshot.matchId, { kind: this.localKind, matchId: snapshot.matchId, snapshot });
        }
        this.publish(at);
    }
    async choose(zone: number) {
        const active = this.state.active;
        if (!active || active.kind !== 'penalty' || !Number.isInteger(zone) || zone < 0 || zone > 3)
            return;
        const s = active.snapshot as PublicPenalty;
        if (s.phase !== 'choosing')
            return;
        if (this.local) {
            this.local.choose?.(s.turnId, zone, this.state.now);
            this.tick();
            return;
        }
        if (this.socket?.status === 'online')
            try {
                await this.socket.gameCommand('penalty.choose', { matchId: s.matchId, turnId: s.turnId, zone });
            }
            catch (e) {
                this.state = { ...this.state, message: (e as Error).message };
                this.publish();
            }
    }
    async action(action: 'return' | 'rematch' | 'leave') {
        const active = this.state.active;
        if (!active)
            return;
        this.controls(neutral());
        if (!this.socket) {
            if (action === 'rematch') {
                await this.startLocal(active.kind);
                return;
            }
            this.clearLocal();
            return;
        }
        if (this.socket.status !== 'online')
            return;
        try {
            await this.socket.gameCommand(action === 'leave' ? 'lobby.leave' : `match.${action}`, action === 'leave' ? {} : { matchId: active.matchId });
            await this.reload();
        }
        catch (e) {
            this.state = { ...this.state, message: (e as Error).message };
            this.publish();
        }
    }
    clearLocal() {
        this.localVehicles.reset();
        this.localStarting = false;
        this.rideHeld = { x: 0, y: 0 };
        this.epoch++;
        this.local?.clear();
        this.local = null;
        this.localKind = null;
        if (!this.socket)
            this.games.clear();
        this.publish(performance.now(), true);
    }
    start() {
        if (!this.timer)
            this.timer = setInterval(() => {
                activityInput.repeat();
                this.tick();
            }, 50);
    }
    dispose() {
        if (this.timer)
            clearInterval(this.timer);
        this.timer = null;
        this.controls(neutral());
        this.local?.clear();
        this.local = null;
        this.epoch++;
        this.listeners.clear();
        this.sceneListeners.clear();
    }
}
export const activities = new ActivityStore();
/** Transport listeners live for the application, independently of any menu or HUD. */
export function mountActivities(world: {
    setActivityState: (state: ActivityState) => void;
    setClosedZones: (zones: any[]) => void;
    stopInput: () => void;
}) {
    let socket: Transport | null = null;
    let wasOnline = false;
    const scene = activities.subscribeScene(() => {
        const state = activities.snapshot();
        world.setActivityState(state);
        activityInput.setOwner(state.inputMode);
        activityInput.block('disconnected', !state.connected);
    });
    const raceInput = activityInput.register('race', input => activities.controls(input));
    const rideInput = activityInput.register('ride', input => activities.rideControls(input));
    const attachment = (e: Event) => {
        world.stopInput();
        activityInput.release();
        socket = (e as CustomEvent).detail;
        wasOnline = socket?.status === 'online';
        void activities.attach(socket);
    };
    const event = (e: Event) => {
        const d = (e as CustomEvent).detail;
        if (!socket || d?.socket !== socket)
            return;
        if (d.type === 'world.snapshot' && !wasOnline) {
            wasOnline = true;
            void activities.reload();
        }
        else
            activities.receive(d.type, d.payload);
    };
    const status = (e: Event) => {
        const d = (e as CustomEvent).detail;
        if (d?.socket !== socket)
            return;
        const online = d.state === 'online';
        if (online === wasOnline)
            return;
        activityInput.block('disconnected', !online);
        if (online && !wasOnline)
            void activities.reload();
        wasOnline = online;
        activities.tick();
    };
    const reset = () => {
        activityInput.release();
        activities.clearLocal();
    };
    let closureVersion = '';
    const closures = activities.subscribeScene(() => {
        const areas = activities.snapshot().areas, version = areas.map(a => `${a.incarnation}:${a.areaId}:${a.generation}:${a.payload.phase}`).join('|');
        if (version === closureVersion)
            return;
        closureVersion = version;
        world.setClosedZones(areas.filter(a => a.payload.phase !== 'OPEN').flatMap(a => {
            const area = activityContract.areas.find(v => v.id === a.areaId);
            return area ? [{ id: area.id, bounds: area.bounds, entrance: area.entrance, label: `${area.kind} · ${a.payload.phase}` }] : [];
        }));
    });
    window.addEventListener('patra-world-attachment', attachment);
    window.addEventListener('patra-world-event', event);
    window.addEventListener('patra-world-status', status);
    window.addEventListener('patra-auth-change', reset);
    window.addEventListener('patra-guest-reset', reset);
    activities.start();
    world.setActivityState(activities.snapshot());
    return () => {
        scene();
        closures();
        raceInput();
        rideInput();
        activities.dispose();
        window.removeEventListener('patra-world-attachment', attachment);
        window.removeEventListener('patra-world-event', event);
        window.removeEventListener('patra-world-status', status);
        window.removeEventListener('patra-auth-change', reset);
        window.removeEventListener('patra-guest-reset', reset);
    };
}
