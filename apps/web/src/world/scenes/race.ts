import areas from '../../../../../contracts/activities.v1.json';
import type {RaceSnapshot} from '../../features/race/local';
import type {ActivityGame} from '../../activities/store';
import type {SceneActor} from '../../activities/scene';
const transform=areas.areas.find(a=>a.id==='race')!.sceneTransform;
const colors=['#e7a74a','#8caf82','#bc8dcb','#73afc2','#dd7971'];
export const racePoint=(point:{x:number;y:number})=>({x:transform.offsetX+point.x*transform.scale,y:transform.offsetY+point.y*transform.scale});
export function sceneActors(game:ActivityGame,_at:number,previous?:ActivityGame,fraction=1):SceneActor[]{return(game.snapshot as RaceSnapshot).karts.map(k=>{const old=(previous?.snapshot as RaceSnapshot|undefined)?.karts.find(a=>a.actorId===k.actorId)??k;const heading=old.heading+((k.heading-old.heading+540)%360-180)*fraction,p=racePoint({x:old.x+(k.x-old.x)*fraction,y:old.y+(k.y-old.y)*fraction});return{id:`race:${k.actorId}`,userId:k.userId,...p,label:`${k.position}. ${k.displayName}`,draw(ctx){ctx.save();ctx.translate(p.x,p.y);ctx.rotate(heading*Math.PI/180);ctx.fillStyle='#292a2d';ctx.fillRect(-8,-7,15,4);ctx.fillRect(-8,3,15,4);ctx.fillStyle=colors[k.startSlot%colors.length];ctx.beginPath();ctx.roundRect(-9,-5,18,10,3);ctx.fill();ctx.fillStyle='#343137';ctx.fillRect(-2,-4,6,8);ctx.fillStyle='#ffe6af';ctx.beginPath();ctx.moveTo(8,-3);ctx.lineTo(12,0);ctx.lineTo(8,3);ctx.fill();ctx.restore()}}})}
