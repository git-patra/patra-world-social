import { validateAvatars, type Avatar } from '../world/avatars';
import map from '../../../../contracts/map.v1.json';
import rules from '../../../../contracts/rules.v1.json';
export interface Snapshot {
    worldId: string;
    selfId: string;
    sessionId: string;
    epoch: number;
    mapId: string;
    mapChecksum: string;
    rulesId: string;
    rulesChecksum: string;
    protocolVersion: number;
    tick: number;
    timeMs: number;
    avatars: (Avatar & {
        seq: number;
    })[];
}
export function validateSnapshot(raw: unknown, worldId: string): Snapshot {
    const s = raw as Snapshot;
    if (!s || !Object.keys(s).every(k => ['worldId', 'selfId', 'sessionId', 'epoch', 'mapId', 'mapChecksum', 'rulesId', 'rulesChecksum', 'protocolVersion', 'tick', 'timeMs', 'avatars'].includes(k)) || s.worldId !== worldId || s.protocolVersion !== 2 || s.mapId !== map.id || s.mapChecksum !== map.checksum || s.rulesId !== rules.id || s.rulesChecksum !== rules.checksum || !Number.isSafeInteger(s.epoch) || s.epoch < 1 || !Number.isSafeInteger(s.tick) || s.tick < 0 || !Number.isSafeInteger(s.timeMs) || s.timeMs < 0 || typeof s.sessionId !== 'string' || s.sessionId.length < 1 || s.sessionId.length > 128 || typeof s.selfId !== 'string')
        throw Error('Incompatible world snapshot');
    validateAvatars(s);
    if (!s.avatars.some(a => a.id === s.selfId) || s.avatars.some(a => !Object.keys(a).every(k => ['id', 'name', 'status', 'x', 'y', 'seq'].includes(k)) || typeof a.id !== 'string' || !a.name || Array.from(a.name).length > 40 || a.id.length > 128 || !Number.isSafeInteger(a.seq) || a.seq < 0))
        throw Error('Invalid authoritative avatar');
    return structuredClone(s);
}
/** Bounded interpolation with no extrapolation: visuals cannot invent movement. */
export class SnapshotBuffer {
    private previous: Snapshot | null = null;
    private current: Snapshot | null = null;
    push(s: Snapshot) {
        if (this.current && s.epoch === this.current.epoch && s.tick < this.current.tick)
            return;
        this.previous = this.current?.epoch === s.epoch && this.current.worldId === s.worldId ? this.current : null;
        this.current = s;
    }
    at(timeMs: number): Snapshot {
        if (!this.current)
            throw Error('No snapshot');
        const b = this.current, a = this.previous;
        if (!a || b.timeMs <= a.timeMs)
            return structuredClone(b);
        const fraction = Math.max(0, Math.min(1, (timeMs - a.timeMs) / (b.timeMs - a.timeMs)));
        return { ...b, avatars: b.avatars.map(p => { const old = a.avatars.find(o => o.id === p.id); return old ? { ...p, x: old.x + (p.x - old.x) * fraction, y: old.y + (p.y - old.y) * fraction } : { ...p }; }) };
    }
}
