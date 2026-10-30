import {activityContract,type FishingView} from '../../activities/contracts';
import {fishingCatchLabels} from '../../activities/fishing';
import type {SceneActor} from '../../activities/scene';
/** Four anchored dock/bobber actors, using the world's existing RAF and public timestamps. */
export function fishingActors(spots:FishingView[],at:number):SceneActor[]{return spots.map(s=>{
 const p=activityContract.fishingSpots.find(p=>p.id===s.id)!.position,x=(p.x+.5)*40,y=(p.y+.5)*40-320,f=s.fishing,visible=f.phase!=='result'||at<f.endsAt;
 return{id:`fishing:${s.id}`,x,y,label:f.phase==='idle'?'Fishing · E':!visible?'':f.phase==='result'?(fishingCatchLabels[f.catchId]??'Lepas! Coba lagi'):f.phase==='bite'?'Bite! · Reel / Space':`${f.ownerName} · tunggu…`,draw(ctx){
 ctx.fillStyle='#ae8053';ctx.fillRect(x-18,y-8,39,19);ctx.strokeStyle='#6d513a';ctx.lineWidth=1;for(let n=0;n<4;n++){ctx.beginPath();ctx.moveTo(x-18,y-6+n*6);ctx.lineTo(x+21,y-6+n*6);ctx.stroke()}
 if(!f.ownerId||!visible)return;
 const bx=x+38,by=y+5+(f.phase==='bite'?Math.sin(at/65)*3:Math.sin(at/350));ctx.strokeStyle='#e7dab0';ctx.beginPath();ctx.moveTo(x-8,y-18);ctx.lineTo(x+12,y-44);ctx.lineTo(bx,by);ctx.stroke();ctx.fillStyle=f.phase==='bite'?'#ffdc6b':'#e68455';ctx.beginPath();ctx.arc(bx,by,4,0,Math.PI*2);ctx.fill();
 if(f.phase==='bite'){ctx.strokeStyle='#ffe9a6';ctx.beginPath();ctx.ellipse(bx,by+2,10,4,0,0,Math.PI*2);ctx.stroke()}
 if(f.phase==='result'){ctx.font='22px sans-serif';ctx.textAlign='center';ctx.fillText(({fish:'🐟',boot:'👢',can:'🥫',leaf:'🍂'} as Record<string,string>)[f.catchId]??'💧',x,y-25)}
 }}
})}
