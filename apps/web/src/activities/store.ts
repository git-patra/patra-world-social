import type {BlasterSnapshot,BlasterControls} from '../features/blaster/local';
import { activityContract } from './contracts';
import type { ActivitySnapshot, AreaSnapshot, VehiclesSnapshot, Vehicle, FootballSnapshot, Football } from './contracts';
import { LocalVehicles, nearbyVehicle } from './vehicles';
import { LocalFootball, nearFootball } from './football';
import {LocalFishing,nearbyFishing,fishingSpawn} from './fishing';
import type {Fishing,FishingSnapshot,FishingAction,FishingView} from './contracts';
import { LocalClaw, nearClaw } from './claw';
import type {Claw,ClawSnapshot,ClawAction} from './contracts';
import { audio } from '../audio/service';
import type { RaceSnapshot, Controls } from '../features/race/local';
import type { PingpongSnapshot } from '../features/pingpong/local';
import type { PublicPenalty } from '../features/penalty/local';
import { activityInput } from './input-router';
export type ActivityKind = 'race' | 'penalty' | 'pingpong' | 'blaster';
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
    snapshot: RaceSnapshot | PublicPenalty | PingpongSnapshot | BlasterSnapshot;
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
    snapshot: (at: number) => RaceSnapshot | PublicPenalty | PingpongSnapshot | BlasterSnapshot | null;
    clear: () => void;
    blast?: (controls:BlasterControls,at:number)=>boolean;
    paddle?: (axis:number,at:number)=>boolean;
    serve?: (rally:number,at:number)=>boolean;
    input?: (controls: Controls, at: number) => boolean;
    choose?: (turn: string, zone: number, at: number) => boolean;
};
export type ActivityState = {
    games: ActivityGame[];
    areas: AreaSnapshot[];
    fishing:FishingView[];
    nearbyFishing:FishingView|null;
    claw: Claw | null;
    nearbyClaw:boolean;
    football: Football | null;
    nearbyFootball: boolean;
    vehicles: Vehicle[];
    nearbyRide: Vehicle | null;
    active: ActivityGame | null;
    spectating: ActivityGame | null;
    inputMode: 'walk' | 'ride' | 'claw' | 'fishing' | ActivityKind;
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
    private localFishing=new LocalFishing();
    private fishingStates=new Map<string,FishingSnapshot>();
    private fishingPending=false;
    private localClaw=new LocalClaw();
    private clawState:ClawSnapshot|null=null;
    private clawHeld={x:0,y:0};
    private clawPending=false;
    private clawSending=false;
    private localFootball = new LocalFootball();
    private footballState: FootballSnapshot | null = null;
    private kickPending = false;
    private penaltyStarting = false;
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
    private gameRetryAt = new Map<string, number>();
    private games = new Map<string, ActivityGame>();
    private areas = new Map<string, AreaSnapshot>();
    private lobbies: ActivityLobby[] = [];
    private listeners = new Set<() => void>();
    private sceneListeners = new Set<() => void>();
    private held = neutral();
    private sending = false;
    private lastNotify = -Infinity;
    private timer: ReturnType<typeof setInterval> | null = null;
    private state: ActivityState = { fishing:this.localFishing.spots,nearbyFishing:null,claw:this.localClaw.claw,nearbyClaw:false,games: [], areas: [], football: this.localFootball.ball, nearbyFootball: false, vehicles: this.localVehicles.vehicles, nearbyRide: null, lobbies: [], active: null, spectating: null, inputMode: 'walk', selfId: 'guest', online: false, connected: true, now: 0, message: '' };
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
        const spectating = active ? null : games.find(g => {
            if (g.kind !== 'penalty' && g.kind !== 'pingpong' && g.kind !== 'blaster')
                return false;
            const area = this.areas.get(g.kind);
            if (area && (area.payload.phase === 'OPEN' || area.payload.matchId !== g.matchId))
                return false;
            return activityContract.areas.find(a => a.id === g.kind)?.spectatorStrips.some(b => this.position.x >= b.x && this.position.x < b.x + b.width && this.position.y >= b.y && this.position.y < b.y + b.height);
        }) ?? null;
        const area = active ? this.areas.get(active.kind) : undefined, gameMode = active && (!this.socket || !area || area.payload.phase !== 'OPEN' && area.payload.matchId === active.matchId) ? active.kind : 'walk';
        const vehicles = this.socket ? this.vehicleState?.payload.vehicles ?? [] : this.localVehicles.vehicles;
        const raceClosed = this.socket ? this.areas.get('race')?.payload.phase !== 'OPEN' : this.localStarting || this.localKind === 'race';
        const penaltyClosed = this.socket ? this.areas.get('penalty')?.payload.phase !== 'OPEN' : this.penaltyStarting || this.localKind === 'penalty';
        const ball = this.socket ? this.footballState?.payload.ball ?? null : this.localFootball.ball;
        const football = ball && penaltyClosed ? { ...ball, visible: false } : ball;
        const claw=this.socket?this.clawState?.payload.claw??null:this.localClaw.claw;
        const fishing=this.socket?[...this.fishingStates.values()].map(s=>({id:s.areaId,generation:s.generation,fishing:this.socket?.status==='online'?s.payload.fishing:fishingSpawn(s.payload.fishing.castId)})):this.localFishing.spots;
        const ownFish=fishing.find(s=>s.fishing.ownerId===selfId);
        const inputMode = gameMode !== 'walk' ? gameMode : ownFish ? 'fishing' : claw?.operatorId===selfId ? 'claw' : !raceClosed && vehicles.some(v => v.riderId === selfId) ? 'ride' : 'walk';
        if(this.state.inputMode==='claw'&&inputMode!=='claw')this.clawHeld={x:0,y:0};
        if (this.state.inputMode === 'ride' && inputMode !== 'ride')
            this.rideHeld = { x: 0, y: 0 };
        const nearbyRide = gameMode === 'walk' && !raceClosed ? nearbyVehicle(vehicles, this.position) ?? null : null;
        this.state = { ...this.state,fishing,nearbyFishing:!active?nearbyFishing(fishing,this.position)??null:null,claw,nearbyClaw:!active&&nearClaw(this.position),football, nearbyFootball: inputMode === 'walk' && nearFootball(football, this.position), vehicles, nearbyRide, games, areas: [...this.areas.values()], lobbies: this.lobbies, active, spectating, inputMode, selfId, online: Boolean(this.socket), connected: !this.socket || this.socket.status === 'online', now: at };
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
        this.blasterHeld={dx:0,dy:0,aim:0,fire:false};this.blasterLast=-Infinity;audio.stopMode('blaster');
        this.paddleAxis = 0;
        audio.stopMode('pingpong');
        this.epoch++;
        this.resyncRequest++;
        this.local?.clear();
        this.local = null;
        this.localKind = null;
        this.socket = socket;
        this.games.clear();
        this.localVehicles.reset();
        this.localFootball.reset();
        this.localClaw.reset();
        this.localFishing.reset();this.fishingStates.clear();this.fishingPending=false;audio.stopMode("fishing");
        this.clawState=null;
        this.clawHeld={x:0,y:0};
        this.clawPending=this.clawSending=false;
        this.footballState = null;
        this.kickPending = false;
        this.penaltyStarting = false;
        this.vehicleState = null;
        this.rideHeld = { x: 0, y: 0 };
        this.rideSending = false;
        this.ridePending = false;
        this.localStarting = false;
        this.areas.clear();
        this.areaRevisions.clear();
        this.retiredIncarnations.clear();
        this.gameReads.clear();
        this.gameRetryAt.clear();
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
            await this.loadGames(true);
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
        for (const id of this.gameRetryAt.keys())
            if (!this.lobbies.some(l => l.matchId === id))
                this.gameRetryAt.delete(id);
        for (const [id] of this.games)
            if (!this.lobbies.some(l => l.matchId === id))
                this.games.delete(id);
        this.publish();
    }
    private async loadGames(force = false) {
        const socket = this.socket, epoch = this.epoch, resync = this.resyncRequest;
        if (!socket || socket.status !== 'online')
            return;
        const lobbies = this.lobbies.filter(l => (l.gameType === 'race' || l.gameType === 'penalty' || l.gameType === 'pingpong' || l.gameType === 'blaster') && l.matchId);
        await Promise.all(lobbies.map(async (lobby) => {
            if (!force && (this.games.has(lobby.matchId!) || this.gameReads.has(lobby.matchId!) || performance.now() < (this.gameRetryAt.get(lobby.matchId!) ?? 0)))
                return;
            const read = ++this.readSerial;
            this.gameReads.set(lobby.matchId!, read);
            try {
                const snapshot = await socket.gameCommand(`${lobby.gameType}.snapshot`, { matchId: lobby.matchId });
                if (epoch === this.epoch && socket === this.socket && resync === this.resyncRequest && this.gameReads.get(lobby.matchId!) === read) {
                    this.gameRetryAt.delete(lobby.matchId!);
                    this.acceptGame(lobby.gameType as ActivityKind, snapshot as RaceSnapshot | PublicPenalty | PingpongSnapshot | BlasterSnapshot);
                }
            }
            catch {
                if (epoch === this.epoch && socket === this.socket && resync === this.resyncRequest && this.gameReads.get(lobby.matchId!) === read && this.lobbies.some(l => l.matchId === lobby.matchId))
                    this.gameRetryAt.set(lobby.matchId!, performance.now() + 1000);
            }
            finally {
                if (this.gameReads.get(lobby.matchId!) === read)
                    this.gameReads.delete(lobby.matchId!);
            }
        }));
    }
    private acceptGame(kind: ActivityKind, snapshot: RaceSnapshot | PublicPenalty | PingpongSnapshot | BlasterSnapshot, live = false) {
        // Online visual projection uses bounded [id,x,y,owner] projectile tuples.
        // Local simulation retains its full-precision motion and expiry fields.
        if(kind==='blaster'&&'projectiles' in snapshot){
            const shots=snapshot.projectiles as unknown[];
            if(shots.some(Array.isArray)){
                if(shots.length>48||shots.some(p=>!Array.isArray(p)||p.length!==4||!p.every(Number.isFinite)))return;
                snapshot={...snapshot,projectiles:shots.map(p=>{const [id,x,y,owner]=p as number[];return{id,x,y,owner,vx:0,vy:0,ttl:0}})};
            }
        }
        if (!snapshot?.matchId || !this.lobbies.some(l => l.matchId === snapshot.matchId && l.gameType === kind))
            return;
        const previous = this.games.get(snapshot.matchId);
        if ((kind === 'race' || kind === 'pingpong' || kind === 'blaster') && previous && 'frame' in previous.snapshot && 'frame' in snapshot && (snapshot.frame < previous.snapshot.frame || (kind === 'pingpong' || kind === 'blaster') && snapshot.frame === previous.snapshot.frame))
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
        if(kind==='blaster'&&live&&previous)this.blasterCue(previous.snapshot as BlasterSnapshot,snapshot as BlasterSnapshot,this.socket?.serverNow()??this.state.now);
        if (kind === 'pingpong' && live && previous) this.pingpongCue(previous.snapshot as PingpongSnapshot,snapshot as PingpongSnapshot,this.socket?.serverNow()??this.state.now);
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
            if(payload.kind==='fishing'){const previous=this.fishingStates.get(payload.areaId);if(previous&&payload.incarnation!==previous.incarnation)return;this.mergeFishing(payload);const age=this.socket.serverNow()-payload.serverTimeMs;if(type==='activity.lifecycle'&&previous&&this.fishingStates.get(payload.areaId)!==previous&&age>=-250&&age<=1000){const f=payload.payload.fishing;if(f.castId>previous.payload.fishing.castId||f.phase==='result'&&previous.payload.fishing.phase!=='result')audio.play({kind:f.phase==='result'?'catch':'cast',mode:'fishing',eventId:`fishing:${payload.incarnation}:${payload.areaId}:${payload.generation}`,distance:Math.hypot(this.position.x-53,this.position.y-31)})};this.publish(this.socket.serverNow());return}
            if(payload.kind==='claw') {
                const previous=this.clawState;
                if(previous&&payload.incarnation!==previous.incarnation)return;
                this.mergeClaw(payload);
                const age=this.socket.serverNow()-payload.serverTimeMs;
                if(type==='activity.lifecycle'&&previous&&this.clawState!==previous&&payload.payload.claw.dropId>previous.payload.claw.dropId&&age>=-250&&age<=1000)
                    audio.play({kind:'grab',mode:'claw',eventId:`claw:${payload.incarnation}:${payload.payload.claw.dropId}`,distance:Math.hypot(this.position.x-41,this.position.y-22)});
                this.publish(this.socket.serverNow());return;
            }
            if (payload.kind === 'football') {
                const previous = this.footballState;
                if (previous && payload.incarnation !== previous.incarnation)
                    return;
                this.mergeFootball(payload);
                const age = this.socket.serverNow() - payload.serverTimeMs;
                if (age >= -250 && age <= 1000 && type === 'activity.lifecycle' && previous && this.footballState !== previous && payload.generation > previous.generation && payload.payload.ball.visible && Math.hypot(payload.payload.ball.vx, payload.payload.ball.vy) > 0)
                    audio.play({ kind: 'kick', mode: 'football', eventId: `kick:${payload.incarnation}:${payload.generation}`, distance: Math.hypot(payload.payload.ball.x - this.position.x, payload.payload.ball.y - this.position.y) });
                this.publish(this.socket.serverNow());
                return;
            }
            if (payload.kind === 'vehicles') {
                if (this.vehicleState && payload.incarnation !== this.vehicleState.incarnation)
                    return;
                const previous = this.vehicleState;
                this.mergeVehicles(payload);
                const oldRide = previous?.payload.vehicles.find(v => v.riderId === this.socket?.selfId);
                const ownRide = this.vehicleState?.payload.vehicles.find(v => v.riderId === this.socket?.selfId);
                const age = this.socket.serverNow() - payload.serverTimeMs;
                if (age >= -250 && age <= 1000 && type === 'activity.lifecycle' && previous && this.vehicleState !== previous && ownRide && (!oldRide || oldRide.id !== ownRide.id || oldRide.generation !== ownRide.generation))
                    audio.play({ kind: 'ride', mode: 'ride', eventId: `ride:${payload.incarnation}:${ownRide.id}:${ownRide.generation}` });
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
        if(type === 'blaster.snapshot' && payload?.players)this.acceptGame('blaster',payload,true);
        if (type === 'pingpong.snapshot' && payload?.actors)
            this.acceptGame('pingpong', payload, true);
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
            else if(area.kind==='fishing')this.mergeFishing(area,revisions);
            else if(area.kind==='claw')this.mergeClaw(area,revisions);
            else if (area.kind === 'football')
                this.mergeFootball(area, revisions);
            else if (area.kind === 'vehicles')
                this.mergeVehicles(area, revisions);
        }
        this.publish(this.socket?.serverNow());
    }
    private mergeFishing(snapshot:FishingSnapshot,revisions?:Map<string,number>){
        const key=snapshot.areaId,previous=this.fishingStates.get(key);
        if(previous){if(snapshot.incarnation===previous.incarnation){if(snapshot.generation<previous.generation||snapshot.generation===previous.generation&&snapshot.tick<=previous.tick)return}else{if(this.retiredIncarnations.get(key)?.has(snapshot.incarnation)||revisions&&(revisions.get(key)??0)!==(this.areaRevisions.get(key)??0))return;const retired=this.retiredIncarnations.get(key)??new Set<string>();retired.add(previous.incarnation);this.retiredIncarnations.set(key,retired)}}
        this.fishingStates.set(key,snapshot);this.areaRevisions.set(key,(this.areaRevisions.get(key)??0)+1);
    }
    interactFishing(position=this.position){
        this.position={...position};
        if(this.state.inputMode==='fishing'){this.reelFishing();return true}
        if(this.state.inputMode!=='walk'||this.state.active||this.localStarting||!this.state.connected||this.fishingPending)return false;
        const spot=nearbyFishing(this.state.fishing,position);if(!spot||spot.fishing.ownerId)return false;
        void this.fishingAction('fishing.cast',spot);return true;
    }
    reelFishing(){const own=this.state.fishing.find(s=>s.fishing.ownerId===this.state.selfId);if(!own||!['waiting','bite'].includes(own.fishing.phase)||!this.state.connected||this.fishingPending)return false;void this.fishingAction('fishing.reel',own);return true}
    leaveFishing(){const own=this.state.fishing.find(s=>s.fishing.ownerId===this.state.selfId);if(own)void this.fishingAction('fishing.leave',own)}
    fishingControls(input:{x:number;y:number}){if(input.x||input.y)this.leaveFishing()}
    private async fishingAction(action:FishingAction,spot:FishingView){
        if(!this.socket){if(action==='fishing.cast')this.localFishing.cast(spot.id,this.position,this.state.now);else if(action==='fishing.reel')this.localFishing.reel(this.state.now);else this.localFishing.reset();if(action!=='fishing.leave')audio.play({kind:action==='fishing.cast'?'cast':'catch',mode:'fishing',eventId:`fishing:local:${spot.id}:${spot.generation}:${action}`});this.publish(undefined,true);return}
        const s=this.fishingStates.get(spot.id);if(!s||this.socket.status!=='online'||this.fishingPending)return;
        this.fishingPending=true;const epoch=this.epoch;
        try{await this.socket.gameCommand('activity.fishing',{incarnation:s.incarnation,areaGeneration:0,generation:s.generation,spotId:spot.id,action});if(epoch===this.epoch)await this.reload()}
        catch(e){if(epoch===this.epoch){this.state={...this.state,message:(e as Error).message};this.publish(undefined,true)}}
        finally{if(epoch===this.epoch)this.fishingPending=false}
    }
    private mergeClaw(snapshot: ClawSnapshot, revisions?: Map<string, number>) {
        const previous = this.clawState, key = 'claw';
        if (previous) {
            if (snapshot.incarnation === previous.incarnation) {
                if (snapshot.generation < previous.generation || snapshot.generation === previous.generation && snapshot.tick <= previous.tick)
                    return;
            } else {
                if (this.retiredIncarnations.get(key)?.has(snapshot.incarnation) || revisions && (revisions.get(key) ?? 0) !== (this.areaRevisions.get(key) ?? 0))
                    return;
                const retired = this.retiredIncarnations.get(key) ?? new Set<string>();
                retired.add(previous.incarnation);
                this.retiredIncarnations.set(key, retired);
            }
        }
        this.clawState = snapshot;
        this.areaRevisions.set(key, (this.areaRevisions.get(key) ?? 0) + 1);
    }
    private mergeFootball(snapshot: FootballSnapshot, revisions?: Map<string, number>) {
        const previous = this.footballState, key = 'football';
        if (previous) {
            if (snapshot.incarnation === previous.incarnation) {
                if (snapshot.generation < previous.generation || snapshot.generation === previous.generation && snapshot.tick <= previous.tick)
                    return;
            } else {
                if (this.retiredIncarnations.get(key)?.has(snapshot.incarnation) || revisions && (revisions.get(key) ?? 0) !== (this.areaRevisions.get(key) ?? 0))
                    return;
                const retired = this.retiredIncarnations.get(key) ?? new Set<string>();
                retired.add(previous.incarnation);
                this.retiredIncarnations.set(key, retired);
            }
        }
        this.footballState = snapshot;
        this.areaRevisions.set(key, (this.areaRevisions.get(key) ?? 0) + 1);
    }
    interactClaw(position=this.position) {
        this.position={...position};
        if(this.state.inputMode==='claw'){void this.clawAction('claw.leave');return true}
        if(this.state.inputMode!=='walk'||this.state.active||this.localStarting||!this.state.connected||this.clawPending||!nearClaw(position)||!this.state.claw||this.state.claw.operatorId)return false;
        void this.clawAction('claw.enter');return true;
    }
    dropClaw() {
        if(this.state.inputMode!=='claw'||!this.state.connected||this.state.claw?.phase!=='aiming'||this.clawPending)return false;
        void this.clawAction('claw.drop');return true;
    }
    private async clawAction(action:ClawAction) {
        this.clawHeld={x:0,y:0};
        if(!this.socket) {
            if(action==='claw.enter')this.localClaw.enter(this.position,this.state.now);
            else if(action==='claw.leave')this.localClaw.reset(this.state.now);
            else if(action==='claw.drop'&&this.localClaw.drop(this.state.now))audio.play({kind:'grab',mode:'claw',eventId:`claw:local:${this.localClaw.claw.dropId}`});
            this.publish(undefined,true);return;
        }
        if(!this.clawState||this.socket.status!=='online'||this.clawPending)return;
        this.clawPending=true;const epoch=this.epoch;
        try{await this.socket.gameCommand('activity.claw',this.clawIntent(action));if(epoch===this.epoch)await this.reload()}
        catch(e){if(epoch===this.epoch){this.state={...this.state,message:(e as Error).message};this.publish(undefined,true)}}
        finally{if(epoch===this.epoch)this.clawPending=false}
    }
    private clawIntent(action:ClawAction):Record<string,unknown> {return {incarnation:this.clawState!.incarnation,areaGeneration:0,generation:this.clawState!.generation,machineId:'claw-1',action,...(action==='claw.move'?{dx:this.clawHeld.x,dy:this.clawHeld.y}:{})}}
    clawControls(input:{x:number;y:number;steer?:number;throttle?:number;brake?:number}) {
        this.clawHeld={x:input.x||input.steer||0,y:input.y||(input.brake??0)-(input.throttle??0)};
        if(!this.clawHeld.x&&!this.clawHeld.y)this.sendClawInput();
        else if(!this.socket)this.localClaw.controls(this.clawHeld.x,this.clawHeld.y,this.state.now);
    }
    private sendClawInput() {
        if(this.state.inputMode!=='claw'||this.clawPending||this.state.claw?.phase!=='aiming')return;
        if(!this.socket){this.localClaw.controls(this.clawHeld.x,this.clawHeld.y,this.state.now);return}
        if(!this.clawState||this.socket.status!=='online'||this.clawSending)return;
        this.clawSending=true;const epoch=this.epoch;
        void this.socket.gameCommand('activity.claw',this.clawIntent('claw.move')).catch(()=>{if(epoch===this.epoch)this.clawHeld={x:0,y:0}}).finally(()=>{if(epoch===this.epoch)this.clawSending=false});
    }
    interactFootball(position = this.position) {
        this.position = { ...position };
        const ball = this.state.football;
        if (this.state.inputMode !== 'walk' || !this.state.connected || this.kickPending || !nearFootball(ball, position) || !ball)
            return false;
        const x = ball.x - position.x, y = ball.y - position.y, length = Math.hypot(x, y);
        const dx = length > .05 ? x / length : 0, dy = length > .05 ? y / length : -1;
        if (!this.socket) {
            if (!this.localFootball.kick(position, dx, dy, this.state.now))
                return false;
            audio.play({ kind: 'kick', mode: 'football', eventId: `kick:local:${this.localFootball.revision}` });
            this.publish(undefined, true);
            return true;
        }
        if (!this.footballState || this.areas.get('penalty')?.payload.phase !== 'OPEN')
            return false;
        this.kickPending = true;
        const epoch = this.epoch, socket = this.socket;
        const intent = {
            incarnation: this.footballState.incarnation,
            areaGeneration: this.areas.get('penalty')?.generation ?? 0,
            generation: this.footballState.generation,
            action: 'kick', dx, dy
        };
        void socket.gameCommand('activity.kick', intent).then(() => {
            if (epoch === this.epoch)
                return this.reload();
        }).catch(e => {
            if (epoch === this.epoch) {
                this.state = { ...this.state, message: (e as Error).message };
                this.publish(undefined, true);
            }
        }).finally(() => {
            if (epoch === this.epoch)
                this.kickPending = false;
        });
        return true;
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
        if(!this.socket&&this.state.inputMode==='fishing')this.localFishing.reset();
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
        this.localFootball.reset(kind !== 'penalty', at);
        this.localClaw.reset(at);
        this.localFishing.reset();audio.stopMode("fishing");
        this.clawHeld={x:0,y:0};
        this.penaltyStarting = kind === 'penalty';
        this.rideHeld = { x: 0, y: 0 };
        this.localStarting = kind === 'race';
        this.publish(at, true);
        const epoch = ++this.epoch;
        this.local?.clear();
        this.games.clear();
        this.local = null;
        this.localKind = null;
        const session = kind === 'blaster' ? (await import('../features/blaster/session')).guestBlaster : kind === 'pingpong' ? (await import('../features/pingpong/session')).guestPingpong : kind === 'race' ? (await import('../features/race/session')).guestRace : (await import('../features/penalty/session')).guestSession;
        if (epoch !== this.epoch || this.socket)
            return;
        const snapshot = kind === 'blaster' ? (session as typeof import('../features/blaster/session').guestBlaster).start(at,seed,id) : kind === 'pingpong' ? (session as typeof import('../features/pingpong/session').guestPingpong).start(at, seed, id) : kind === 'race' ? (session as typeof import('../features/race/session').guestRace).start(at, seed, id) : (session as typeof import('../features/penalty/session').guestSession).start(at, undefined, id);
        this.localStarting = false;
        this.penaltyStarting = false;
        this.local = session;
        this.localKind = kind;
        this.games.set(snapshot.matchId, { kind, matchId: snapshot.matchId, snapshot });
        this.publish(at, true);
    }
    private pingpongCue(old:PingpongSnapshot,next:PingpongSnapshot,at:number) {
        if(next.frame<=old.frame||at-next.serverTimeMs>1000||at<next.serverTimeMs-250)return;
        const kind=next.phase==='results'&&old.phase!=='results'?'finish':next.scores.some((n,i)=>n!==old.scores[i])?'goal':next.phase==='rally'&&old.phase==='rally'&&next.ball.vx*old.ball.vx<0?'paddle':null;
        if(kind)audio.play({kind,mode:'pingpong',eventId:`pingpong:${next.matchId}:${next.frame}:${kind}`,distance:Math.hypot(this.position.x-45,this.position.y-23)});
    }
    private blasterHeld:BlasterControls={dx:0,dy:0,aim:0,fire:false};
    private blasterLast=-Infinity;
    blast(controls:BlasterControls){this.blasterHeld={...controls};}
    private sendBlaster(){
        const active=this.state.active,at=this.state.now;
        if(active?.kind!=='blaster'||this.state.inputMode!=='blaster'||active.snapshot.phase==='results'||!this.state.connected||at-this.blasterLast<50)return;
        if(this.local){this.blasterLast=at;this.local.blast?.(this.blasterHeld,at);return}
        if(!this.socket||this.socket.status!=='online'||this.sending)return;
        this.blasterLast=at;this.sending=true;const epoch=this.epoch;
        void this.socket.gameCommand('blaster.input',{matchId:active.matchId,...this.blasterHeld}).catch(()=>{}).finally(()=>{if(epoch===this.epoch)this.sending=false});
    }
    private blasterCue(old:BlasterSnapshot,next:BlasterSnapshot,at:number){
        if(next.frame<=old.frame||at-next.serverTimeMs>1000||at<next.serverTimeMs-250)return;
        const kind=next.phase==='results'&&old.phase!=='results'?'finish':next.players.some((p,i)=>p.hitAt> (old.players[i]?.hitAt??0))?'hit':next.projectiles.some(p=>p.owner>=0&&!old.projectiles.some(o=>o.id===p.id))?'fire':null;
        if(kind)audio.play({kind,mode:'blaster',eventId:`blaster:${next.matchId}:${next.frame}:${kind}`,distance:Math.hypot(this.position.x-43,this.position.y-30)});
    }
    private paddleAxis = 0;
    paddle(axis:number) {
        this.paddleAxis = Math.max(-1,Math.min(1,axis));
        if (this.local || !this.paddleAxis) this.sendPaddle();
    }
    private sendPaddle() {
        const active=this.state.active;
        if(active?.kind!=='pingpong'||this.state.inputMode!=='pingpong'||active.snapshot.phase==='results')return;
        if(this.local){this.local.paddle?.(this.paddleAxis,this.state.now);return}
        if(!this.socket||this.socket.status!=='online'||this.sending)return;
        this.sending=true;const epoch=this.epoch;
        void this.socket.gameCommand('pingpong.input',{matchId:active.matchId,axis:this.paddleAxis}).catch(()=>{}).finally(()=>{if(epoch===this.epoch)this.sending=false});
    }
    async serve() {
        const active=this.state.active;
        if(active?.kind!=='pingpong'||this.state.inputMode!=='pingpong'||active.snapshot.phase!=='serve'||!this.state.connected)return;
        const s=active.snapshot as PingpongSnapshot;
        if(this.local){this.local.serve?.(s.rallyId,this.state.now);this.tick();return}
        try{await this.socket?.gameCommand('pingpong.serve',{matchId:s.matchId,rallyId:s.rallyId})}catch(e){this.state={...this.state,message:(e as Error).message};this.publish()}
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
        if (!this.socket) {
            this.localFishing.tick(at);
            this.localClaw.tick(at);
            this.localVehicles.tick(at);
            this.localFootball.tick(at);
        }
        this.sendClawInput();
        this.sendRideInput();
        this.sendBlaster();
        this.sendPaddle();
        this.sendInput();
        if (this.local && this.localKind) {
            const snapshot = this.local.snapshot(at);
            if (snapshot) {
                const previous=this.games.get(snapshot.matchId);
                if(this.localKind==='blaster'&&previous)this.blasterCue(previous.snapshot as BlasterSnapshot,snapshot as BlasterSnapshot,at);
                if(this.localKind==='pingpong'&&previous)this.pingpongCue(previous.snapshot as PingpongSnapshot,snapshot as PingpongSnapshot,at);
                this.games.set(snapshot.matchId, { kind: this.localKind, matchId: snapshot.matchId, snapshot });
            }
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
        this.localFishing.reset();audio.stopMode("fishing");
        this.clawHeld={x:0,y:0};
        this.sendClawInput();
        this.localClaw.reset();
        audio.stopMode('claw');
        this.localVehicles.reset();
        this.localFootball.reset();
        this.penaltyStarting = false;
        this.localStarting = false;
        this.rideHeld = { x: 0, y: 0 };
        this.blasterHeld={dx:0,dy:0,aim:0,fire:false};this.blasterLast=-Infinity;audio.stopMode('blaster');
        this.paddleAxis = 0;
        audio.stopMode('pingpong');
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
        this.localFishing.reset();this.fishingStates.clear();audio.stopMode("fishing");
        this.clawHeld={x:0,y:0};this.sendClawInput();audio.stopMode('claw');
        if (this.timer)
            clearInterval(this.timer);
        this.timer = null;
        this.controls(neutral());
        this.local?.clear();
        this.local = null;
        this.blasterHeld={dx:0,dy:0,aim:0,fire:false};this.blasterLast=-Infinity;audio.stopMode('blaster');
        this.paddleAxis = 0;
        audio.stopMode('pingpong');
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
    const blasterInput=activityInput.register('blaster',input=>activities.blast({dx:input.x,dy:input.y,aim:input.aim??0,fire:input.fire??false}));
    const pingpongInput=activityInput.register('pingpong',input=>activities.paddle(input.steer));
    const fishingInput=activityInput.register('fishing',input=>activities.fishingControls(input));
    const clawInput=activityInput.register('claw',input=>activities.clawControls(input));
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
    const drop=(e:KeyboardEvent)=>{if(e.code==='Space'&&!e.repeat&&!e.ctrlKey&&!e.altKey&&!e.metaKey&&activityInput.mode()==='claw'&&!(e.target instanceof Element&&e.target.closest('button,input,textarea,select,[contenteditable="true"]'))){e.preventDefault();activities.dropClaw()}else if(e.code==='Space'&&!e.repeat&&activityInput.mode()==='fishing'&&!(e.target instanceof Element&&e.target.closest('button,input,textarea,select,[contenteditable="true"]'))){e.preventDefault();activities.reelFishing()}};
    window.addEventListener('keydown',drop);
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
        window.removeEventListener('keydown',drop);
        scene();
        closures();
        raceInput();
        rideInput();
        blasterInput();
        pingpongInput();
        clawInput();
        fishingInput();
        activities.dispose();
        window.removeEventListener('patra-world-attachment', attachment);
        window.removeEventListener('patra-world-event', event);
        window.removeEventListener('patra-world-status', status);
        window.removeEventListener('patra-auth-change', reset);
        window.removeEventListener('patra-guest-reset', reset);
    };
}
