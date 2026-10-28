import {audio} from '../../audio/service';
import {gameAudio} from '../../audio/events';
import {Completion} from '../../guest/completion';
import {preferences} from '../../guest/history';
import {LocalPenalty} from './local';
// Modal lifetime is independent of guest match lifetime; reopen catches up by time.
export class GuestPenaltySession{
 private engine:LocalPenalty|null=null;private completion:Completion|null=null;private started=0;private timer:ReturnType<typeof setInterval>|null=null;
 start(at:number,rng:(n:number)=>number=n=>Math.floor(Math.random()*n),id:string=crypto.randomUUID()){
 this.clear();this.started=at;this.completion=new Completion(id,'penalty');if(typeof window!=='undefined')this.timer=setInterval(()=>this.snapshot(performance.now()),100);
 this.engine=new LocalPenalty(id,[{actorId:'guest',actorType:'human',userId:'guest',displayName:preferences().displayName,avatarId:preferences().avatar},{actorId:'bot',actorType:'bot',displayName:'BOT',avatarId:'pumpkin'}],at,rng);return this.snapshot(at)
 }
 snapshot(at:number){if(!this.engine)return null;this.engine.tick(at);const s=this.engine.snapshot();gameAudio.observe('penalty',s,at);if(s.result){this.completion?.finish(s.animationEndsAt-this.started,`${s.scores[0]} : ${s.scores[1]}`,s);if(this.timer!==null)clearInterval(this.timer);this.timer=null}return s}
 choose(turn:string,zone:number,at:number){if(!this.engine)return false;this.engine.tick(at);return this.engine.choose('guest',turn,zone,at)}
 clear(){audio.stopMode('penalty');if(this.timer!==null)clearInterval(this.timer);this.timer=null;this.engine=null;this.completion=null}
}
export const guestSession=new GuestPenaltySession();
