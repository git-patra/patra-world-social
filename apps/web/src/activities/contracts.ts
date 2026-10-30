import track from '../../../../contracts/race-track.v1.json';
import contract from '../../../../contracts/activities.v1.json';
export type AreaPhase = 'OPEN' | 'CLEARING' | 'RESERVED' | 'ACTIVE' | 'RESETTING';
export interface AreaSnapshot {
    worldId: string;
    areaId: string;
    incarnation: string;
    generation: number;
    serverTimeMs: number;
    tick: number;
    kind: 'area';
    payload: {
        phase: AreaPhase;
        matchId: string;
        startsAt: number;
        clearAt: number;
        resetAt: number;
    };
}
export type Vehicle = {
    id: string;
    kind: 'bicycle' | 'car';
    generation: number;
    riderId: string;
    x: number;
    y: number;
    heading: number;
};
export type VehiclesSnapshot = Omit<AreaSnapshot, 'kind' | 'payload'> & {
    kind: 'vehicles';
    payload: {
        vehicles: Vehicle[];
    };
};
export type Football = { x: number; y: number; vx: number; vy: number; visible: boolean };
export type FootballSnapshot = Omit<AreaSnapshot, 'kind' | 'payload'> & {
    kind: 'football';
    payload: { ball: Football };
};
export type KickIntent = {
    incarnation: string;
    areaGeneration: number;
    generation: number;
    action: 'kick';
    dx: number;
    dy: number;
};
export const footballRules = { kickRadius: 1.5, radius: .15, speed: 8, cooldownMs: 400, idleMs: 15000, friction: .9 };
export const footballInside = (x: number, y: number) => {
    const b = contract.areas.find(a => a.id === 'penalty')!.bounds, r = footballRules.radius;
    return x >= b.x + r && x <= b.x + b.width - r && y >= b.y + r && y <= b.y + b.height - r;
};
export type ActivitySnapshot = AreaSnapshot | VehiclesSnapshot | FootballSnapshot | ClawSnapshot | FishingSnapshot;
export type RideIntent = {
    incarnation: string;
    areaGeneration: number;
    vehicleId: string;
    generation: number;
    action: 'mount' | 'dismount';
} | {
    incarnation: string;
    areaGeneration: number;
    vehicleId: string;
    generation: number;
    action: 'drive';
    dx: number;
    dy: number;
};
export const rideDriveable = (x: number, y: number) => {
    const t = contract.areas.find(a => a.id === 'race')!.sceneTransform, o = track.outer, i = track.inner, r = 512;
    x = ((x + .5) * 40 - t.offsetX) / t.scale;
    y = ((y + .5) * 40 - 320 - t.offsetY) / t.scale;
    return x >= o.x + r && y >= o.y + r && x <= o.x + o.width - r && y <= o.y + o.height - r && !(x > i.x - r && x < i.x + i.width + r && y > i.y - r && y < i.y + i.height + r);
};
export function validateRideIntent(raw: unknown): RideIntent {
    const p = raw as Record<string, unknown>, drive = p?.action === 'drive';
    if (!keys(raw, ['incarnation', 'areaGeneration', 'vehicleId', 'generation', 'action', ...(drive ? ['dx', 'dy'] : [])]) || !text(p.incarnation) || !int(p.areaGeneration) || !int(p.generation) || Number(p.generation) < 1 || !contract.vehicleSlots.some(v => v.id === p.vehicleId) || !['mount', 'dismount', 'drive'].includes(String(p.action)) || drive && (!number(p.dx, -1, 1) || !number(p.dy, -1, 1)))
        throw Error('Invalid ride intent');
    return structuredClone(raw) as RideIntent;
}
export function validateKickIntent(raw: unknown): KickIntent {
    const p = raw as Record<string, unknown>;
    if (!keys(raw, ['incarnation', 'areaGeneration', 'generation', 'action', 'dx', 'dy']) || !text(p.incarnation) || !int(p.areaGeneration) || !int(p.generation) || Number(p.generation) < 1 || p.action !== 'kick' || !number(p.dx, -1, 1) || !number(p.dy, -1, 1) || p.dx === 0 && p.dy === 0)
        throw Error('Invalid kick intent');
    return structuredClone(raw) as KickIntent;
}
export type ActivityIntent = {
    type: 'activity.claw';
    payload: ClawIntent;
} | {
    type: 'activity.kick';
    payload: KickIntent;
} | {
    type: 'activity.snapshot';
    payload: Record<string, never>;
} | {
    type: 'activity.ride';
    payload: RideIntent;
};
export const activityContract = contract;
const keys = (value: unknown, expected: string[]): value is Record<string, unknown> => !!value && typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === expected.length && Object.keys(value).every(k => expected.includes(k));
const number = (v: unknown, min: number, max: number) => typeof v === 'number' && Number.isFinite(v) && v >= min && v <= max;
const int = (v: unknown) => Number.isSafeInteger(v) && Number(v) >= 0;
const text = (v: unknown) => typeof v === 'string' && v.length > 0 && v.length <= 128;
export function validateActivitySnapshot(raw: unknown, worldId?: string): ActivitySnapshot {
    if (!keys(raw, ['worldId', 'areaId', 'incarnation', 'generation', 'serverTimeMs', 'tick', 'kind', 'payload']) || !text(raw.worldId) || (worldId !== undefined && raw.worldId !== worldId) || !text(raw.incarnation) || !contract.areas.some(a => a.id === raw.areaId) && !(raw.kind === 'claw' && raw.areaId === 'claw-1') && !(raw.kind === 'fishing' && contract.fishingSpots.some(s=>s.id===raw.areaId)) || !['area', 'vehicles', 'football', 'claw', 'fishing'].includes(String(raw.kind)) || ![raw.generation, raw.serverTimeMs, raw.tick].every(int))
        throw Error('Invalid activity snapshot');
    if(raw.kind==='fishing'){if(!contract.fishingSpots.some(s=>s.id===raw.areaId)||Number(raw.generation)<1||!keys(raw.payload,['fishing'])||!validFishing(raw.payload.fishing))throw Error('Invalid fishing snapshot');return structuredClone(raw) as FishingSnapshot;}
    if (raw.kind === 'claw') {
        if (Number(raw.generation)<1 || !keys(raw.payload,['claw']) || !validClaw(raw.payload.claw)) throw Error('Invalid claw snapshot');
        return structuredClone(raw) as ClawSnapshot;
    }
    if (raw.kind === 'football') {
        if (raw.areaId !== 'penalty' || Number(raw.generation) < 1 || !keys(raw.payload, ['ball']))
            throw Error('Invalid football snapshot');
        const b = raw.payload.ball;
        if (!keys(b, ['x', 'y', 'vx', 'vy', 'visible']) || !number(b.x, 0, 63) || !number(b.y, 0, 63) || !footballInside(Number(b.x), Number(b.y)) || !number(b.vx, -8, 8) || !number(b.vy, -8, 8) || Math.hypot(Number(b.vx), Number(b.vy)) > 8.000000001 || typeof b.visible !== 'boolean' || !b.visible && (b.vx !== 0 || b.vy !== 0))
            throw Error('Invalid football');
        return structuredClone(raw) as FootballSnapshot;
    }
    if (raw.kind === 'vehicles') {
        if (raw.areaId !== 'race' || Number(raw.generation) < 1 || !keys(raw.payload, ['vehicles']) || !Array.isArray(raw.payload.vehicles) || raw.payload.vehicles.length !== 4)
            throw Error('Invalid vehicle pool');
        const seen = new Set(), riders = new Set();
        for (const v of raw.payload.vehicles) {
            if (!keys(v, ['id', 'kind', 'generation', 'riderId', 'x', 'y', 'heading']) || !contract.vehicleSlots.some(slot => slot.id === v.id && slot.kind === v.kind) || seen.has(v.id) || !int(v.generation) || Number(v.generation) < 1 || typeof v.riderId !== 'string' || v.riderId.length > 128 || v.riderId !== '' && riders.has(v.riderId) || !number(v.x, 0, 63) || !number(v.y, 0, 63) || !number(v.heading, -180, 180) || !rideDriveable(Number(v.x), Number(v.y)))
                throw Error('Invalid vehicle');
            seen.add(v.id);
            riders.add(v.riderId);
        }
        return structuredClone(raw) as VehiclesSnapshot;
    }
    if (!keys(raw.payload, ['phase', 'matchId', 'startsAt', 'clearAt', 'resetAt']))
        throw Error('Invalid area payload');
    const p = raw.payload;
    if (!['OPEN', 'CLEARING', 'RESERVED', 'ACTIVE', 'RESETTING'].includes(String(p.phase)) || typeof p.matchId !== 'string' || p.matchId.length > 128 || ![p.startsAt, p.clearAt, p.resetAt].every(int) || (p.phase !== 'OPEN' && (!text(p.matchId) || raw.generation === 0)) || (p.phase === 'OPEN' && p.matchId !== '') || (p.phase === 'CLEARING' && p.clearAt === 0) || (p.phase === 'ACTIVE' && p.startsAt === 0) || (p.phase === 'RESETTING' && p.resetAt === 0))
        throw Error('Invalid area phase');
    return structuredClone(raw) as unknown as ActivitySnapshot;
}

