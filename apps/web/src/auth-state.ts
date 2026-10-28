import type {User} from './net/api';
export type AuthSnapshot={generation:number;user:User|null};
/** Session identity includes an epoch: logging back into the same account is new. */
export class AuthState{
 private value:AuthSnapshot={generation:0,user:null};private listeners=new Set<(s:AuthSnapshot)=>void>();
 snapshot():AuthSnapshot{return {...this.value,user:this.value.user?{...this.value.user}:null}}
 is(generation:number){return generation===this.value.generation}
 subscribe(listener:(s:AuthSnapshot)=>void){this.listeners.add(listener);return()=>{this.listeners.delete(listener)}}
 private emit(){for(const listener of this.listeners)listener(this.snapshot())}
 begin(){this.value={...this.value,generation:this.value.generation+1};this.emit();return this.value.generation}
 transition(user:User|null,expected=this.value.generation){if(!this.is(expected))return false;this.value={user,generation:this.value.generation+1};this.emit();return true}
 update(expected:number,user:User|null){if(!this.is(expected))return false;if(!user)return this.transition(null,expected);if(user.id!==this.value.user?.id)return false;this.value={...this.value,user};this.emit();return true}
}
export const authState=new AuthState();
// Profile refreshes notify state subscribers, but only a new session tears down play.
let bridgedGeneration=authState.snapshot().generation;
authState.subscribe(({user,generation})=>{if(generation===bridgedGeneration)return;bridgedGeneration=generation;if(typeof window!=='undefined')window.dispatchEvent(new CustomEvent('patra-auth-change',{detail:user}))});
