export type GameType='race'|'penalty'|'duck'|'pingpong';
export type Preferences={displayName:string;avatar:'pumpkin'|'autumn'|'moon';audio:boolean;controls:'both'|'keyboard'|'touch';reducedMotion:boolean};
export type LocalResult={id:string;gameType:GameType;finishedAt:number;durationMs:number;scope:'device';version:{mapId:string;mapVersion:number;rulesId:string;rulesVersion:number;protocolVersion:number};summary:string;detail:unknown};
export const defaults:Preferences={displayName:'Guest',avatar:'pumpkin',audio:true,controls:'both',reducedMotion:false};
export function retained(results:LocalResult[]){return [...new Map(results.map(r=>[r.id,r])).values()].sort((a,b)=>b.finishedAt-a.finishedAt||b.id.localeCompare(a.id)).slice(0,100)}
/** A transaction owns insertion, duplicate rejection and oldest-first retention. */
export class GuestHistory{
 private generation=0;
 constructor(readonly name='patra-guest-v1',private factory:()=>IDBFactory=()=>indexedDB){}
 private open():Promise<IDBDatabase>{return new Promise((resolve,reject)=>{let r:IDBOpenDBRequest;try{r=this.factory().open(this.name,1)}catch(e){reject(e);return}r.onupgradeneeded=()=>{r.result.createObjectStore('results',{keyPath:'id'});r.result.createObjectStore('settings')};r.onsuccess=()=>resolve(r.result);r.onerror=()=>reject(r.error)})}
 private async transaction<T>(stores:string[],mode:IDBTransactionMode,work:(tx:IDBTransaction,done:(v:T)=>void)=>void):Promise<T>{const db=await this.open();return new Promise((resolve,reject)=>{const tx=db.transaction(stores,mode);let value:T;tx.oncomplete=()=>{db.close();resolve(value)};tx.onerror=tx.onabort=()=>{db.close();reject(tx.error??Error('Penyimpanan perangkat gagal.'))};try{work(tx,v=>{value=v})}catch(e){tx.abort();reject(e)}})}
 append(result:LocalResult):Promise<void>{const generation=this.generation;if(result.scope!=='device'||!result.id||!['race','penalty','duck','pingpong'].includes(result.gameType)||!Number.isFinite(result.finishedAt))return Promise.reject(Error('Hasil lokal tidak valid.'));return this.transaction<void>(['results'],'readwrite',(tx,done)=>{if(generation!==this.generation){done();return}const store=tx.objectStore('results'),read=store.getAll();read.onsuccess=()=>{const old=read.result as LocalResult[];if(old.some(r=>r.id===result.id)){done();return}const all=retained([...old,result]),keep=new Set(all.map(r=>r.id));store.put(result);for(const r of [...old,result])if(!keep.has(r.id))store.delete(r.id);done()}})}
 list(){return this.transaction<LocalResult[]>(['results'],'readonly',(tx,done)=>{const r=tx.objectStore('results').getAll();r.onsuccess=()=>done(retained(r.result))})}
 settings(){return this.transaction<Preferences>(['settings'],'readonly',(tx,done)=>{const r=tx.objectStore('settings').get('preferences');r.onsuccess=()=>done({...defaults,...r.result})})}
 saveSettings(value:Preferences){const generation=this.generation;return this.transaction<void>(['settings'],'readwrite',(tx,done)=>{if(generation===this.generation)tx.objectStore('settings').put(value,'preferences');done()})}
 setMotion(reducedMotion:boolean){const generation=this.generation;return this.transaction<Preferences|undefined>(['settings'],'readwrite',(tx,done)=>{if(generation!==this.generation){done(undefined);return}const store=tx.objectStore('settings'),r=store.get('preferences');r.onsuccess=()=>{if(generation!==this.generation){done(undefined);return}const next={...defaults,...r.result,reducedMotion};store.put(next,'preferences');done(next)}})}
 reset(){this.generation++;return this.transaction<void>(['results','settings'],'readwrite',(tx,done)=>{tx.objectStore('results').clear();tx.objectStore('settings').clear();done()})}
}
export const guestHistory=new GuestHistory();
let current={...defaults};
export function preferences(){return {...current}}
export async function loadPreferences(){current=await guestHistory.settings();return preferences()}
export async function savePreferences(value:Preferences){await guestHistory.saveSettings(value);current={...value};if(typeof window!=='undefined')window.dispatchEvent(new CustomEvent('patra-device-preferences',{detail:preferences()}))}
export async function resetGuestData(){await guestHistory.reset();current={...defaults};if(typeof window!=='undefined'){window.dispatchEvent(new Event('patra-guest-reset'));window.dispatchEvent(new CustomEvent('patra-device-preferences',{detail:preferences()}))}}

export async function saveMotion(reducedMotion:boolean){const p=await guestHistory.setMotion(reducedMotion);if(p)current=p;return p}