export const clawRules = contract.clawRules;
export type Claw = {operatorId:string;operatorName:string;leaseUntil:number;x:number;y:number;phase:'idle'|'aiming'|'dropping'|'lifting'|'releasing';phaseAt:number;dropId:number;prizeId:string};
export type ClawSnapshot=Omit<AreaSnapshot,'kind'|'payload'> & {kind:'claw';payload:{claw:Claw}};
export type ClawAction='claw.enter'|'claw.leave'|'claw.move'|'claw.drop';
export type ClawIntent={incarnation:string;areaGeneration:0;generation:number;machineId:'claw-1';action:ClawAction;dx?:number;dy?:number};
export function validateClawIntent(raw:unknown):ClawIntent {
 const p=raw as Record<string,unknown>,move=p?.action==='claw.move';
 if(!keys(raw,['incarnation','areaGeneration','generation','machineId','action',...(move?['dx','dy']:[])])||!text(p.incarnation)||p.areaGeneration!==0||!int(p.generation)||Number(p.generation)<1||p.machineId!=='claw-1'||!['claw.enter','claw.leave','claw.move','claw.drop'].includes(String(p.action))||move&&(!number(p.dx,-1,1)||!number(p.dy,-1,1)))throw Error('Invalid claw intent');
 return structuredClone(raw) as ClawIntent;
}
function validClaw(raw:unknown) {
 if(!keys(raw,['operatorId','operatorName','leaseUntil','x','y','phase','phaseAt','dropId','prizeId']))return false;
 const p=raw;
 return [p.leaseUntil,p.phaseAt,p.dropId].every(int)&&number(p.x,0,1)&&number(p.y,0,1)&&['idle','aiming','dropping','lifting','releasing'].includes(String(p.phase))&&typeof p.operatorId==='string'&&p.operatorId.length<=128&&typeof p.operatorName==='string'&&p.operatorName.length<=128&&(p.prizeId===''||clawRules.presets.some(v=>v.id===p.prizeId))&&(p.phase==='idle'?p.operatorId===''&&p.operatorName===''&&p.leaseUntil===0&&p.prizeId==='':text(p.operatorId)&&text(p.operatorName)&&Number(p.leaseUntil)>0&&(p.phase!=='aiming'||p.prizeId===''));
}

