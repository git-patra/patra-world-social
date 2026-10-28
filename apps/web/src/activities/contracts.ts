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
export type ActivitySnapshot = AreaSnapshot | VehiclesSnapshot;
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
export type ActivityIntent = {
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
    if (!keys(raw, ['worldId', 'areaId', 'incarnation', 'generation', 'serverTimeMs', 'tick', 'kind', 'payload']) || !text(raw.worldId) || (worldId !== undefined && raw.worldId !== worldId) || !text(raw.incarnation) || !contract.areas.some(a => a.id === raw.areaId) || !['area', 'vehicles'].includes(String(raw.kind)) || ![raw.generation, raw.serverTimeMs, raw.tick].every(int))
        throw Error('Invalid activity snapshot');
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
