import mapData from '../../../../contracts/map.v1.json';
import {loadMap,collisionTiles,type Point} from './map';
import {findPath,isPlayable,tileKey} from '../world-math.js';
import {advanceMovement} from './input';
import {validateAvatars,type Avatar} from './avatars';
import type {Snapshot} from '../net/reconcile';
import {TABLES} from '../features/voice/zones';
import {createWorldArt,drawCharacter} from './village-art';
import {VILLAGE_DESTINATIONS,projectVillage,unprojectVillage,villageLocation} from './village';

const MAP=loadMap(mapData),occupied=collisionTiles(MAP);
const clamp=(v:number,a:number,b:number)=>Math.max(a,Math.min(b,v));
type Adapter={input:(dx:number,dy:number)=>void;destination:(x:number,y:number)=>void};
type Options={stations:{id:string;name?:string}[];onNear?:(id:string|null)=>void;onSelect?:(id:string)=>void;onMove?:(position:Point&{walking:boolean})=>void};

/** Screen axes and authoritative grid axes match. Rounded cells mirror Go's movement boundary. */
export function freeMovement(position:Point,direction:Point,dt:number):Point{
 const magnitude=Math.hypot(direction.x,direction.y);if(!magnitude)return {...position};
 const step=Math.max(0,Math.min(dt,.05))*4.5;
 const next={x:position.x+direction.x/Math.max(1,magnitude)*step,y:position.y+direction.y/Math.max(1,magnitude)*step};
 const blocked=(x:number,y:number)=>x<0||y<0||x>63||y>63||occupied.has(tileKey(Math.round(x),Math.round(y)));
 return blocked(next.x,next.y)||blocked(next.x,position.y)||blocked(position.x,next.y)?{...position}:next;
}

/** Largest foreground spans 350 art pixels from its depth anchor (Warkop canopy).
 * Keep a conservative art-space margin, scaled with zoom, for every prop. */
export function visibleVillageProp(point:Point,camera:Point,zoom:number,viewport:{width:number;height:number}){
 const x=camera.x+point.x*zoom,y=camera.y+point.y*zoom,padding=400*zoom;
 return x>-padding&&x<viewport.width+padding&&y>-padding&&y<viewport.height+padding;
}

