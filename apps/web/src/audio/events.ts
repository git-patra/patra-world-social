import {audio,AudioService,type SoundKind} from './service';
import {activePreferences} from '../preferences';
import {authState} from '../auth-state';
type Snapshot={matchId:string;worldId?:string;phase:string;startsAt?:number;serverTimeMs?:number;turns?:{turn:number;goal:boolean}[];result?:unknown;shots?:number};
type Observed={phase:string;count:number;turn:number;at:number};
export class GameAudio{
 private matches=new Map<string,Observed>();
 constructor(private service:AudioService){}
 reset(){this.matches.clear()}
 duck(s:{matchId:string;worldId?:string;shots:number;points:number},generation?:number,resync=false){if(!Number.isSafeInteger(s.shots)||s.shots<1)return;const scope=`duck:${s.worldId??'local'}:${s.matchId}:shot`;this.service.play({kind:s.points>0?'hit':'fire',mode:'duck',eventId:`${scope}:${s.shots}`,scope,revision:s.shots,generation,resync})}
 observe(mode:string,s:Snapshot,at:number,resync=false){
  const key=`${mode}:${s.worldId??'local'}:${s.matchId}`,count=s.phase==='countdown'?Math.max(0,Math.ceil(((s.startsAt??0)-at)/1000)):0,turn=s.turns?.at(-1)?.turn??0,previous=this.matches.get(key);
  if(previous&&at<previous.at)return;
  const phase=s.result||s.phase==='results'?'finished':s.phase;const next={phase,count,turn,at};this.matches.set(key,next);if(this.matches.size>64)this.matches.delete(this.matches.keys().next().value!);
  if(!previous||resync){if(mode==='duck'&&s.shots){const scope=`duck:${s.worldId??'local'}:${s.matchId}:shot`;this.service.play({kind:'fire',mode,eventId:`${scope}:${s.shots}`,scope,revision:s.shots,resync:true})}return;}
  const emit=(kind:SoundKind,id:string)=>this.service.play({kind,mode,eventId:`${key}:${id}`});
  if(count>0&&count<previous.count)emit('countdown',`count:${count}`);
  if(previous.phase==='countdown'&&s.phase!=='countdown'&&s.phase!=='finished')emit('countdown','go');
  if(phase==='finished'&&previous.phase!=='finished')emit('finish','finish');
  const reveal=s.turns?.at(-1);if(mode==='penalty'&&turn>previous.turn&&reveal)emit(reveal.goal?'goal':'save',`turn:${turn}`);
 }
}
export const gameAudio=new GameAudio(audio);
/** Mount once with the application, independently of game panel lifetimes. */
export function mountAudio(){
 let generation=authState.snapshot().generation;
 audio.setGeneration(generation);audio.setEnabled(activePreferences.ready(generation)&&activePreferences.snapshot().audio);audio.setHidden(document.hidden);
 const preference=activePreferences.subscribe(p=>audio.setEnabled(activePreferences.ready(generation)&&p.audio));
 const auth=authState.subscribe(s=>{if(s.generation===generation)return;generation=s.generation;audio.setGeneration(generation);gameAudio.reset()});
 let socket:{worldId:string;serverNow:()=>number;status:string}|null=null;
 const attachment=(e:Event)=>{socket=(e as CustomEvent).detail;gameAudio.reset();blur()};
 const event=(e:Event)=>{const d=(e as CustomEvent).detail;if(!socket||d?.socket!==socket||socket.status!=='online')return;if(d.type==='world.snapshot'){gameAudio.reset();blur();return}if(d.type==='duck.shot_resolved'){gameAudio.duck(d.payload,generation,!Number.isFinite(d.payload.serverTimeMs)||socket.serverNow()-d.payload.serverTimeMs>1000);return}for(const mode of ['race','penalty','duck'])if(d.type.startsWith(`${mode}.`)&&d.payload?.matchId&&typeof d.payload.phase==='string')gameAudio.observe(mode,d.payload,socket.serverNow())};
 const status=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.socket===socket&&d.state!=='online'){gameAudio.reset();blur()}};
 const gesture=(e:Event)=>{if(e.isTrusted)void audio.unlock()};
 const visibility=()=>{audio.setHidden(document.hidden);gameAudio.reset()};
 const blur=()=>{for(const mode of ['race','penalty','duck','ride','blaster','pingpong','claw','fishing'])audio.stopMode(mode)};
 const ui=(e:Event)=>{const target=e.target;if(e.isTrusted&&target instanceof Element&&target.closest('button'))audio.play({kind:'ui',mode:'ui',eventId:`ui:${++uiSequence}`})};
 document.addEventListener('pointerdown',gesture,true);document.addEventListener('keydown',gesture,true);document.addEventListener('click',ui);document.addEventListener('visibilitychange',visibility);window.addEventListener('blur',blur);window.addEventListener('patra-world-attachment',attachment);window.addEventListener('patra-world-event',event);window.addEventListener('patra-world-status',status);
 return()=>{preference();auth();document.removeEventListener('pointerdown',gesture,true);document.removeEventListener('keydown',gesture,true);document.removeEventListener('click',ui);document.removeEventListener('visibilitychange',visibility);window.removeEventListener('blur',blur);window.removeEventListener('patra-world-attachment',attachment);window.removeEventListener('patra-world-event',event);window.removeEventListener('patra-world-status',status);gameAudio.reset();audio.dispose()};
}
let uiSequence=0;
