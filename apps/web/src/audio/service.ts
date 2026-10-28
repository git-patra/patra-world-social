export type SoundKind='ui'|'paddle'|'kick'|'goal'|'save'|'countdown'|'finish'|'grab'|'ride'|'fire'|'hit'|'cast'|'catch';
export type AudioEvent={id:string;kind:SoundKind;at:number;scope?:string;worldPosition?:{x:number;y:number}};
export type SoundEvent={kind:SoundKind;eventId:string;mode:string;scope?:string;revision?:number;generation?:number;distance?:number;loop?:boolean;resync?:boolean};
const tones:Record<SoundKind,[number,number,number]>={ui:[520,700,.055],paddle:[430,170,.065],kick:[160,60,.09],goal:[660,1100,.22],save:[280,180,.14],countdown:[440,440,.09],finish:[520,1040,.3],grab:[220,450,.18],ride:[100,80,.15],fire:[190,45,.075],hit:[760,330,.11],cast:[320,90,.17],catch:[620,980,.22]};
type Voice={mode:string;stop:()=>void};
/** Short synthesized SFX only. This service never owns a microphone or media track. */
export class AudioService{
 private context:AudioContext|null=null;private enabled=false;private hidden=false;private generation=0;private unlocked=false;private disposed=false;private unlockRevision=0;
 private listenerPosition:{x:number;y:number}|null=null;
 private voices=new Set<Voice>();private seen=new Set<string>();private revisions=new Map<string,number>();private lastFire=-Infinity;
 constructor(private createContext:()=>AudioContext=()=>new AudioContext()){}
 setEnabled(v:boolean){this.enabled=v;if(!v){this.unlockRevision++;this.unlocked=false;this.stopAll()}}
 setHidden(v:boolean){this.hidden=v;if(v){this.unlockRevision++;this.stopAll();this.unlocked=false;void this.context?.suspend().catch(()=>{})}}
 setGeneration(g:number){if(g===this.generation)return;this.generation=g;this.unlockRevision++;this.enabled=false;this.unlocked=false;this.stopAll();this.seen.clear();this.revisions.clear();this.lastFire=-Infinity}
 async unlock(){if(this.disposed||!this.enabled||this.hidden)return;const generation=this.generation,revision=this.unlockRevision;try{const context=this.context??=this.createContext();await context.resume();if(generation===this.generation&&revision===this.unlockRevision&&this.enabled&&!this.hidden&&!this.disposed)this.unlocked=context.state==='running'}catch{this.unlocked=false}}
 setListenerPosition(position:{x:number;y:number}|null){this.listenerPosition=position}
 emit(e:AudioEvent){const age=performance.now()-e.at;const distance=e.worldPosition&&this.listenerPosition?Math.hypot(e.worldPosition.x-this.listenerPosition.x,e.worldPosition.y-this.listenerPosition.y):0;return this.play({eventId:e.id,kind:e.kind,mode:e.scope??e.kind,distance,resync:!Number.isFinite(age)||age>1000||age< -250})}
 stopScope(scope:string){this.stopMode(scope)}
 play(e:SoundEvent){
  if(this.disposed||e.generation!==undefined&&e.generation!==this.generation)return false;
  if(this.seen.has(e.eventId))return false;this.seen.add(e.eventId);if(this.seen.size>512)this.seen.delete(this.seen.values().next().value!);
  if(e.scope&&e.revision!==undefined){const previous=this.revisions.get(e.scope)??-Infinity;if(e.revision<=previous)return false;this.revisions.set(e.scope,e.revision);if(this.revisions.size>128)this.revisions.delete(this.revisions.keys().next().value!)}
  const context=this.context,distance=e.distance??0;
  if(e.resync||!this.enabled||this.hidden||!this.unlocked||context?.state!=='running'||!Number.isFinite(distance)||distance>=24)return false;
  if(e.kind==='fire'&&context.currentTime-this.lastFire<.075)return false;if(e.kind==='fire')this.lastFire=context.currentTime;
  if(this.voices.size>=8)this.voices.values().next().value!.stop();
  const [from,to,duration]=tones[e.kind],osc=context.createOscillator(),gain=context.createGain(),at=context.currentTime;
  let released=false;const release=()=>{if(released)return;released=true;osc.disconnect();gain.disconnect();this.voices.delete(voice)};
  const voice:Voice={mode:e.mode,stop:()=>{try{osc.stop()}catch{}release()}};this.voices.add(voice);osc.onended=release;
  osc.type=e.kind==='fire'||e.kind==='ride'?'triangle':'sine';osc.frequency.setValueAtTime(from,at);osc.frequency.exponentialRampToValueAtTime(to,at+duration);
  gain.gain.setValueAtTime(.12/(1+Math.max(0,distance)/4),at);if(!e.loop)gain.gain.exponentialRampToValueAtTime(.0001,at+duration);
  osc.connect(gain);gain.connect(context.destination);osc.start();if(!e.loop)osc.stop(at+duration);return true;
 }
 stopMode(mode:string){for(const voice of this.voices)if(voice.mode===mode)voice.stop()}
 private stopAll(){for(const voice of this.voices)voice.stop()}
 dispose(){this.stopAll();this.unlocked=false;this.disposed=true;this.seen.clear();this.revisions.clear();void this.context?.close().catch(()=>{})}
}
export const audio=new AudioService();
