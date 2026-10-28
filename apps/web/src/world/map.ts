import rulesData from '../../../../contracts/rules.v1.json';
export interface Point {
    x: number;
    y: number;
}
export interface Building extends Point {
    id: string;
    width: number;
    height: number;
    entrance: Point;
}
export interface MapDefinition {
    v: 1;
    id: string;
    checksum: string;
    width: 64;
    height: 64;
    spawn: Point;
    buildings: Building[];
    obstacles: Point[];
    districts: {
        id: string;
        x: number;
        y: number;
        width: number;
        height: number;
    }[];
}
export const finite = (n: unknown, min = 0, max = 63): n is number => typeof n === 'number' && Number.isFinite(n) && n >= min && n <= max;
export const point = (p: any): p is Point => p && typeof p === 'object' && Object.keys(p).every(k => ['x', 'y'].includes(k)) && finite(p.x) && finite(p.y);
const text = (v: unknown): v is string => typeof v === 'string' && Array.from(v).length > 0 && Array.from(v).length <= 128;
const coordinates = (p: any) => p && finite(p.x) && finite(p.y);
const keys = (o: object, allowed: string[]) => Object.keys(o).every(k => allowed.includes(k));
export function loadMap(data: unknown): MapDefinition {
    const m = data as MapDefinition;
    if (!m || !keys(m, ['v', 'id', 'checksum', 'width', 'height', 'spawn', 'buildings', 'districts', 'obstacles']) || m.v !== 1 || m.width !== 64 || m.height !== 64 || !text(m.id) || !text(m.checksum) || !point(m.spawn) || !Number.isInteger(m.spawn.x) || !Number.isInteger(m.spawn.y) || !Array.isArray(m.buildings) || !Array.isArray(m.districts) || !Array.isArray(m.obstacles) || m.obstacles.some(p => !point(p) || !Number.isInteger(p.x) || !Number.isInteger(p.y)))
        throw Error('Invalid map');
    const ids = new Set<string>();
    for (const b of m.buildings) {
        if (!b || !keys(b, ['id', 'x', 'y', 'width', 'height', 'entrance']) || !text(b.id) || ids.has(b.id) || !coordinates(b) || !point(b.entrance) || ![b.x, b.y, b.width, b.height, b.entrance.x, b.entrance.y].every(Number.isInteger) || !finite(b.width, 1, 64) || !finite(b.height, 1, 64) || b.x + b.width > 64 || b.y + b.height > 64)
            throw Error('Invalid building');
        ids.add(b.id);
    }
    const blocked = collisionTiles(m);
    if (blocked.has(`${m.spawn.x},${m.spawn.y}`) || m.buildings.some(b => blocked.has(`${b.entrance.x},${b.entrance.y}`)))
        throw Error('Blocked spawn or entrance');
    const districtIds = new Set<string>();
    for (const d of m.districts) {
        if (districtIds.has(d.id))
            throw Error('Duplicate district');
        districtIds.add(d.id);
        if (!keys(d, ['id', 'x', 'y', 'width', 'height']) || !text(d.id) || ![d.x, d.y, d.width, d.height].every(Number.isInteger) || !coordinates(d) || !finite(d.width, 1, 64) || !finite(d.height, 1, 64) || d.x + d.width > 64 || d.y + d.height > 64)
            throw Error('Invalid district');
    }
    for (let i=0;i<m.districts.length;i++) for(let j=i+1;j<m.districts.length;j++) {
        const a=m.districts[i],b=m.districts[j];
        if(a.x<b.x+b.width&&b.x<a.x+a.width&&a.y<b.y+b.height&&b.y<a.y+a.height)throw Error('Overlapping districts');
    }
    const queue=[m.spawn],reachable=new Set([`${m.spawn.x},${m.spawn.y}`]);
    for(let cursor=0;cursor<queue.length;cursor++)for(const [dx,dy] of [[1,0],[0,1],[-1,0],[0,-1]]){
        const next={x:queue[cursor].x+dx,y:queue[cursor].y+dy},key=`${next.x},${next.y}`;
        if(finite(next.x)&&finite(next.y)&&!blocked.has(key)&&!reachable.has(key)){reachable.add(key);queue.push(next)}
    }
    if(m.buildings.some(b=>!reachable.has(`${b.entrance.x},${b.entrance.y}`)))throw Error('Disconnected entrance');
    return structuredClone(m);
}
export function collisionTiles(map: MapDefinition): Set<string> { const result = new Set<string>(map.obstacles.map(p => `${p.x},${p.y}`)); for (const b of map.buildings)
    for (let x = b.x; x < b.x + b.width; x++)
        for (let y = b.y; y < b.y + b.height; y++)
            result.add(`${x},${y}`); return result; }
export function loadRules(data: unknown) { const r = data as typeof rulesData; if (!r || !keys(r, Object.keys(rulesData)) || r.v !== 1 || !text(r.id) || !text(r.checksum) || r.subunitsPerTile !== 1024 || r.hubTickHz !== 20 || r.raceTickHz !== 30 || r.penaltyZones !== 4 || r.raceLaps !== 5 || r.raceSlots !== 5 || r.penaltyDeadlineMs !== 3000 || !Number.isInteger(r.hubSpeedSubunitsPerSecond) || !Number.isInteger(r.movementIntentExpiryMs) || !finite(r.hubSpeedSubunitsPerSecond, 1, 100000) || !finite(r.movementIntentExpiryMs, 1, 1000))
    throw Error('Invalid rules'); if(!r.race||!keys(r.race,Object.keys(rulesData.race))||!text(r.race.trackId)||!text(r.race.trackChecksum)||Object.keys(rulesData.race).some(k=>!['trackId','trackChecksum'].includes(k)&&(!Number.isInteger(r.race[k])||!finite(r.race[k],1,300000))))throw Error('Invalid race rules'); return Object.freeze({ ...r }); }
export const rules = loadRules(rulesData);
