import {validateActivitySnapshot,validateRideIntent} from '../activities/contracts';
import { finite, point, type Point } from './map';
export interface Avatar extends Point {
    id: string;
    name: string;
    status: 'available' | 'playing' | 'AFK' | 'voice';
    facing?: number;
    avatarId?: string;
}
export interface PublicState {
    avatars: Avatar[];
}
export function validateAvatars(state: PublicState): PublicState { if (!state || !Array.isArray(state.avatars) || state.avatars.length > 20)
    throw Error('Invalid public state'); const ids = new Set<string>(); for (const a of state.avatars) {
    if (!a || !finite(a.x) || !finite(a.y) || !a.id || ids.has(a.id) || typeof a.name !== 'string' || Array.from(a.name).length > 40 || !['available', 'playing', 'AFK', 'voice'].includes(a.status))
        throw Error('Invalid avatar');
    ids.add(a.id);
} return structuredClone(state); }
export const commandTypes = ['activity.ride','activity.snapshot', 'world.join', 'world.leave', 'player.input', 'player.destination', 'player.interact', 'lobby.list', 'lobby.join', 'lobby.bot', 'lobby.ready', 'lobby.leave', 'match.start', 'race.input', 'race.snapshot', 'penalty.snapshot', 'match.return', 'match.rematch', 'penalty.choose', 'duck.shoot', 'duck.snapshot', 'duck.clock', 'duck.clock_reply', 'chat.send', 'voice.join', 'voice.leave','voice.state','voice.token','voice.unmute','voice.chat','voice.moderate', 'session.resume'] as const;
export interface Command {
    v: 2;
    commandId: string;
    seq: number;
    worldId: string;
    type: typeof commandTypes[number];
    payload: Record<string, unknown>;
}
export function validateCommand(raw: unknown): Command {
    const c = raw as Command;
    const text = (v: unknown) => typeof v === 'string' && Array.from(v).length > 0 && Array.from(v).length <= 128;
    if (!c || !Object.keys(c).every(k => ['v', 'commandId', 'seq', 'worldId', 'type', 'payload'].includes(k)) || c.v !== 2 || !text(c.commandId) || !text(c.worldId) || !Number.isSafeInteger(c.seq) || c.seq < 0 || !commandTypes.includes(c.type) || !c.payload || typeof c.payload !== 'object' || Array.isArray(c.payload))
        throw Error('Invalid command');
    const p = c.payload;
 if(c.type==='activity.ride')validateRideIntent(p);
    const integer = (n: unknown, min: number, max: number) => finite(n, min, max) && Number.isInteger(n);
    if (c.type === 'penalty.choose' && (!integer(p.zone, 0, 3)||!text(p.matchId)||!text(p.turnId)) || ['duck.snapshot','duck.clock','race.snapshot','penalty.snapshot','match.return','match.rematch'].includes(c.type)&&!text(p.matchId) || c.type === 'player.destination' && !point(p) || c.type === 'player.input' && (!finite(p.dx, -1, 1) || !finite(p.dy, -1, 1)) || c.type === 'chat.send' && (typeof p.text !== 'string' || Array.from(p.text).length < 1 || Array.from(p.text).length > 500 || p.channelId !== 'world' || !text(p.clientMessageId)) || c.type === 'lobby.join' && !text(p.arenaId) || c.type === 'match.start' && !text(p.lobbyId) || c.type === 'lobby.bot' && (!text(p.lobbyId) || typeof p.enabled !== 'boolean') || c.type === 'lobby.ready' && (!text(p.lobbyId) || typeof p.ready !== 'boolean') || c.type === 'duck.shoot' && (!text(p.matchId)||!finite(p.clientTimeMs,0,9007199254740991)||!finite(p.x, 0, 1) || !finite(p.y, 0, 1)) || c.type === 'duck.clock_reply' && (!text(p.matchId)||!text(p.nonce)||!finite(p.clientTimeMs,0,9007199254740991)) || c.type === 'race.input' && (!text(p.matchId)||!integer(p.steer, -1, 1) || !integer(p.throttle, 0, 1)||!integer(p.brake,0,1)) || c.type === 'player.interact' && !text(p.targetId) || c.type === 'voice.join' && !integer(p.table, 0, 1))
        throw Error('Invalid command payload');
    if(c.type==='voice.chat'&&(!voiceMessageID(p.clientMessageId)||typeof p.text!=='string'||Array.from(p.text).length<1||Array.from(p.text).length>500)||c.type==='voice.moderate'&&(!text(p.target)||!['mute','kick'].includes(String(p.action))))throw Error('Invalid voice command');
 const allowed: Record<string, string[]> = {'activity.snapshot':[],'voice.leave':[],'voice.state':[],'voice.token':[],'voice.unmute':[],'voice.chat':['clientMessageId','text'],'voice.moderate':['target','action'], 'penalty.snapshot':['matchId'],'match.return':['matchId'],'match.rematch':['matchId'],'penalty.choose': ['zone','matchId','turnId'], 'player.input': ['dx', 'dy'], 'player.destination': ['x', 'y'], 'player.interact': ['targetId'], 'lobby.list': [], 'lobby.leave': [], 'lobby.join': ['arenaId'], 'match.start': ['lobbyId'], 'lobby.bot': ['lobbyId','enabled'], 'lobby.ready': ['lobbyId','ready'], 'race.snapshot':['matchId'],'race.input': ['matchId','steer', 'throttle','brake'], 'duck.snapshot':['matchId'],'duck.clock':['matchId'],'duck.clock_reply':['matchId','nonce','clientTimeMs'],'duck.shoot': ['matchId','x','y','clientTimeMs'], 'chat.send': ['channelId', 'clientMessageId', 'text'], 'voice.join': ['table'] };
    if (allowed[c.type] && !Object.keys(p).every(k => allowed[c.type].includes(k)))
        throw Error('Unknown payload field');
    const serialized = JSON.stringify(c, (_key, value) => { if (typeof value === 'number' && !Number.isFinite(value))
        throw Error('Nonfinite number'); return value; });
    if (new TextEncoder().encode(serialized).length > 8192)
        throw Error('Command too large');
    return structuredClone(c);
}