export function createWorld(canvas:HTMLCanvasElement,{stations,onNear=()=>{},onSelect=()=>{},onMove=()=>{}}:Options){
 const ctx=canvas.getContext('2d')!,art=createWorldArt();
 const mini=document.createElement('canvas');mini.width=256;mini.height=192;
 const mc=mini.getContext('2d')!;mc.drawImage(art.background,0,0,256,192);mc.save();mc.scale(.1,.1);[...art.foregroundObjects].sort((a,b)=>a.y-b.y).forEach(o=>o.draw(mc,0));mc.restore();
 let player:Point={...MAP.spawn},soloPosition:Point|null=null,direction:Point={x:0,y:0},route:Point[]=[],target:Point|null=null;
 let online:Adapter|null=null,publicState:{avatars:Avatar[]}={avatars:[]},selfID:string|null=null,connectionStatus='solo';
 let localName='Guest',localAvatar='pumpkin',paused=false,motion=true,dead=false,dirty=true,near:string|null=null,facing='down';
 let width=0,height=0,dpr=1,zoom=1,lastTime=0,elapsed=0,raf=0,lastMove='',walking=false;
 let camera={x:0,y:0},pointerStart:{x:number;y:number;id:number}|null=null;
 const previous=new Map<string,Point>();
 const positions=VILLAGE_DESTINATIONS;
 function resize(){const bounds=canvas.getBoundingClientRect();width=bounds.width;height=bounds.height;dpr=Math.min(window.devicePixelRatio||1,2);canvas.width=Math.round(width*dpr);canvas.height=Math.round(height*dpr);ctx.imageSmoothingEnabled=false;dirty=true;draw()}
 function label(text:string,x:number,y:number,own=false){ctx.font='600 11px "DM Sans",sans-serif';const w=ctx.measureText(text).width+16;ctx.fillStyle=own?'#f5e5bd':'#243b32ed';ctx.beginPath();ctx.roundRect(x-w/2,y-16,w,22,6);ctx.fill();ctx.fillStyle=own?'#304732':'#fcf1d4';ctx.textAlign='center';ctx.fillText(text,x,y-1)}
 function draw(){
  const p=projectVillage(player);camera={x:width/2-p.x*zoom,y:height/2-p.y*zoom};
  ctx.setTransform(dpr,0,0,dpr,0,0);ctx.fillStyle='#7f956b';ctx.fillRect(0,0,width,height);ctx.save();ctx.translate(camera.x,camera.y);ctx.scale(zoom,zoom);ctx.imageSmoothingEnabled=false;ctx.drawImage(art.background,0,0);
  if(route.length){ctx.strokeStyle='#fff4c480';ctx.lineWidth=3;ctx.setLineDash([3,9]);ctx.beginPath();ctx.moveTo(p.x,p.y);for(const tile of route){const r=projectVillage(tile);ctx.lineTo(r.x,r.y)}ctx.stroke();ctx.setLineDash([])}
  if(target){const t=projectVillage(target);ctx.strokeStyle='#ffe7a8';ctx.lineWidth=2;ctx.beginPath();ctx.arc(t.x,t.y,10,0,Math.PI*2);ctx.stroke()}
  for(const t of TABLES.tables){const c=projectVillage(t);ctx.strokeStyle='#f3d49670';ctx.lineWidth=1;ctx.setLineDash([4,6]);ctx.beginPath();ctx.arc(c.x,c.y,t.radius*40,0,Math.PI*2);ctx.stroke();ctx.setLineDash([])}
  const avatars=online?publicState.avatars:[{...player,id:'local',name:localName,avatarId:localAvatar,status:'available' as const}];
  const objects=art.foregroundObjects.filter(o=>visibleVillageProp(o,camera,zoom,{width,height})).map(o=>({depth:o.y,draw:()=>o.draw(ctx,motion?elapsed*1000:0)}));
  for(const avatar of avatars){const at=projectVillage(avatar),old=previous.get(avatar.id);const moving=old&&Math.hypot(old.x-avatar.x,old.y-avatar.y)>.001;const dx=old?avatar.x-old.x:0,dy=old?avatar.y-old.y:0;
   const own=avatar.id==='local'||avatar.id===selfID;const look=own?facing:moving?(Math.abs(dx)>Math.abs(dy)?dx>0?'right':'left':dy>0?'down':'up'):'down';
   objects.push({depth:at.y,draw:()=>drawCharacter(ctx,{...at,name:'',color:avatar.avatarId==='moon'?'#b3a0cf':avatar.avatarId==='autumn'?'#a4ba83':'#e5bd85',isPlayer:own,facing:look,walking:motion&&(own?walking:Boolean(moving)),seated:false},elapsed*1000)});previous.set(avatar.id,{x:avatar.x,y:avatar.y});
  }
  objects.sort((a,b)=>a.depth-b.depth).forEach(o=>o.draw());
  ctx.restore();const labels:{x:number;y:number;w:number}[]=[];
  for(const a of avatars){const p=projectVillage(a);const x=camera.x+p.x*zoom;let y=camera.y+(p.y-53)*zoom;const text=`${a.name}${a.status==='available'?'':` · ${a.status}`}`;ctx.font='600 11px "DM Sans",sans-serif';const w=ctx.measureText(text).width+16;while(labels.some(l=>Math.abs(l.x-x)<(w+l.w)/2+4&&Math.abs(l.y-y)<24))y-=24;labels.push({x,y,w});label(text,x,y,a.id==='local'||a.id===selfID)}
  const compact=width<700,mw=compact?104:160,mh=mw*.75,mx=width-mw-18,my=height-(compact?205:155)-mh;
  ctx.fillStyle='#213b32ee';ctx.beginPath();ctx.roundRect(mx-7,my-23,mw+14,mh+34,10);ctx.fill();ctx.drawImage(mini,mx,my,mw,mh);ctx.font='8px "DM Sans",sans-serif';ctx.fillStyle='#e5e8d1';ctx.textAlign='left';ctx.fillText('KAMPUNG SENJA',mx,my-9);
  ctx.fillStyle='#ffe4aa';ctx.beginPath();ctx.arc(mx+p.x/art.width*mw,my+p.y/art.height*mh,3,0,Math.PI*2);ctx.fill();dirty=false;
 }
 function notifyPosition(){
  const signature=`${player.x.toFixed(2)},${player.y.toFixed(2)},${walking}`;
  if(signature!==lastMove){lastMove=signature;onMove({...player,walking})}
  const table=TABLES.tables.some(t=>Math.hypot(player.x-t.x,player.y-t.y)<=t.radius+1);
  const closest=table?'warkop':positions.map(s=>({id:s.id,d:Math.hypot(player.x-s.x,player.y-s.y)})).filter(s=>s.d<3.5).sort((a,b)=>a.d-b.d)[0]?.id??null;
  if(closest!==near){near=closest;onNear(near)}
 }
 function stopDirection(){direction={x:0,y:0};online?.input(0,0);walking=false}
 function destination(point:Point){
  if(paused||!isPlayable(point.x,point.y)||occupied.has(tileKey(point.x,point.y)))return;
  direction={x:0,y:0};target={...point};
  if(online){online.destination(point.x,point.y);return}
  const start={x:Math.round(player.x),y:Math.round(player.y)},path=findPath(start,point,occupied);
  route=path.length?[start,...path]:[];if(!route.length)target=null;dirty=true;
 }
 function travelTo(id:string){const place=positions.find(s=>s.id===id);if(place)destination(place)}
 function move(dx:number,dy:number){if(paused||!Number.isFinite(dx)||!Number.isFinite(dy))return;direction={x:Math.sign(dx),y:Math.sign(dy)};route=[];target=null;if(online)online.input(direction.x,direction.y);dirty=true}
 function interact(){if(paused||!near)return;stopDirection();route=[];target=null;if(stations.some(s=>s.id===near))onSelect(near)}
 const pointer=(e:PointerEvent)=>{const r=canvas.getBoundingClientRect();return{x:e.clientX-r.left,y:e.clientY-r.top}};
 function pointerDown(e:PointerEvent){if(e.button!==undefined&&e.button!==0)return;pointerStart={...pointer(e),id:e.pointerId};canvas.setPointerCapture?.(e.pointerId)}
 function pointerUp(e:PointerEvent){const p=pointer(e),start=pointerStart;pointerStart=null;if(!start||start.id!==e.pointerId||Math.hypot(p.x-start.x,p.y-start.y)>10)return;const tile=unprojectVillage({x:(p.x-camera.x)/zoom,y:(p.y-camera.y)/zoom});destination({x:Math.round(tile.x),y:Math.round(tile.y)});canvas.focus?.({preventScroll:true})}
 function cancelPointer(){pointerStart=null}
 function frame(timestamp:number){
  if(dead)return;const dt=Math.min((timestamp-(lastTime||timestamp))/1000,.05);lastTime=timestamp;
  if(!paused){if(motion)elapsed+=dt;if(!online){const before={...player};if(direction.x||direction.y)player=freeMovement(player,direction,dt);else if(route.length){player=advanceMovement(player,route[0],dt);if(Math.hypot(player.x-route[0].x,player.y-route[0].y)<.0001){route.shift();if(!route.length)target=null}}
    walking=Math.hypot(player.x-before.x,player.y-before.y)>.0001;const dx=player.x-before.x,dy=player.y-before.y;if(walking)facing=Math.abs(dx)>Math.abs(dy)?dx>0?'right':'left':dy>0?'down':'up';if(walking)dirty=true;notifyPosition();
   }}
  if(dirty||motion&&!paused||route.length)draw();raf=requestAnimationFrame(frame);
 }
 const observer=new ResizeObserver(resize);observer.observe(canvas);
 canvas.addEventListener('pointerdown',pointerDown);canvas.addEventListener('pointerup',pointerUp);canvas.addEventListener('pointercancel',cancelPointer);
 resize();notifyPosition();raf=requestAnimationFrame(frame);
 return {
  travelTo,travelToLocation:travelTo,move,interact,
  drawMap(targetCanvas:HTMLCanvasElement){const c=targetCanvas.getContext('2d')!;c.clearRect(0,0,targetCanvas.width,targetCanvas.height);c.save();c.scale(targetCanvas.width/art.width,targetCanvas.height/art.height);c.drawImage(art.background,0,0);[...art.foregroundObjects].sort((a,b)=>a.y-b.y).forEach(o=>o.draw(c,0));c.restore();for(const d of positions){if(d.id==='arcade')continue;const p=projectVillage(d),x=p.x/art.width*targetCanvas.width,y=p.y/art.height*targetCanvas.height;c.font='600 12px sans-serif';const w=c.measureText(d.name).width+12;c.fillStyle='#213b32ed';c.fillRect(x-w/2,y-22,w,20);c.fillStyle='#fff0cd';c.textAlign='center';c.fillText(d.name,x,y-8)}},
  stopInput(){stopDirection();dirty=true},
  setConnectionStatus(status:string){connectionStatus=status;dirty=true},
  isOnline(){return online!==null},
  setLocalIdentity(name:string,avatar:string){localName=name;localAvatar=avatar;dirty=true},
  setOnline(adapter:Adapter|null){if(adapter&&!online)soloPosition={...player};online=adapter;direction={x:0,y:0};route=[];target=null;publicState={avatars:[]};selfID=null;previous.clear();walking=false;if(!adapter&&soloPosition){player={...soloPosition};soloPosition=null}dirty=true;notifyPosition()},
  setOnlineSnapshot(authoritative:Snapshot,interpolated:Snapshot){if(!online)return;const state=validateAvatars(interpolated),self=authoritative.avatars.find(a=>a.id===authoritative.selfId);if(!self)return;const dx=self.x-player.x,dy=self.y-player.y;walking=Math.hypot(dx,dy)>.001;if(walking)facing=Math.abs(dx)>Math.abs(dy)?dx>0?'right':'left':dy>0?'down':'up';player={x:self.x,y:self.y};selfID=authoritative.selfId;publicState={avatars:state.avatars.map(a=>a.id===selfID?{...self}:a)};if(target&&Math.hypot(player.x-target.x,player.y-target.y)<.2)target=null;dirty=true;notifyPosition()},
  setPublicState(state:{avatars:Avatar[]}){publicState=validateAvatars(state);dirty=true},
  getState(){return{player:{...player},route:route.map(p=>({...p})),mapId:MAP.id,location:villageLocation(player),connectionStatus}},
  setZoom(value:number){zoom=Number.isFinite(value)?clamp(value,.65,1.5):1;dirty=true;draw()},
  recenter(){zoom=1;dirty=true;draw()},
  setPaused(value:boolean){paused=Boolean(value);if(paused){stopDirection();cancelPointer()}dirty=true},
  setMotion(value:boolean){motion=Boolean(value);dirty=true},
  reset(){stopDirection();route=[];target=null;if(!online)player={...MAP.spawn};dirty=true;notifyPosition()},
  destroy(){dead=true;stopDirection();cancelAnimationFrame(raf);observer.disconnect();canvas.removeEventListener('pointerdown',pointerDown);canvas.removeEventListener('pointerup',pointerUp);canvas.removeEventListener('pointercancel',cancelPointer)},
 };
}
