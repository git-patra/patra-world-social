import {activityContract} from '../activities/contracts';
import {ActivityScene} from '../activities/scene';
import {nearFootball} from '../activities/football';
import {activities,type ActivityState} from '../activities/store';
import mapData from '../../../../contracts/map.v1.json';
import {mapPoint,type CanvasRect,type ClosedMapZone} from './minimap';
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
type DestinationResolver=(point:Point)=>Point|null;
type Options={minimapCanvas?:HTMLCanvasElement;resolveDestination?:DestinationResolver;stations:{id:string;name?:string}[];onNear?:(id:string|null)=>void;onSelect?:(id:string)=>void;onMove?:(position:Point&{walking:boolean})=>void};

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

export function createWorld(canvas:HTMLCanvasElement,{stations,minimapCanvas,resolveDestination,onNear=()=>{},onSelect=()=>{},onMove=()=>{}}:Options){
 const ctx=canvas.getContext('2d')!,art=createWorldArt();
 const mini=document.createElement('canvas');mini.width=256;mini.height=192;
 const mc=mini.getContext('2d')!;mc.drawImage(art.background,0,0,256,192);mc.save();mc.scale(.1,.1);[...art.foregroundObjects].sort((a,b)=>a.y-b.y).forEach(o=>o.draw(mc,0));mc.restore();
 let player:Point={...MAP.spawn},soloPosition:Point|null=null,direction:Point={x:0,y:0},route:Point[]=[],target:Point|null=null;
 let online:Adapter|null=null,publicState:{avatars:Avatar[]}={avatars:[]},selfID:string|null=null,connectionStatus='solo';
 let localName='Guest',localAvatar='pumpkin',paused=false,motion=true,dead=false,dirty=true,near:string|null=null,facing='down';
 let width=0,height=0,dpr=1,zoom=1,lastTime=0,elapsed=0,raf=0,lastMove='',walking=false;
 let camera={x:0,y:0},pointerStart:{x:number;y:number;id:number}|null=null;
 let activityOwner=false,lastActivityKind='',displayPlayer:Point|null=null;
 let closedZones:ClosedMapZone[]=[];let onlineSampler:(()=>void)|undefined;
 const previous=new Map<string,Point>();
 const activityScene=new ActivityScene(()=>{dirty=true});
 const positions=[...VILLAGE_DESTINATIONS,{id:"pingpong",name:"Meja Pingpong",x:45,y:25}];
 function resize(){const bounds=canvas.getBoundingClientRect();width=bounds.width;height=bounds.height;dpr=Math.min(window.devicePixelRatio||1,2);canvas.width=Math.round(width*dpr);canvas.height=Math.round(height*dpr);ctx.imageSmoothingEnabled=false;dirty=true;draw()}
 function label(text:string,x:number,y:number,own=false){ctx.font='600 11px "DM Sans",sans-serif';const w=ctx.measureText(text).width+16;ctx.fillStyle=own?'#f5e5bd':'#243b32ed';ctx.beginPath();ctx.roundRect(x-w/2,y-16,w,22,6);ctx.fill();ctx.fillStyle=own?'#304732':'#fcf1d4';ctx.textAlign='center';ctx.fillText(text,x,y-1)}
 function draw(){
  const activityActors=activityScene.actors(performance.now()),ownActor=activityScene.ownActor(activityActors);
  displayPlayer=ownActor?unprojectVillage(ownActor):null;if(ownActor)notifyPosition();
  const p=ownActor??projectVillage(player);camera={x:width/2-p.x*zoom,y:height/2-p.y*zoom};
  ctx.setTransform(dpr,0,0,dpr,0,0);ctx.fillStyle='#7f956b';ctx.fillRect(0,0,width,height);ctx.save();ctx.translate(camera.x,camera.y);ctx.scale(zoom,zoom);ctx.imageSmoothingEnabled=false;ctx.drawImage(art.background,0,0);
  if(route.length){ctx.strokeStyle='#fff4c480';ctx.lineWidth=3;ctx.setLineDash([3,9]);ctx.beginPath();ctx.moveTo(p.x,p.y);for(const tile of route){const r=projectVillage(tile);ctx.lineTo(r.x,r.y)}ctx.stroke();ctx.setLineDash([])}
  if(target){const t=projectVillage(target);ctx.strokeStyle='#ffe7a8';ctx.lineWidth=2;ctx.beginPath();ctx.arc(t.x,t.y,10,0,Math.PI*2);ctx.stroke()}
  for(const t of TABLES.tables){const c=projectVillage(t);ctx.strokeStyle='#f3d49670';ctx.lineWidth=1;ctx.setLineDash([4,6]);ctx.beginPath();ctx.arc(c.x,c.y,t.radius*40,0,Math.PI*2);ctx.stroke();ctx.setLineDash([])}
  const avatars=(online?publicState.avatars:[{...player,id:'local',name:localName,avatarId:localAvatar,status:'available' as const}]).filter(a=>!activityScene.replaces(a.id==='local'?'guest':a.id));
  const objects=art.foregroundObjects.filter(o=>visibleVillageProp(o,camera,zoom,{width,height})).map(o=>({depth:o.y,draw:()=>o.draw(ctx,motion?elapsed*1000:0)}));
  for(const avatar of avatars){const at=projectVillage(avatar),old=previous.get(avatar.id);const moving=old&&Math.hypot(old.x-avatar.x,old.y-avatar.y)>.001;const dx=old?avatar.x-old.x:0,dy=old?avatar.y-old.y:0;
   if(!visibleVillageProp(at,camera,zoom,{width,height}))continue;
   const own=avatar.id==='local'||avatar.id===selfID;const look=own?facing:moving?(Math.abs(dx)>Math.abs(dy)?dx>0?'right':'left':dy>0?'down':'up'):'down';
   objects.push({depth:at.y,draw:()=>drawCharacter(ctx,{...at,name:'',color:avatar.avatarId==='moon'?'#b3a0cf':avatar.avatarId==='autumn'?'#a4ba83':'#e5bd85',isPlayer:own,facing:look,walking:motion&&(own?walking:Boolean(moving)),seated:false},elapsed*1000)});previous.set(avatar.id,{x:avatar.x,y:avatar.y});
  }
  for(const actor of activityActors)if(visibleVillageProp(actor,camera,zoom,{width,height}))objects.push({depth:actor.y,draw:()=>actor.draw(ctx,performance.now())});
  objects.sort((a,b)=>a.depth-b.depth).forEach(o=>o.draw());
  ctx.restore();const labels:{x:number;y:number;w:number}[]=[];
  for(const a of avatars){const p=projectVillage(a);if(!visibleVillageProp(p,camera,zoom,{width,height}))continue;const x=camera.x+p.x*zoom;let y=camera.y+(p.y-53)*zoom;const text=`${a.name}${a.status==='available'?'':` · ${a.status}`}`;ctx.font='600 11px "DM Sans",sans-serif';const w=ctx.measureText(text).width+16;while(labels.some(l=>Math.abs(l.x-x)<(w+l.w)/2+4&&Math.abs(l.y-y)<24))y-=24;labels.push({x,y,w});label(text,x,y,a.id==='local'||a.id===selfID)}
  for(const actor of activityActors)if(actor.label&&visibleVillageProp(actor,camera,zoom,{width,height}))label(actor.label,camera.x+actor.x*zoom,camera.y+(actor.y-32)*zoom,actor.userId===(selfID??'guest'));
  if(minimapCanvas){const c=minimapCanvas.getContext('2d')!;c.clearRect(0,0,minimapCanvas.width,minimapCanvas.height);c.drawImage(mini,0,0,minimapCanvas.width,minimapCanvas.height);drawClosedZones(c,minimapCanvas.width,minimapCanvas.height);c.fillStyle='#ffe4aa';c.beginPath();c.arc(p.x/art.width*minimapCanvas.width,p.y/art.height*minimapCanvas.height,4,0,Math.PI*2);c.fill()}
  dirty=false;
 }
 function notifyPosition(){activities.setPosition(displayPlayer??player);
  const display=activityOwner&&displayPlayer?displayPlayer:player;
  const signature=`${display.x.toFixed(2)},${display.y.toFixed(2)},${walking}`;
  if(signature!==lastMove){lastMove=signature;onMove({...display,walking})}
  const table=TABLES.tables.some(t=>Math.hypot(player.x-t.x,player.y-t.y)<=t.radius+1);
  const closest=activityOwner?lastActivityKind:table?'warkop':positions.map(s=>({id:s.id,d:Math.hypot(player.x-s.x,player.y-s.y)})).filter(s=>s.d<3.5).sort((a,b)=>a.d-b.d)[0]?.id??null;
  if(closest!==near){near=closest;onNear(near)}
 }
 function stopDirection(){direction={x:0,y:0};online?.input(0,0);walking=false}
 function validDestination(point:Point|null|undefined):point is Point{return Boolean(point&&Number.isInteger(point.x)&&Number.isInteger(point.y)&&isPlayable(point.x,point.y)&&!occupied.has(tileKey(point.x,point.y)))}
 function resolvePoint(point:Point):Point|null{
  if(!validDestination(point))return null;
  const zone=closedZones.find(z=>point.x>=z.bounds.x&&point.x<z.bounds.x+z.bounds.width&&point.y>=z.bounds.y&&point.y<z.bounds.y+z.bounds.height);
  const resolved=resolveDestination?resolveDestination(zone?{...zone.entrance}:{...point}):zone?zone.entrance:point;
  if(!validDestination(resolved)||closedZones.some(z=>resolved.x>=z.bounds.x&&resolved.x<z.bounds.x+z.bounds.width&&resolved.y>=z.bounds.y&&resolved.y<z.bounds.y+z.bounds.height))return null;
  return{x:resolved.x,y:resolved.y};
 }
 function destination(point:Point){
  if(paused||activityOwner)return;
  const resolved=resolvePoint(point);if(!resolved)return;
  direction={x:0,y:0};target={...resolved};
  if(online){online.destination(resolved.x,resolved.y);return}
  const start={x:Math.round(player.x),y:Math.round(player.y)},path=findPath(start,resolved,occupied);
  route=path.length?[start,...path]:[];if(!route.length)target=null;dirty=true;
 }
 function drawClosedZones(c:CanvasRenderingContext2D,w:number,h:number){
  for(const zone of closedZones){const p=projectVillage(zone.bounds);c.fillStyle='#bf663e88';c.fillRect(p.x/art.width*w,p.y/art.height*h,zone.bounds.width*40/art.width*w,zone.bounds.height*40/art.height*h);c.fillStyle='#fff0cd';c.font='600 11px sans-serif';c.textAlign='left';c.fillText(zone.label??'In use',p.x/art.width*w,p.y/art.height*h+12)}
 }
 function travelTo(id:string){const place=positions.find(s=>s.id===id);if(place)destination(place)}
 function move(dx:number,dy:number){if(paused||activityOwner||!Number.isFinite(dx)||!Number.isFinite(dy))return;direction={x:Math.sign(dx),y:Math.sign(dy)};route=[];target=null;if(online)online.input(direction.x,direction.y);dirty=true}
 function interact(){
  if(paused)return;
  const position=displayPlayer??player;
  if(nearFootball(activities.snapshot().football,position)){
   activities.interactFootball(position);
   stopDirection();route=[];target=null;
   return;
  }
  if(activities.interactRide(position)){stopDirection();route=[];target=null;return}
  if(!near)return;
  stopDirection();route=[];target=null;
  if(stations.some(s=>s.id===near))onSelect(near);
 }
 const pointer=(e:PointerEvent)=>{const r=canvas.getBoundingClientRect();return{x:e.clientX-r.left,y:e.clientY-r.top}};
 function pointerDown(e:PointerEvent){if(e.button!==undefined&&e.button!==0)return;pointerStart={...pointer(e),id:e.pointerId};canvas.setPointerCapture?.(e.pointerId)}
 function pointerUp(e:PointerEvent){const p=pointer(e),start=pointerStart;pointerStart=null;if(!start||start.id!==e.pointerId||Math.hypot(p.x-start.x,p.y-start.y)>10)return;const tile=unprojectVillage({x:(p.x-camera.x)/zoom,y:(p.y-camera.y)/zoom});destination({x:Math.round(tile.x),y:Math.round(tile.y)});canvas.focus?.({preventScroll:true})}
 function cancelPointer(){pointerStart=null}
 function frame(timestamp:number){
  if(dead||document.hidden){raf=0;return}onlineSampler?.();const dt=Math.min((timestamp-(lastTime||timestamp))/1000,.05);lastTime=timestamp;
  if(!paused){if(motion)elapsed+=dt;if(!online&&!activityOwner){const before={...player};if(direction.x||direction.y)player=freeMovement(player,direction,dt);else if(route.length){player=advanceMovement(player,route[0],dt);if(Math.hypot(player.x-route[0].x,player.y-route[0].y)<.0001){route.shift();if(!route.length)target=null}}
    walking=Math.hypot(player.x-before.x,player.y-before.y)>.0001;const dx=player.x-before.x,dy=player.y-before.y;if(walking)facing=Math.abs(dx)>Math.abs(dy)?dx>0?'right':'left':dy>0?'down':'up';if(walking)dirty=true;notifyPosition();
   }}
  if(dirty||motion&&!paused||route.length)draw();raf=requestAnimationFrame(frame);
 }
 function visibility(){stopDirection();lastTime=0;if(document.hidden){cancelAnimationFrame(raf);raf=0}else if(!dead&&!raf){dirty=true;raf=requestAnimationFrame(frame)}}
 document.addEventListener?.('visibilitychange',visibility);
 const observer=new ResizeObserver(resize);observer.observe(canvas);
 canvas.addEventListener('pointerdown',pointerDown);canvas.addEventListener('pointerup',pointerUp);canvas.addEventListener('pointercancel',cancelPointer);
 resize();notifyPosition();if(!document.hidden)raf=requestAnimationFrame(frame);
 return {
  travelTo,travelToLocation:travelTo,move,interact,
  setOnlineSampler(sampler:(()=>void)|undefined){onlineSampler=sampler},
  setActivityState(state:ActivityState){const active=state.inputMode!=='walk';if(active&&!activityOwner){stopDirection();route=[];target=null}if(!active&&activityOwner&&!online){const area=activityContract.areas.find(a=>a.kind===lastActivityKind);if(area){player={...area.safeReturn};displayPlayer=null;activityOwner=active;notifyPosition()}}activityOwner=active;lastActivityKind=state.inputMode==='ride'?'race':state.active?.kind??'';activityScene.set(state);dirty=true},
  setClosedZones(zones:ClosedMapZone[]){closedZones=structuredClone(zones);dirty=true},
  setDestinationResolver(resolver:DestinationResolver|undefined){resolveDestination=resolver},
  mapDestination(clientPoint:Point,rect:CanvasRect){const p=mapPoint(clientPoint,rect,{x:0,y:0,width:art.width,height:art.height});if(!p)return null;const tile=unprojectVillage(p);return resolvePoint({x:Math.round(tile.x),y:Math.round(tile.y)})},
  walkToPoint:destination,
  drawMap(targetCanvas:HTMLCanvasElement){const c=targetCanvas.getContext('2d')!;c.clearRect(0,0,targetCanvas.width,targetCanvas.height);c.save();c.scale(targetCanvas.width/art.width,targetCanvas.height/art.height);c.drawImage(art.background,0,0);[...art.foregroundObjects].sort((a,b)=>a.y-b.y).forEach(o=>o.draw(c,0));c.restore();drawClosedZones(c,targetCanvas.width,targetCanvas.height);for(const d of positions){if(d.id==='arcade')continue;const p=projectVillage(d),x=p.x/art.width*targetCanvas.width,y=p.y/art.height*targetCanvas.height;c.font='600 12px sans-serif';const w=c.measureText(d.name).width+12;c.fillStyle='#213b32ed';c.fillRect(x-w/2,y-22,w,20);c.fillStyle='#fff0cd';c.textAlign='center';c.fillText(d.name,x,y-8)}},
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
  destroy(){dead=true;stopDirection();cancelAnimationFrame(raf);document.removeEventListener?.('visibilitychange',visibility);observer.disconnect();canvas.removeEventListener('pointerdown',pointerDown);canvas.removeEventListener('pointerup',pointerUp);canvas.removeEventListener('pointercancel',cancelPointer)},
 };
}
