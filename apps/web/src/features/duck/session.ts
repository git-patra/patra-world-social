import {LocalDuck,type Aim} from './local';
import {Completion} from '../../guest/completion';
import {preferences} from '../../guest/history';
export class GuestDuckSession{
 private engine:LocalDuck|null=null;private completion:Completion|null=null;private id='';private timer:ReturnType<typeof setInterval>|null=null;
 start(at:number){this.clear();this.id=crypto.randomUUID();this.completion=new Completion(this.id,'duck');this.engine=new LocalDuck(at+3000,Math.floor(at)||1);if(typeof window!=='undefined')this.timer=setInterval(()=>this.snapshot(performance.now()),100);return this.snapshot(at)!}
 snapshot(at:number){this.engine?.tick(at);const s=this.engine?.snapshot()??null;if(s){s.matchId=this.id;s.actor.displayName=preferences().displayName;if(s.phase==='finished'){this.completion?.finish(s.deadlineAt-s.startsAt,`Skor ${s.score} · ${s.hits}/${s.shots} hit`,s);if(this.timer!==null)clearInterval(this.timer);this.timer=null}}return s}
 shoot(aim:Aim,at:number){return this.engine?.shoot(aim,at)??null}
 clear(){if(this.timer!==null)clearInterval(this.timer);this.timer=null;this.engine=null;this.completion=null}
}
export const guestDuck=new GuestDuckSession();
