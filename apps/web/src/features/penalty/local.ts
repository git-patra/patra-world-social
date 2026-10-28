import type {Actor} from '../history/MatchDetail';
export type Turn={turn:number;kickerId:string;keeperId:string;kickZone:number;keepZone:number;kickTimeout:boolean;keepTimeout:boolean;goal:boolean};
export type DuelResult={participants:(Actor&{outcome:string;score:number})[];penaltyTurns:Turn[]};
export type PublicPenalty={matchId:string;phase:string;turnId:string;turn:number;kickerId:string;keeperId:string;choiceOpensAt:number;choiceDeadlineAt:number;animationEndsAt:number;actors:Actor[];scores:number[];chosen:boolean[];turns:Turn[];result?:DuelResult};
/** Guest practice uses a monotonic clock and the same deterministic fixture rules. */
export class LocalPenalty{
 private first:number;private turn=1;private opens:number;private choices=[0,0];private chosen=[false,false];private automatic=[false,false];private scores=[0,0];private kicks=[0,0];private turns:Turn[]=[];private revealed=false;private result?:DuelResult;
 constructor(private id:string,private actors:Actor[],at:number,private rng:(n:number)=>number){this.first=rng(2);this.opens=at;this.open()}
 private kicker(){return(this.first+this.turn-1)%2}
 private turnID(){return `${this.id}:${this.turn}`}
 private open(){this.chosen=[false,false];this.automatic=[false,false];this.revealed=false;this.actors.forEach((a,i)=>{if(a.actorType==='bot'){this.choices[i]=this.rng(4);this.chosen[i]=true}})}
 choose(actor:string,turn:string,zone:number,receivedAt:number){if(this.result||this.revealed||turn!==this.turnID()||!Number.isInteger(zone)||zone<0||zone>3||receivedAt<this.opens||receivedAt>=this.opens+3000)return false;const i=this.actors.findIndex(a=>a.actorId===actor&&a.actorType==='human');if(i<0||this.chosen[i])return false;this.choices[i]=zone;this.chosen[i]=true;return true}
 tick(now:number){while(!this.result){if(!this.revealed){if(now<this.opens+3000)break;for(let i=0;i<2;i++)if(!this.chosen[i]){this.choices[i]=this.rng(4);this.chosen[i]=true;this.automatic[i]=true};const k=this.kicker(),goal=this.choices[k]!==this.choices[1-k];if(goal)this.scores[k]++;this.kicks[k]++;this.turns.push({turn:this.turn,kickerId:this.actors[k].actorId,keeperId:this.actors[1-k].actorId,kickZone:this.choices[k],keepZone:this.choices[1-k],kickTimeout:this.automatic[k],keepTimeout:this.automatic[1-k],goal});this.revealed=true}
 if(now<this.opens+4500)break;
 const regular=this.kicks[0]<=5&&this.kicks[1]<=5&&(this.scores[0]>this.scores[1]+5-this.kicks[1]||this.scores[1]>this.scores[0]+5-this.kicks[0]);const paired=this.kicks[0]>=5&&this.kicks[0]===this.kicks[1]&&(this.scores[0]!==this.scores[1]||this.kicks[0]>=10);
 if(regular||paired){this.result={participants:this.actors.map((a,i)=>({...a,score:this.scores[i],outcome:this.scores[i]>this.scores[1-i]?'win':this.scores[i]<this.scores[1-i]?'loss':'draw'})),penaltyTurns:this.turns};break}this.turn++;this.opens+=4500;this.open()}}
 snapshot():PublicPenalty{const k=this.kicker();return structuredClone({matchId:this.id,phase:this.result?'results':this.revealed?'animation':'choosing',turnId:this.turnID(),turn:this.turn,kickerId:this.actors[k].actorId,keeperId:this.actors[1-k].actorId,choiceOpensAt:this.opens,choiceDeadlineAt:this.opens+3000,animationEndsAt:this.opens+4500,actors:this.actors,scores:this.scores,chosen:this.chosen,turns:this.turns,...(this.result?{result:this.result}:{})})}
}