export const eventTypes=['activity.snapshot','activity.lifecycle','world.snapshot','presence.changed','world.delta','command.ack','lobby.updated','match.started','match.result_status','leaderboard.changed','race.snapshot','race.finished','penalty.snapshot','penalty.choice_ack','penalty.turn_opened','penalty.revealed','duck.shot_resolved','duck.snapshot','duck.finished','chat.message','chat.deleted','command.error','voice.seat_granted','voice.seat_revoked'] as const;
export interface ServerEvent {v:2;eventId:string;serverTick:number;serverTimeMs:number;type:typeof eventTypes[number];payload:Record<string,unknown>}
/** Detailed event payloads are owned by the corresponding game/public snapshot contracts. */
export function validateEvent(raw:unknown):ServerEvent {
 const e=raw as ServerEvent;
 if(!e||!Object.keys(e).every(k=>['v','eventId','serverTick','serverTimeMs','type','payload'].includes(k))||e.v!==2||typeof e.eventId!=='string'||Array.from(e.eventId).length<1||Array.from(e.eventId).length>128||!Number.isSafeInteger(e.serverTick)||e.serverTick<0||!Number.isSafeInteger(e.serverTimeMs)||e.serverTimeMs<0||!eventTypes.includes(e.type)||!e.payload||typeof e.payload!=='object'||Array.isArray(e.payload))throw Error('Invalid event');
 if(e.type==='activity.snapshot'||e.type==='activity.lifecycle')validateActivitySnapshot(e.payload);
 if(e.type==='leaderboard.changed'&&(!Object.keys(e.payload).every(k=>['worldId','ruleset','resultId'].includes(k))||!['worldId','ruleset','resultId'].every(k=>typeof e.payload[k]==='string'&&String(e.payload[k]).length>0&&String(e.payload[k]).length<=128)))throw Error('Invalid podium invalidation');
 if(e.type==='match.result_status'&&(!Object.keys(e.payload).every(k=>['matchId','status'].includes(k))||typeof e.payload.matchId!=='string'||!e.payload.matchId||e.payload.matchId.length>128||!['finalizing','unconfirmed','completed','aborted_storage','aborted_server','abandoned','unavailable'].includes(String(e.payload.status))))throw Error('Invalid result status');
 const serialized=JSON.stringify(e,(_key,value)=>{if(typeof value==='number'&&!Number.isFinite(value))throw Error('Nonfinite event number');return value});
 if(new TextEncoder().encode(serialized).length>1048576)throw Error('Event too large');
 return structuredClone(e);
}

function voiceMessageID(value:unknown):boolean {return typeof value==='string'&&/^[a-f0-9]{48}:[1-9][0-9]{0,15}$/.test(value)&&Number.isSafeInteger(Number(value.split(':')[1]))}
