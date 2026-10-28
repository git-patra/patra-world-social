import contract from '../../../../contracts/activities.v1.json';
export type AreaPhase = 'OPEN' | 'CLEARING' | 'RESERVED' | 'ACTIVE' | 'RESETTING';
export interface ActivitySnapshot {
 worldId:string; areaId:string; incarnation:string; generation:number; serverTimeMs:number; tick:number; kind:'area';
 payload:{phase:AreaPhase;matchId:string;startsAt:number;clearAt:number;resetAt:number};
}
/** Only resync is accepted in the foundation; gameplay intents add exact unions with their engines. */
export type ActivityIntent={type:'activity.snapshot';payload:Record<string,never>};
export const activityContract=contract;
const keys=(value:unknown,expected:string[]):value is Record<string,unknown>=>!!value&&typeof value==='object'&&!Array.isArray(value)&&Object.keys(value).length===expected.length&&Object.keys(value).every(k=>expected.includes(k));
const int=(v:unknown)=>Number.isSafeInteger(v)&&Number(v)>=0;
const text=(v:unknown)=>typeof v==='string'&&v.length>0&&v.length<=128;
export function validateActivitySnapshot(raw:unknown,worldId?:string):ActivitySnapshot {
 if(!keys(raw,['worldId','areaId','incarnation','generation','serverTimeMs','tick','kind','payload'])||!text(raw.worldId)||(worldId!==undefined&&raw.worldId!==worldId)||!text(raw.incarnation)||!contract.areas.some(a=>a.id===raw.areaId)||raw.kind!=='area'||![raw.generation,raw.serverTimeMs,raw.tick].every(int)||!keys(raw.payload,['phase','matchId','startsAt','clearAt','resetAt']))throw Error('Invalid activity snapshot');
 const p=raw.payload;
 if(!['OPEN','CLEARING','RESERVED','ACTIVE','RESETTING'].includes(String(p.phase))||typeof p.matchId!=='string'||p.matchId.length>128||![p.startsAt,p.clearAt,p.resetAt].every(int)||(p.phase!=='OPEN'&&(!text(p.matchId)||raw.generation===0))||(p.phase==='OPEN'&&p.matchId!=='')||(p.phase==='CLEARING'&&p.clearAt===0)||(p.phase==='ACTIVE'&&p.startsAt===0)||(p.phase==='RESETTING'&&p.resetAt===0))throw Error('Invalid area phase');
 return structuredClone(raw) as unknown as ActivitySnapshot;
}
