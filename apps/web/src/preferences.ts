import {AuthState,authState,type AuthSnapshot} from './auth-state';
import {api,type User,type Profile} from './net/api';
import {defaults,loadPreferences,savePreferences,saveMotion,resetGuestData,type Preferences} from './guest/history';
export type PreferenceIO={deviceLoad:()=>Promise<Preferences>;deviceSave:(p:Preferences)=>Promise<void>;deviceReset:()=>Promise<void>;deviceMotion:(reduced:boolean)=>Promise<Preferences|undefined>;accountMotion:(reduced:boolean,signal?:AbortSignal)=>Promise<Profile>;accountLoad:(signal?:AbortSignal)=>Promise<Profile>;accountSave:(p:Profile,signal?:AbortSignal)=>Promise<User>};
const io:PreferenceIO={deviceLoad:loadPreferences,deviceSave:savePreferences,deviceReset:resetGuestData,deviceMotion:saveMotion,accountMotion:(reduced,signal)=>api.updateMotion(reduced,signal),accountLoad:signal=>api.profile(signal),accountSave:(p,signal)=>api.updateProfile(p,signal)};
export class ActivePreferences{
 private value:Preferences={...defaults};private loadedGeneration=-1;private loadRevision=0;private listeners=new Set<(p:Preferences)=>void>();
 constructor(readonly auth:AuthState,private io:PreferenceIO){}
 snapshot(){return {...this.value}}
 ready(g=this.auth.snapshot().generation){return this.loadedGeneration===g}
 subscribe(fn:(p:Preferences)=>void){this.listeners.add(fn);return()=>{this.listeners.delete(fn)}}
 private publish(g:number,p:Preferences){if(!this.auth.is(g))return false;this.value={...p};this.loadedGeneration=g;for(const listener of this.listeners)listener(this.snapshot());return true}
 async read(g:number,signal?:AbortSignal){if(!this.auth.is(g))throw Error('Sesi berubah.');const user=this.auth.snapshot().user;const p=user?await this.io.accountLoad(signal):await this.io.deviceLoad();return user?{...(p as Profile).settings,displayName:(p as Profile).displayName,avatar:(p as Profile).avatar}:p as Preferences}
 async load(){const g=this.auth.snapshot().generation,revision=++this.loadRevision;const p=await this.read(g);if(revision===this.loadRevision)this.publish(g,p)}
 start(){let last=-1;const load=({generation}:AuthSnapshot)=>{if(last===generation)return;last=generation;void this.load().catch(()=>{})};const off=this.auth.subscribe(load);load(this.auth.snapshot());return off}
 async save(p:Preferences,g=this.auth.snapshot().generation,signal?:AbortSignal){if(!this.auth.is(g)||signal?.aborted)return false;const user=this.auth.snapshot().user;let updated:User|undefined;
 if(user){updated=await this.io.accountSave({displayName:p.displayName,avatar:p.avatar,settings:{audio:p.audio,controls:p.controls,reducedMotion:p.reducedMotion}},signal)}else await this.io.deviceSave(p);
 if(!this.auth.is(g)||signal?.aborted)return false;this.loadRevision++;if(updated&&!this.auth.update(g,updated))return false;return this.publish(g,p)
 }
 async toggleMotion(){
 const g=this.auth.snapshot().generation,controller=new AbortController();
 // Own cancellation here, not in start(): standalone callers need the same fence.
 const off=this.auth.subscribe(({generation})=>{if(generation!==g)controller.abort()});
 try{
 if(this.loadedGeneration!==g)await this.load();
 if(controller.signal.aborted||!this.auth.is(g)||this.loadedGeneration!==g)return false;
 const user=this.auth.snapshot().user,reduced=!this.value.reducedMotion;
 const result=user?await this.io.accountMotion(reduced,controller.signal):await this.io.deviceMotion(reduced);
 if(!result||controller.signal.aborted||!this.auth.is(g))return false;
 const p=user?{...(result as Profile).settings,displayName:(result as Profile).displayName,avatar:(result as Profile).avatar}:result as Preferences;
 this.loadRevision++;if(user&&!this.auth.update(g,{...user,displayName:p.displayName,avatar:p.avatar}))return false;return this.publish(g,p);
 }catch(error){if(controller.signal.aborted||!this.auth.is(g))return false;throw error}
 finally{off()}
 }
 async resetDevice(g=this.auth.snapshot().generation){await this.io.deviceReset();if(this.auth.is(g)&&!this.auth.snapshot().user){this.loadRevision++;this.publish(g,{...defaults})}}
}
export const activePreferences=new ActivePreferences(authState,io);
