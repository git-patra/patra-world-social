import type { MapDefinition, Point } from './map';
import type { PublicState, Avatar } from './avatars';
export interface RenderOptions {
    objects: {
        depth: number;
        draw: () => void;
    }[];
    paintAvatar: (avatar: Avatar) => void;
}
/** Paint callbacks preserve the original procedural art. Avatar positions come only from public state. */
export function drawWorld(canvas: HTMLCanvasElement, map: MapDefinition, publicState: PublicState, options: RenderOptions) { if (!canvas || map.v !== 1 || !publicState)
    throw Error('Invalid render state'); [...options.objects, ...publicState.avatars.map(avatar => ({ depth: avatar.x + avatar.y + .3, draw: () => options.paintAvatar(avatar) }))].sort((a, b) => a.depth - b.depth).forEach(object => object.draw()); }
export function drawMinimap(ctx: CanvasRenderingContext2D, map: MapDefinition, player: Point, width: number, mode='local solo') { const left = width - 126, top = 110; ctx.fillStyle = '#201b2ce8'; ctx.fillRect(left - 6, top - 6, 120, 134); ctx.fillStyle = '#8e7766'; ctx.fillRect(left, top, 108, 108); for (const d of map.districts) {
    ctx.fillStyle = '#57414c';
    ctx.fillRect(left + d.x / 64 * 108, top + d.y / 64 * 108, d.width / 64 * 108, d.height / 64 * 108);
} ctx.fillStyle = '#ffcf89'; ctx.beginPath(); ctx.arc(left + player.x / 64 * 108, top + player.y / 64 * 108, 3, 0, Math.PI * 2); ctx.fill(); ctx.font = '10px sans-serif'; ctx.textAlign = 'left'; ctx.fillStyle = '#ffecd6'; ctx.fillText(`64 × 64 · ${mode}`, left, top + 123); }

interface Nameplate {avatar:Avatar;x:number;y:number;anchorY:number;width:number;text:string}
/** Stack colliding labels upwards; positions remain independent of sprite depth or fixture IDs. */
export function layoutNameplates(avatars:Avatar[],measure:(text:string)=>number):Nameplate[]{
 const labels:Nameplate[]=[];
 for(const avatar of [...avatars].sort((a,b)=>a.x+a.y-b.x-b.y||a.id.localeCompare(b.id))){
  const text=`${avatar.name} · ${avatar.status}`,x=(avatar.x-avatar.y)*30,anchorY=(avatar.x+avatar.y)*15-32;
  let y=anchorY-22;const width=measure(text)+10;
  while(labels.some(label=>Math.abs(label.x-x)<(label.width+width)/2+4&&Math.abs(label.y-y)<16))y-=18;
  labels.push({avatar,x,y,anchorY,width,text});
 }
 return labels;
}
export function drawNameplates(ctx:CanvasRenderingContext2D,avatars:Avatar[]){
 ctx.save();ctx.font='9px sans-serif';ctx.textAlign='center';
 for(const label of layoutNameplates(avatars,text=>ctx.measureText(text).width)){
  ctx.strokeStyle='#d9b48a88';ctx.lineWidth=.8;ctx.beginPath();ctx.moveTo(label.x,label.anchorY);ctx.lineTo(label.x,label.y+5);ctx.stroke();
  ctx.fillStyle='#211b2eed';ctx.fillRect(label.x-label.width/2,label.y-9,label.width,14);
  ctx.fillStyle='#ffecd6';ctx.fillText(label.text,label.x,label.y+1);
 }
 ctx.restore();
}
