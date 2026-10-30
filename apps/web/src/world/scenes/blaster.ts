import {activityContract} from '../../activities/contracts';
import type {ActivityGame} from '../../activities/store';
import type {SceneActor} from '../../activities/scene';
import type {BlasterSnapshot} from '../../features/blaster/local';
const area=activityContract.areas.find(a=>a.id==='blaster')!;
export const blasterPoint=(x:number,y:number)=>({x:(area.bounds.x+x*area.bounds.width)*area.sceneTransform.scale+area.sceneTransform.offsetX,y:(area.bounds.y+y*area.bounds.height)*area.sceneTransform.scale+area.sceneTransform.offsetY});
/** Bounded actors/projectiles on the existing canvas; hit flashes expire in 120ms. */
export function sceneActors(game:ActivityGame,at:number,previous?:ActivityGame,fraction=1):SceneActor[]{
 const s=game.snapshot as BlasterSnapshot,old=previous?.snapshot as BlasterSnapshot|undefined;
 const blend=(a:{x:number;y:number}|undefined,b:{x:number;y:number})=>a&&old?.wave===s.wave?{x:a.x+(b.x-a.x)*fraction,y:a.y+(b.y-a.y)*fraction}:b;
 const actors:SceneActor[]=s.players.map((p,i)=>{const point=blasterPoint(...([blend(old?.players[i],p).x,blend(old?.players[i],p).y] as [number,number]));return{id:`blaster:${p.actorId}`,userId:s.actors[i]?.userId,...point,label:`${s.actors[i]?.displayName??''} · ${p.hp?'♥'.repeat(p.hp):'Down'}`,draw(ctx){ctx.save();ctx.translate(point.x,point.y);ctx.globalAlpha=p.hp?1:.35;ctx.fillStyle=at-p.hitAt>=0&&at-p.hitAt<120?'#fff5dc':['#e6be7a','#95b7a1','#bc9ec9','#da9476'][i];ctx.beginPath();ctx.arc(0,0,9,0,Math.PI*2);ctx.fill();ctx.strokeStyle='#293f35';ctx.lineWidth=3;ctx.stroke();ctx.strokeStyle='#fff0ce';ctx.lineWidth=3;const ax=Math.cos(p.aim),ay=Math.sin(p.aim);ctx.beginPath();ctx.moveTo(0,0);ctx.lineTo(ax*15,ay*15);ctx.stroke();if(p.hp){ctx.lineWidth=1;ctx.beginPath();ctx.arc(ax*28,ay*28,4,0,Math.PI*2);ctx.stroke()}ctx.restore()}}});
 for(const b of s.bots){const p=blend(old?.bots.find(o=>o.id===b.id),b),point=blasterPoint(p.x,p.y);actors.push({id:`blaster:bot:${b.id}`,...point,label:'',draw(ctx){ctx.fillStyle='#994b47';ctx.fillRect(point.x-7,point.y-7,14,14);ctx.fillStyle='#f3c980';ctx.fillRect(point.x-4,point.y-3,3,3);ctx.fillRect(point.x+2,point.y-3,3,3)}})}
 for(const b of s.projectiles){const p=blend(old?.projectiles.find(o=>o.id===b.id),b),point=blasterPoint(p.x,p.y);actors.push({id:`blaster:shot:${b.id}`,...point,label:'',draw(ctx){ctx.fillStyle=b.owner>=0?'#ffebad':'#ed8b7b';ctx.beginPath();ctx.arc(point.x,point.y,3,0,Math.PI*2);ctx.fill()}})}
 const origin=blasterPoint(0,0),end=blasterPoint(1,1);
 actors.unshift({id:'blaster:arena',x:origin.x,y:origin.y-1,label:'',draw(ctx){
  ctx.save();ctx.fillStyle='#68775a55';ctx.fillRect(origin.x,origin.y,end.x-origin.x,end.y-origin.y);
  ctx.strokeStyle='#ead2a0';ctx.lineWidth=3;ctx.strokeRect(origin.x+2,origin.y+2,end.x-origin.x-4,end.y-origin.y-4);
  ctx.strokeStyle='#425641';ctx.lineWidth=2;ctx.strokeRect(origin.x+6,origin.y+6,end.x-origin.x-12,end.y-origin.y-12);
  ctx.fillStyle='#6e523b';for(let x=origin.x+2;x<end.x;x+=40){ctx.fillRect(x-2,origin.y,4,8);ctx.fillRect(x-2,end.y-8,4,8)}for(let y=origin.y+40;y<end.y;y+=40){ctx.fillRect(origin.x,y-2,8,4);ctx.fillRect(end.x-8,y-2,8,4)}
  ctx.fillStyle='#293f35';ctx.fillRect(origin.x+75,origin.y+8,130,19);ctx.fillStyle='#f7e4ba';ctx.font='bold 11px sans-serif';ctx.textAlign='center';ctx.fillText('BLASTER · CO-OP',origin.x+140,origin.y+22);ctx.restore();
 }});
 return actors;
}
