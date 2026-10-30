import {activityContract,clawRules,type Claw} from '../../activities/contracts';
import type {SceneActor} from '../../activities/scene';

/** One public cabinet; animation is derived from the owner's phase timestamp for every viewer. */
export function clawActors(b:Claw,at:number):SceneActor[] {
 const p=activityContract.clawMachines[0].position,x=(p.x+.5)*40,y=(p.y+.5)*40-320;
 return [{id:'claw:claw-1',x,y,label:b.operatorId?`Claw · ${b.operatorName}`:'Claw · E',draw(ctx){
  const w=76,h=94,left=x-w/2,top=y-h;
  ctx.fillStyle='#344c4340';ctx.fillRect(left-3,y-3,w+6,9);
  ctx.fillStyle='#b6644c';ctx.fillRect(left,top,w,h);ctx.fillStyle='#f7d9a0';ctx.fillRect(left+4,top+4,w-8,12);
  ctx.font='bold 8px sans-serif';ctx.textAlign='center';ctx.fillStyle='#4f4835';ctx.fillText('CAPIT BONEKA',x,top+13);
  ctx.fillStyle='#c5dcce';ctx.fillRect(left+5,top+20,w-10,55);
  const bx=left+10,by=top+25,bw=w-20,bh=43;
  const plush=(id:string,px:number,py:number)=>{ctx.fillStyle=id==='bear'?'#cf945d':id==='bunny'?'#f2dbe0':'#dd8056';ctx.beginPath();ctx.arc(px-3,py-4,2.5,0,Math.PI*2);ctx.arc(px+3,py-4,2.5,0,Math.PI*2);ctx.arc(px,py,5,0,Math.PI*2);ctx.fill();ctx.fillStyle='#514b43';ctx.fillRect(px-2,py-1,1,1);ctx.fillRect(px+1,py-1,1,1)};
  for(const preset of clawRules.presets)if(!(b.prizeId===preset.id&&(b.phase==='lifting'||b.phase==='releasing')))plush(preset.id,bx+preset.x*bw,by+preset.y*bh);
  const elapsed=Math.max(0,at-b.phaseAt),phase=clawRules.phaseMs;
  const lift=b.phase==='dropping'?Math.min(1,elapsed/phase):b.phase==='lifting'?1-Math.min(1,(elapsed-phase)/phase):0;
  const release=b.phase==='releasing'?Math.min(1,(elapsed-2*phase)/phase):0;
  const aimX=bx+b.x*bw,aimY=by+b.y*bh;
  if(b.phase==='aiming'){ctx.strokeStyle='#f7d9a0';ctx.lineWidth=1;ctx.beginPath();ctx.arc(aimX,aimY,8,0,Math.PI*2);ctx.stroke()}
  const gripX=aimX+(left+13-aimX)*release,gripY=aimY-6+lift*6;
  ctx.strokeStyle='#61786e';ctx.lineWidth=2;ctx.beginPath();ctx.moveTo(gripX,top+19);ctx.lineTo(gripX,gripY);ctx.moveTo(gripX-6,gripY);ctx.lineTo(gripX-3,gripY+6);ctx.lineTo(gripX,gripY+3);ctx.lineTo(gripX+3,gripY+6);ctx.lineTo(gripX+6,gripY);ctx.stroke();
  if(b.prizeId&&(b.phase==='lifting'||b.phase==='releasing'))plush(b.prizeId,gripX,gripY+release*(top+85-gripY));
  ctx.fillStyle='#4b6356';ctx.fillRect(left+6,top+80,23,10);ctx.fillStyle=b.operatorId?'#e5ba65':'#b5dc9b';ctx.beginPath();ctx.arc(left+55,top+84,4,0,Math.PI*2);ctx.fill();
 }}];
}