export const fishingRules=contract.fishingRules;
export type Fishing={ownerId:string;ownerName:string;phase:'idle'|'waiting'|'bite'|'result';castId:number;biteAt:number;endsAt:number;catchId:string};
export type FishingView={id:string;generation:number;fishing:Fishing};
export type FishingSnapshot=Omit<AreaSnapshot,'kind'|'payload'>&{kind:'fishing';payload:{fishing:Fishing}};
export type FishingAction='fishing.cast'|'fishing.reel'|'fishing.leave';
export type FishingIntent={incarnation:string;areaGeneration:0;generation:number;spotId:string;action:FishingAction};
export function validateFishingIntent(raw:unknown):FishingIntent {
 if(!keys(raw,['incarnation','areaGeneration','generation','spotId','action'])||!text(raw.incarnation)||raw.areaGeneration!==0||!int(raw.generation)||Number(raw.generation)<1||!contract.fishingSpots.some(s=>s.id===raw.spotId)||!['fishing.cast','fishing.reel','fishing.leave'].includes(String(raw.action)))throw Error('Invalid fishing intent');
 return structuredClone(raw) as FishingIntent;
}
function validFishing(raw:unknown){
 if(!keys(raw,['ownerId','ownerName','phase','castId','biteAt','endsAt','catchId']))return false;
 const f=raw;
 if(![f.castId,f.biteAt,f.endsAt].every(int)||typeof f.ownerId!=='string'||f.ownerId.length>128||typeof f.ownerName!=='string'||f.ownerName.length>128||!(f.catchId===''||fishingRules.catches.includes(String(f.catchId))))return false;
 if(f.phase==='idle')return f.ownerId===''&&f.ownerName===''&&f.biteAt===0&&f.endsAt===0&&f.catchId==='';
 return ['waiting','bite','result'].includes(String(f.phase))&&text(f.ownerId)&&text(f.ownerName)&&Number(f.castId)>0&&Number(f.biteAt)>0&&Number(f.endsAt)>0&&(f.phase==='result'||Number(f.endsAt)>Number(f.biteAt)&&f.catchId==='');
}
