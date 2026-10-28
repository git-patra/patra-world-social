import areas from '../../../../../contracts/activities.v1.json';
import type {PublicPenalty} from '../../features/penalty/local';
import type {ActivityGame} from '../../activities/store';
import type {SceneActor} from '../../activities/scene';
const transform=areas.areas.find(a=>a.id==='penalty')!.sceneTransform;
export function sceneActors(game:ActivityGame,at:number):SceneActor[]{
 const s=game.snapshot as PublicPenalty,x=transform.offsetX+160,top=transform.offsetY+30,kickY=transform.offsetY+230;
 const last=s.phase==='choosing'?undefined:s.turns.at(-1),progress=last?Math.max(0,Math.min(1,(at-s.choiceDeadlineAt)/750)):0;
 const actors:SceneActor[]=s.actors.map(a=>{const keeper=a.actorId===s.keeperId,zone=last?.keepZone??0,dx=keeper&&last?(zone%2?28:-28)*progress:0,p={x:x+dx,y:keeper?top:kickY};return{id:`penalty:${a.actorId}`,userId:a.userId,...p,label:a.displayName,draw(ctx){ctx.fillStyle='#243c3540';ctx.beginPath();ctx.ellipse(p.x,p.y,12,4,0,0,Math.PI*2);ctx.fill();ctx.fillStyle=keeper?'#dca54c':'#71a9bc';ctx.fillRect(p.x-7,p.y-20,14,16);ctx.fillStyle='#ebc59a';ctx.beginPath();ctx.arc(p.x,p.y-25,6,0,Math.PI*2);ctx.fill();ctx.fillStyle='#344c43';ctx.fillRect(p.x-6,p.y-5,4,7);ctx.fillRect(p.x+2,p.y-5,4,7)}}});
 const targetX=x+(last?(last.kickZone%2?36:-36):0),targetY=top-(last&&last.kickZone<2?30:8),ball={x:x+(targetX-x)*progress,y:kickY-12+(targetY-(kickY-12))*progress};
 actors.push({id:'penalty:ball',...ball,label:'',draw(ctx){ctx.fillStyle='#fff0cc';ctx.beginPath();ctx.arc(ball.x,ball.y,5,0,Math.PI*2);ctx.fill();ctx.fillStyle='#405847';ctx.fillRect(ball.x-2,ball.y-2,3,3)}});return actors;
}
