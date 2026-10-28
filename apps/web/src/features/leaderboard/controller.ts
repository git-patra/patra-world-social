export interface Entry {userId:string;displayName:string;resultId:string;score:number;hits:number;shots:number;achievedAt:number}
export interface ResultSummary extends Entry {version:{mapId:string;mapVersion:number;rulesId:string;rulesVersion:number;protocolVersion:number};status:string;complete:boolean}
export interface Board {worldId:string;ruleset:string;top:Entry[];personalBest:Entry|null;lastResult:ResultSummary|null;changes:{id:number;userId:string;displayName:string;resultId:string;changedAt:string}[];administrativeChanges:{id:number;reason:string;changedAt:string}[];rulesets:string[]}
export class ReadError extends Error {
 constructor(readonly status:number){super(status===401?'Masuk untuk melihat podium dunia privat.':status===403||status===404?'Keanggotaan aktif atau hasil tidak tersedia.':'Podium belum dapat dimuat. Coba lagi.')}
}
export const accessDenied=(error:unknown)=>error instanceof ReadError&&[401,403,404].includes(error.status);
export async function read<T>(path:string,signal:AbortSignal):Promise<T>{const r=await fetch(path,{signal,credentials:'same-origin',cache:'no-store'});if(!r.ok)throw new ReadError(r.status);return r.json()}
// Replace complete committed snapshots; never merge a provisional score into a PB.
export class BoardController {
 private epoch=0;private active=true;private abort:AbortController|null=null;private seen=new Set<string>();settled:Promise<void>=Promise.resolve();
 constructor(readonly world:string,readonly rules:string,private fetchBoard:(path:string,signal:AbortSignal)=>Promise<Board>,private accept:(b:Board)=>void,private failure:(message:string,error?:unknown)=>void=()=>{}){}
 load(){this.abort?.abort();const abort=this.abort=new AbortController(),epoch=++this.epoch;this.settled=this.fetchBoard(`/api/v1/leaderboards/${encodeURIComponent(this.world)}?ruleset=${encodeURIComponent(this.rules)}`,abort.signal).then(b=>{if(this.active&&epoch===this.epoch&&!abort.signal.aborted)this.accept(b)}).catch(e=>{if(this.active&&epoch===this.epoch&&!abort.signal.aborted)this.failure(String(e.message),e)});return this.settled}
 event(type:string,p:Record<string,unknown>){
  let key='';if(type==='leaderboard.changed'&&p.worldId===this.world&&p.ruleset===this.rules&&typeof p.resultId==='string')key=p.resultId;
  // The enclosing socket listener already fences world identity for private status.
  if(type==='match.result_status'&&['completed','aborted_storage','aborted_server','abandoned'].includes(String(p.status))&&typeof p.matchId==='string')key=p.matchId;
  if(!key||this.seen.has(key)||!this.active)return false;this.seen.add(key);if(this.seen.size>256)this.seen.delete(this.seen.values().next().value!);void this.load();return true
 }
 dispose(){this.active=false;this.epoch++;this.abort?.abort();this.seen.clear()}
}
