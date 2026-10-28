import {audio} from '../../audio/service';
import {gameAudio} from '../../audio/events';
import {Completion} from '../../guest/completion';
import {preferences} from '../../guest/history';
import {LocalRace, prepare, type Controls} from './local';

export class GuestRaceSession {
 engine: LocalRace | null = null;
 private sequence = 0;
 private completion:Completion|null=null;
 private timer:ReturnType<typeof setInterval>|null=null;

 start(at: number, seed = crypto.getRandomValues(new Uint32Array(1))[0], id: string = crypto.randomUUID()) {
  this.clear();this.completion=new Completion(id,'race');
  const actors = Array.from({length: 5}, (_, i) => ({
   actorId: i === 0 ? 'guest' : `bot-${i}`,
   actorType: i === 0 ? 'human' : 'bot',
   userId: i === 0 ? 'guest' : undefined,
   displayName: i === 0 ? preferences().displayName : `BOT ${i}`,
   avatarId: preferences().avatar,
  }));
  this.engine = new LocalRace(id, prepare(actors, seed), Math.floor(at) + 3000);
  this.sequence = 0;
  if(typeof window!=='undefined')this.timer=setInterval(()=>this.snapshot(performance.now()),1000/30);
  return this.snapshot(at)!;
 }

 snapshot(at: number) {
  this.engine?.tick(Math.floor(at));
  const s=this.engine?.snapshot()??null;if(s)gameAudio.observe('race',s,at);if(s?.phase==='finished'){this.completion?.finish((s.finishedAt??s.deadlineAt)-s.startsAt,`Posisi ${s.karts.find(k=>k.actorId==='guest')?.position}/5`,s);if(this.timer!==null)clearInterval(this.timer);this.timer=null}return s;
 }

 input(controls: Controls, at: number) {
  this.engine?.tick(Math.floor(at));
  // Sequence lifetime matches the persistent race, not a mounted modal's timer.
  return this.engine?.input('guest', ++this.sequence, controls, Math.floor(at)) ?? false;
 }

 clear() {audio.stopMode('race');
  if(this.timer!==null)clearInterval(this.timer);this.timer=null;this.engine = null;this.completion=null;
  this.sequence = 0;
 }
}

export const guestRace = new GuestRaceSession();
