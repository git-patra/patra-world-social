import {clawRules, activityContract, type Claw} from './contracts';
export const clawSpawn=(dropId=0):Claw=>({operatorId:'',operatorName:'',leaseUntil:0,x:.5,y:.5,phase:'idle',phaseAt:0,dropId,prizeId:''});
export const nearClaw=(p:{x:number;y:number})=>{const e=activityContract.clawMachines[0].entrance;return Math.hypot(p.x-e.x,p.y-e.y)<=2};
/** Guest simulation runs from ActivityStore's existing clock; prizes are transient cosmetics. */
export class LocalClaw {
 claw=clawSpawn();revision=1;private at=0;private dx=0;private dy=0;private expires=0;
 reset(at=this.at){this.claw=clawSpawn(this.claw.dropId);this.dx=this.dy=0;this.at=at;this.revision++}
 enter(p:{x:number;y:number},at:number){if(this.claw.operatorId||!nearClaw(p))return false;this.reset(at);this.claw={...this.claw,operatorId:'guest',operatorName:'Guest',leaseUntil:at+clawRules.leaseMs,phase:'aiming'};return true}
 controls(dx:number,dy:number,at:number){if(this.claw.phase!=='aiming'||!Number.isFinite(dx)||!Number.isFinite(dy)||Math.abs(dx)>1||Math.abs(dy)>1)return;this.dx=dx;this.dy=dy;this.expires=at+clawRules.inputExpiryMs}
 drop(at:number){this.tick(at);const b=this.claw;if(b.phase!=='aiming')return false;this.dx=this.dy=0;this.claw={...b,phase:'dropping',phaseAt:at,dropId:b.dropId+1,prizeId:clawRules.presets.find(p=>Math.hypot(p.x-b.x,p.y-b.y)<=clawRules.gripTolerance)?.id??''};this.revision++;return true}
 tick(at:number){const b={...this.claw};if(!b.operatorId)return;if(at>=b.leaseUntil){this.reset(at);return};if(b.phase!=='aiming'){const elapsed=at-b.phaseAt;b.phase=elapsed>=3*clawRules.phaseMs?'aiming':elapsed>=2*clawRules.phaseMs?'releasing':elapsed>=clawRules.phaseMs?'lifting':'dropping';if(b.phase==='aiming')b.prizeId='';this.at=at}else{let steps=0;while(this.at+clawRules.stepMs<=at&&steps++<clawRules.maxCatchupSteps){this.at+=clawRules.stepMs;if(this.at>this.expires)continue;const speed=clawRules.speed*clawRules.stepMs/1000/Math.max(1,Math.hypot(this.dx,this.dy));b.x=Math.max(0,Math.min(1,b.x+this.dx*speed));b.y=Math.max(0,Math.min(1,b.y+this.dy*speed))};if(this.at+clawRules.stepMs<=at)this.at=at};this.claw=b}
}
