import test from 'node:test';
import assert from 'node:assert/strict';
import data from '../../../contracts/map.v1.json';
import {loadMap, collisionTiles} from '../src/world/map';
import {advanceMovement} from '../src/world/input';
import {findPath,project,unproject,isPlayable} from '../src/world-math.js';
import {validateCommand,validateEvent} from '../src/world/avatars';
test('64 square map, reachable entrances and collision-free spawn',()=>{
 const map=loadMap(data); const blocked=collisionTiles(map);
 assert.equal(map.width,64); assert.ok(isPlayable(62,62)); assert.ok(!isPlayable(64,4));
 assert.ok(!blocked.has(`${map.spawn.x},${map.spawn.y}`));
 for(const s of map.buildings) assert.ok((s.entrance.x===map.spawn.x&&s.entrance.y===map.spawn.y)||findPath(map.spawn,s.entrance,blocked).length>0,s.id);
 for(const p of [{x:63,y:63},{x:24.25,y:32.75}]) assert.deepEqual(unproject(...Object.values(project(p.x,p.y)) as [number,number]),p);
});
test('map decoder fails closed',()=>{for(const edit of [{v:2},{width:17},{spawn:{x:NaN,y:0}},{buildings:undefined}])assert.throws(()=>loadMap({...data,...edit}));});
test('motion preference does not change movement duration or position',()=>{
 let a={x:0,y:0}, b={x:0,y:0}; const target={x:9,y:0};
 for(let tick=0;tick<40;tick++){a=advanceMovement(a,target,.05);b=advanceMovement(b,target,.05);assert.deepEqual(a,b)} assert.deepEqual(a,target);
});
test('protocol validates envelope and penalty zones',()=>{
 const c={v:1,commandId:'c1',seq:1,worldId:'local',type:'penalty.choose',payload:{zone:3,matchId:'m',turnId:'m:1'}};
 assert.equal(validateCommand(c).type,'penalty.choose');
 for(const edit of [{v:2},{seq:NaN},{worldId:undefined},{payload:{zone:4}},{type:'unknown'}])assert.throws(()=>validateCommand({...c,...edit}));
});

import {createWorld} from '../src/world.js';
test('approved open village replaces the old building island with reachable social spaces',()=>{
 const map=loadMap(data),blocked=collisionTiles(map);
 assert.equal(map.id,'kampung-senja-v1');
 for(const [name,x,y] of [['plaza',32,32],['warkop',16,24],['duck',45,24],['race',45,45],['penalty',28,46],['garden',14,44],['river',52,32]] as const){
  assert.ok(!blocked.has(`${x},${y}`),`${name} destination is open`);
  assert.ok(name==='plaza'||findPath(map.spawn,{x,y},blocked).length>0,`${name} reachable`);
 }
 assert.notEqual(map.checksum,'cb56899197324a5017021b345d60acc9a38da9b4ad78d14cace30130e29c99b2');
});
test('actual world controller preserves route when motion is toggled',()=>{
 let scheduled:FrameRequestCallback=()=>{};
 const old={window:globalThis.window,document:globalThis.document,ResizeObserver:globalThis.ResizeObserver,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame};
 const gradient={addColorStop(){}};
 const ctx=new Proxy({measureText:()=>({width:40}),createRadialGradient:()=>gradient,createLinearGradient:()=>gradient},{get:(o,k)=>k in o?o[k as keyof typeof o]:()=>{},set:()=>true});
 Object.assign(globalThis,{window:{devicePixelRatio:1},document:{createElement:()=>({getContext:()=>ctx,children:Array.from({length:4},()=>({addEventListener(){}})),setAttribute(){},remove(){},innerHTML:''})},ResizeObserver:class {observe(){} disconnect(){}},requestAnimationFrame:(callback:FrameRequestCallback)=>{scheduled=callback;return 1},cancelAnimationFrame:()=>{}});
 const listeners:Record<string,Function>={};
 const canvas={getContext:()=>ctx,getBoundingClientRect:()=>({width:1000,height:700,left:0,top:0}),parentElement:{append(){}},style:{},addEventListener(name:string,fn:Function){listeners[name]=fn},removeEventListener(){}};
 try{
  const run=(reduced:boolean)=>{const world=createWorld(canvas as any,{stations:[],onNear:()=>{},onSelect:()=>{},onMove:()=>{}});world.travelTo('race');scheduled(100);const positions=[];for(let i=1;i<=80;i++){if(i===10)world.setMotion(!reduced);if(i===20)world.setMotion(reduced);scheduled(100+i*50);positions.push(world.getState())}world.destroy();return positions};
  assert.deepEqual(run(false),run(true));
  const selected:string[]=[];const arrival=createWorld(canvas as any,{stations:data.buildings,onSelect:id=>selected.push(id)});arrival.travelTo('warkop');scheduled(100);for(let i=1;i<=400;i++)scheduled(100+i*50);assert.equal(arrival.getState().route.length,0);assert.deepEqual(selected,[],'arrival never opens an activity automatically');arrival.interact();assert.deepEqual(selected,['warkop'],'explicit interaction opens the real activity');arrival.destroy();
  const world=createWorld(canvas as any,{stations:[],onNear:()=>{},onSelect:()=>{},onMove:()=>{}});
  listeners.pointerdown({clientX:500,clientY:630.5,pointerId:1});listeners.pointerup({clientX:500,clientY:630.5,pointerId:1});
  assert.ok(world.getState().route.length>1,'canvas click starts a route');world.move(-1,0);
  assert.deepEqual(world.getState().route,[],'direction cancels click-to-walk');const origin=world.getState().player;scheduled(100);scheduled(150);assert.ok(world.getState().player.x<origin.x,'held left uses screen x axis');world.stopInput();const stopped=world.getState().player;scheduled(200);assert.deepEqual(world.getState().player,stopped,'release stops continuous input');
  const sent:any[]=[];world.setOnline({input:(dx:number,dy:number)=>sent.push([dx,dy]),destination:()=>{}});
  world.move(1,0);assert.deepEqual(sent.at(-1),[1,0],'top-down online directions remain world axes');assert.equal(sent.length,1);assert.equal(world.getState().route.length,0);
  world.setOnlineSnapshot(snapshot(1000,40),{...snapshot(1000,40),avatars:[...snapshot(1000,40).avatars,{id:'other',name:'Other',status:'available',x:42,y:32,seq:1}]});
  assert.equal(world.getState().player.x,40);world.setPaused(true);world.setOnlineSnapshot(snapshot(1100,41),snapshot(1100,41));assert.equal(world.getState().player.x,41,'modal cannot stop server state');
  world.reset();assert.equal(world.getState().player.x,41,'online reset cannot teleport');world.setOnline(null);assert.equal(world.getState().player.x,stopped.x,'solo state restored');world.destroy();
 }finally{Object.assign(globalThis,old)}
});

import commands from '../../../contracts/commands.golden.json';
import movement from '../../../contracts/movement.golden.json';
import {toSubunits} from '../src/world/input';
test('shared wire and fixed-point golden fixtures',()=>{for(const c of commands.valid)assert.doesNotThrow(()=>validateCommand(c));for(const c of commands.invalid)assert.throws(()=>validateCommand(c));for(const example of movement.examples)assert.equal(toSubunits(example.tiles),example.subunits)});

import {drawWorld} from '../src/world/renderer';
import {loadRules} from '../src/world/map';
import rulesData from '../../../contracts/rules.v1.json';
test('public avatars and world objects share one depth order',()=>{
 const painted:string[]=[];drawWorld({} as HTMLCanvasElement,loadMap(data),{avatars:[{id:'front',name:'Front',status:'available',x:33,y:33},{id:'back',name:'Back',status:'voice',x:31,y:31}]},{objects:[{depth:64,draw:()=>painted.push('building')}],paintAvatar:a=>painted.push(a.id)});assert.deepEqual(painted,['back','building','front']);
 assert.throws(()=>loadRules({...rulesData,subunitsPerTile:100}));assert.throws(()=>loadRules({...rulesData,hubSpeedSubunitsPerSecond:NaN}));
});

import {layoutNameplates} from '../src/world/renderer';
test('co-located avatar nameplates occupy separate lanes',()=>{
 const avatars=Array.from({length:3},(_,i)=>({id:`a${i}`,name:`Avatar ${i}`,status:'available' as const,x:32,y:32}));
 const labels=layoutNameplates(avatars,()=>100);assert.equal(labels.length,3);
 for(let i=0;i<labels.length;i++)for(let j=i+1;j<labels.length;j++)assert.ok(Math.abs(labels[i].y-labels[j].y)>=16);
});

import invalidMaps from '../../../contracts/maps.invalid.golden.json';
import events from '../../../contracts/events.golden.json';
test('shared semantic invalid maps fail at load boundary',()=>{for(const map of Object.values(invalidMaps))assert.throws(()=>loadMap(map));});
test('shared events validate required envelope fields and type set',()=>{for(const event of events.valid)assert.doesNotThrow(()=>validateEvent(event));for(const event of events.invalid)assert.throws(()=>validateEvent(event));assert.throws(()=>validateEvent({...events.valid[0],serverTick:NaN}));});

import {clampZoom} from '../src/world/camera';
import {SnapshotBuffer,validateSnapshot} from '../src/net/reconcile';
import ruleAsset from '../../../contracts/rules.v1.json';
test('nonfinite zoom cannot corrupt camera',()=>{assert.equal(clampZoom(NaN),.85);assert.equal(clampZoom(Infinity),.85)});
const snapshot=(timeMs:number,x:number)=>({worldId:'a',selfId:'u',sessionId:'opaque',epoch:1,mapId:data.id,mapChecksum:data.checksum,rulesId:ruleAsset.id,rulesChecksum:ruleAsset.checksum,protocolVersion:1,tick:timeMs/50,timeMs,avatars:[{id:'u',name:'User',status:'available' as const,x,y:32,seq:1}]});
test('online interpolation is bounded and full resync resets old epochs',()=>{
 const b=new SnapshotBuffer();b.push(validateSnapshot(snapshot(1000,32),'a'));b.push(validateSnapshot(snapshot(1100,32.45),'a'));
 assert.equal(b.at(1050).avatars[0].x,32.225);assert.equal(b.at(9999).avatars[0].x,32.45);
 b.push(validateSnapshot({...snapshot(1200,40),epoch:2},'a'));assert.equal(b.at(1050).avatars[0].x,40);
 assert.throws(()=>validateSnapshot({...snapshot(1000,32),mapChecksum:'wrong'},'a'));assert.throws(()=>validateSnapshot(snapshot(1000,32),'other'));
});
import {WorldSocket} from '../src/net/socket';
test('socket only sends when online and reconnect uses rotated seat without auto takeover',()=>{
 const all:any[]=[];
 class FakeSocket {readyState=0;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;onopen:any;constructor(public url:string){all.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const old=globalThis.WebSocket;globalThis.WebSocket=FakeSocket as any;
 try{const states:string[]=[];const socket=new WorldSocket('a',()=>{},s=>states.push(s),'http://village.test');socket.connect();socket.input(1,0);assert.equal(all[0].sent.length,0);all[0].readyState=1;all[0].onmessage({data:JSON.stringify({v:1,eventId:'e',serverTick:20,serverTimeMs:1000,type:'world.snapshot',payload:snapshot(1000,32)})});socket.input(1,0);assert.equal(all[0].sent.length,1);assert.equal(JSON.parse(all[0].sent[0]).worldId,'a');socket.connect();assert.match(all[1].url,/sessionId=opaque/);assert.doesNotMatch(all[1].url,/takeover=true/);socket.close();assert.equal(states.at(-1),'leave_unconfirmed')}finally{globalThis.WebSocket=old}
});
import snapshots from '../../../contracts/snapshots.golden.json';
test('shared snapshot fixtures reject forged versions, unknown fields, missing self and duplicate avatars',()=>{for(const s of snapshots.valid)assert.doesNotThrow(()=>validateSnapshot(s,'a'));for(const s of snapshots.invalid)assert.throws(()=>validateSnapshot(s,'a'))});
import {PointerHold} from '../src/world/input';
test('canvas pointer release preserves destination; only a directional hold sends stop',()=>{
 const commands:string[]=[];const hold=new PointerHold(()=>commands.push('direction'),()=>commands.push('stop'));
 commands.push('destination');hold.release();assert.deepEqual(commands,['destination']);
 hold.start(1,0);hold.release();assert.deepEqual(commands,['destination','direction','stop']);hold.release();assert.equal(commands.length,3);
});
test('explicit socket leave waits for matching acknowledgement before replacement join',async()=>{
 const all:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;closes=0;constructor(public url:string){all.push(this)}send(s:string){this.sent.push(s)}close(){this.closes++;this.readyState=3}}
 const old=globalThis.WebSocket;globalThis.WebSocket=FakeSocket as any;
 try{const first=new WorldSocket('a',()=>{},()=>{},'http://village.test');first.connect();all[0].onmessage({data:JSON.stringify({v:1,eventId:'snapshot',serverTick:20,serverTimeMs:1000,type:'world.snapshot',payload:snapshot(1000,32)})});
 let finished=false;const leaving=first.close();Promise.resolve(leaving).then(()=>finished=true);await Promise.resolve();assert.equal(finished,false,'leave must remain pending until server ack');assert.equal(all[0].closes,0);
 const cmd=JSON.parse(all[0].sent.at(-1));assert.equal(cmd.type,'world.leave');
 all[0].onmessage({data:JSON.stringify({v:1,eventId:'ack',serverTick:20,serverTimeMs:1000,type:'command.ack',payload:{commandId:cmd.commandId,seq:cmd.seq}})});
 assert.equal(await leaving,true);assert.equal(all[0].closes,1);const next=new WorldSocket('b',()=>{},()=>{},'http://village.test');next.connect();assert.match(all[1].url,/worldId=b/);await next.close();
 }finally{globalThis.WebSocket=old}
});
import {mountOnline} from '../src/online-ui';
test('world picker serializes the real controller switch behind leave acknowledgement',async()=>{
 const previous={window:globalThis.window,document:globalThis.document,WebSocket:globalThis.WebSocket,location:globalThis.location,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame};
 const connections:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;constructor(public url:string){connections.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const buttons=Array.from({length:3},()=>({hidden:true,onclick:null}));const label={textContent:''};const panel={setAttribute(){},querySelector:()=>label,querySelectorAll:()=>buttons,innerHTML:'',className:''};
 const events=new EventTarget();Object.assign(globalThis,{window:events,document:{createElement:()=>panel,querySelector:(s:string)=>s==='#world'?{parentElement:{append(){}}}:label},location:{origin:'http://village.test'},WebSocket:FakeSocket,requestAnimationFrame:()=>1,cancelAnimationFrame(){}});
 const settle=async()=>{for(let i=0;i<6;i++)await Promise.resolve()};
 try{mountOnline({setConnectionStatus(){},setOnline(){},setOnlineSnapshot(){}} as any);events.dispatchEvent(new CustomEvent('patra-auth-change',{detail:{id:'u'}}));events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'a',name:'A'}}));await settle();assert.equal(connections.length,1);
 const event=(type:string,payload:any)=>({data:JSON.stringify({v:1,eventId:'e',serverTick:20,serverTimeMs:1000,type,payload})});connections[0].onmessage(event('world.snapshot',snapshot(1000,32)));
 events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'b',name:'B'}}));await settle();assert.equal(connections.length,1,'controller must not connect replacement before release');const leave=JSON.parse(connections[0].sent.at(-1));
 connections[0].onmessage(event('command.ack',{commandId:leave.commandId,seq:leave.seq}));await settle();assert.equal(connections.length,2);assert.match(connections[1].url,/worldId=b/);events.dispatchEvent(new Event('pagehide'));await settle();
 }finally{Object.assign(globalThis,previous)}
});
for(const reason of ['leave','logout'] as const)test(`disconnected ${reason} restores solo while pending release fences replacement`,async()=>{
 const previous={window:globalThis.window,document:globalThis.document,WebSocket:globalThis.WebSocket,location:globalThis.location,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame};
 const connections:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;constructor(public url:string){connections.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const buttons=Array.from({length:3},()=>({hidden:true,onclick:null}));const label={textContent:''};const panel={setAttribute(){},querySelector:()=>label,querySelectorAll:()=>buttons,innerHTML:'',className:''};
 const events=new EventTarget();Object.assign(globalThis,{window:events,document:{createElement:()=>panel,querySelector:(s:string)=>s==='#world'?{parentElement:{append(){}}}:label},location:{origin:'http://village.test'},WebSocket:FakeSocket,requestAnimationFrame:()=>1,cancelAnimationFrame(){}});
 const settle=async()=>{for(let i=0;i<8;i++)await Promise.resolve()};let adapter:any=null,soloMoves=0,soloRoute=false;
 const world={setConnectionStatus(){},setOnline(value:any){adapter=value;soloRoute=false},setOnlineSnapshot(){},move(){if(!adapter){soloMoves++;soloRoute=true}}};
 try{mountOnline(world as any);events.dispatchEvent(new CustomEvent('patra-auth-change',{detail:{id:'u'}}));events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'a',name:'A'}}));await settle();
 connections[0].onmessage({data:JSON.stringify({v:1,eventId:'s',serverTick:20,serverTimeMs:1000,type:'world.snapshot',payload:snapshot(1000,32)})});assert.ok(adapter);
 connections[0].readyState=3;connections[0].onclose({code:1006});
 if(reason==='leave')buttons[1].onclick();else events.dispatchEvent(new CustomEvent('patra-auth-change',{detail:null}));await settle();
 assert.equal(adapter,null,'local solo must detach without a release ack');world.move();assert.equal(soloMoves,1,'local movement restored');assert.match(label.textContent,/Solo lokal.*belum dikonfirmasi/);
 events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'b',name:'B'}}));await settle();assert.equal(connections.length,1,'pending release or logout must fence replacement joins');
 if(reason==='logout'){buttons[2].onclick();await settle();assert.equal(connections.length,1,'guest retry cannot revive an authenticated socket')}
 else {
  buttons[2].onclick();await settle();assert.equal(connections.length,2);assert.match(connections[1].url,/sessionId=opaque/);assert.equal(adapter,null,'recovery must remain local solo');
  const event=(type:string,payload:any)=>({data:JSON.stringify({v:1,eventId:'recovery',serverTick:22,serverTimeMs:1100,type,payload})});
  connections[1].onmessage(event('world.snapshot',{...snapshot(1100,33),epoch:2,sessionId:'rotated'}));await settle();assert.equal(adapter,null);const release=JSON.parse(connections[1].sent.at(-1));assert.equal(release.type,'world.leave');
  connections[1].onmessage(event('command.ack',{commandId:release.commandId,seq:release.seq}));await settle();assert.equal(label.textContent,'Solo lokal · tidak tersambung');assert.equal(soloRoute,true,'cleanup recovery must preserve the active solo route');
  events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'b',name:'B'}}));await settle();assert.equal(connections.length,3,'confirmed recovery permits later authenticated join');assert.match(connections[2].url,/worldId=b/);
  events.dispatchEvent(new Event('pagehide'));await settle();
 }

 }finally{Object.assign(globalThis,previous)}
});
test('logout fences a queued world switch even when its old leave is later acknowledged',async()=>{
 const previous={window:globalThis.window,document:globalThis.document,WebSocket:globalThis.WebSocket,location:globalThis.location,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame};
 const connections:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;constructor(public url:string){connections.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const buttons=Array.from({length:3},()=>({hidden:true,onclick:null}));const label={textContent:''};const panel={setAttribute(){},querySelector:()=>label,querySelectorAll:()=>buttons,innerHTML:'',className:''};
 const events=new EventTarget();Object.assign(globalThis,{window:events,document:{createElement:()=>panel,querySelector:(s:string)=>s==='#world'?{parentElement:{append(){}}}:label},location:{origin:'http://village.test'},WebSocket:FakeSocket,requestAnimationFrame:()=>1,cancelAnimationFrame(){}});
 const settle=async()=>{for(let i=0;i<8;i++)await Promise.resolve()};let adapter:any=null;
 try{mountOnline({setConnectionStatus(){},setOnline(value:any){adapter=value},setOnlineSnapshot(){}} as any);events.dispatchEvent(new CustomEvent('patra-auth-change',{detail:{id:'u'}}));events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'a',name:'A'}}));await settle();
 const event=(type:string,payload:any)=>({data:JSON.stringify({v:1,eventId:'e',serverTick:20,serverTimeMs:1000,type,payload})});connections[0].onmessage(event('world.snapshot',snapshot(1000,32)));
 events.dispatchEvent(new CustomEvent('patra-world-join',{detail:{id:'b',name:'B'}}));await settle();const leave=JSON.parse(connections[0].sent.at(-1));events.dispatchEvent(new CustomEvent('patra-auth-change',{detail:null}));assert.equal(adapter,null,'logout detaches immediately, before pending ack');
 connections[0].onmessage(event('command.ack',{commandId:leave.commandId,seq:leave.seq}));await settle();assert.equal(connections.length,1,'old queued join cannot run after logout');assert.equal(adapter,null);assert.equal(label.textContent,'Solo lokal · tidak tersambung');
 buttons[2].onclick();await settle();assert.equal(connections.length,1);
 }finally{Object.assign(globalThis,previous)}
});

import {mergeMessages,validMessage} from '../src/chat-model';
import type {ChatMessage} from '../src/net/api';
const message=(id:string,edit:Partial<ChatMessage>={}):ChatMessage=>({id,worldId:'a',userId:'u',username:'friend',clientMessageId:'retry',text:'<img src=x onerror=alert(1)>',createdAt:'2026-10-27T00:00:00Z',deleted:false,...edit});
test('chat resync merges exact durable IDs without duplicates or deletion resurrection',()=>{
 const old=message('9007199254740993');const deleted=message('1',{text:'',deleted:true});
 const merged=mergeMessages([old,deleted],[old,message('1'),message('9007199254740992')]);
 assert.deepEqual(merged.map(m=>m.id),['1','9007199254740992','9007199254740993']);assert.equal(merged[0].deleted,true);assert.equal(merged[0].text,'');assert.ok(validMessage(old,'a'));assert.equal(validMessage(old,'b'),false);
 assert.equal(validMessage(message('2',{text:'🎃'.repeat(501)}),'a'),false);
});
test('chat send succeeds only after matching stored-message ack and failure preserves explicit error',async()=>{
 const all:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;constructor(public url:string){all.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const old=globalThis.WebSocket;globalThis.WebSocket=FakeSocket as any;
 try{const socket=new WorldSocket('a',()=>{},()=>{},'http://village.test');const events:string[]=[];socket.onEvent=type=>events.push(type);socket.connect();
 const event=(type:string,payload:any)=>({data:JSON.stringify({v:1,eventId:'chat',serverTick:20,serverTimeMs:1000,type,payload})});
 all[0].onmessage(event('world.snapshot',snapshot(1000,32)));const pending=socket.sendChat('durable-1','hello');let resolved=false;pending.then(()=>resolved=true);await Promise.resolve();assert.equal(resolved,false);
 const first=JSON.parse(all[0].sent.at(-1));assert.deepEqual(first.payload,{channelId:'world',clientMessageId:'durable-1',text:'hello'});
 all[0].onmessage(event('command.ack',{commandId:'unrelated',result:message('1')}));await Promise.resolve();assert.equal(resolved,false);
 all[0].onmessage(event('command.ack',{commandId:first.commandId,result:message('1')}));assert.equal((await pending as ChatMessage).id,'1');
 const rejected=socket.sendChat('durable-2','too fast');const second=JSON.parse(all[0].sent.at(-1));all[0].onmessage(event('command.error',{commandId:second.commandId,code:'rate',message:'Tunggu sebelum mengirim.'}));await assert.rejects(rejected,/Tunggu/);
 all[0].onmessage(event('chat.message',message('2')));all[0].onmessage(event('chat.deleted',{worldId:'a',id:'2'}));assert.deepEqual(events,['world.snapshot','chat.message','chat.deleted']);
 const leaving=socket.close();const leave=JSON.parse(all[0].sent.at(-1));all[0].onmessage(event('command.ack',{commandId:leave.commandId}));assert.equal(await leaving,true);
 await assert.rejects(socket.sendChat('late','x'),/koneksi dunia aktif/);
 }finally{globalThis.WebSocket=old}
});

test('a delete received before its live message prevents later text resurrection',()=>{
 const tombstones=new Set(['7']);
 const merged=mergeMessages([], [message('7')], tombstones);
 assert.equal(merged[0].deleted,true);assert.equal(merged[0].text,'');
});

import {mountChat} from '../src/chat-ui';
import {api} from '../src/net/api';
class ChatElement {
 value='';textContent='';hidden=false;disabled=false;children:ChatElement[]=[];
 onclick:any;onsubmit:any;controls:Record<string,ChatElement>={};
 querySelector(selector:string){return this.controls[selector]??null}
 replaceChildren(){this.children=[]}
 append(...children:ChatElement[]){this.children.push(...children)}
 focus(){}
}
function mountedChatForm(){
 const previous={window:globalThis.window,document:globalThis.document,members:api.members};
 const nodes=Object.fromEntries(['chat-panel','chat-button','chat-badge','chat-status','chat-messages','chat-form','chat-older','chat-latest','close-chat'].map(id=>[`#${id}`,new ChatElement()]));
 const form=nodes['#chat-form'],input=new ChatElement(),send=new ChatElement();form.controls={textarea:input,button:send};
 const events=new EventTarget();
 Object.assign(globalThis,{window:events,document:{querySelector:(selector:string)=>nodes[selector],createElement:()=>new ChatElement()}});api.members=async()=>[];
 mountChat();
 const attach=(socket:any)=>events.dispatchEvent(new CustomEvent('patra-world-attachment',{detail:socket}));
 const connection=(worldId:string)=>{
  const calls:{id:string;text:string;resolve:(value:ChatMessage)=>void;reject:(error:Error)=>void}[]=[];
  const socket={worldId,status:'online',sendChat:(id:string,text:string)=>new Promise<ChatMessage>((resolve,reject)=>calls.push({id,text,resolve,reject}))};
  return {socket,calls};
 };
 return {nodes,form,input,send,attach,connection,submit:()=>form.onsubmit({preventDefault(){}}) as Promise<void>,restore:()=>{Object.assign(globalThis,{window:previous.window,document:previous.document});api.members=previous.members}};
}
test('actual chat form preserves an edited next draft while the submitted message awaits its ACK',async()=>{
 const chat=mountedChatForm();try{
  const {socket,calls}=chat.connection('a');chat.attach(socket);chat.input.value='message A';
  const first=chat.submit();assert.equal(chat.send.disabled,true);chat.input.value='draft B typed while A is pending';
  calls[0].resolve(message('20',{clientMessageId:calls[0].id,text:'message A'}));await first;
  assert.equal(chat.input.value,'draft B typed while A is pending','ACK for A must preserve the newly edited draft B');assert.equal(chat.send.disabled,false);
  const second=chat.submit();assert.notEqual(calls[1].id,calls[0].id,'next draft needs its own durable ID');assert.equal(calls[1].text,'draft B typed while A is pending');
  calls[1].resolve(message('21',{clientMessageId:calls[1].id,text:calls[1].text}));await second;assert.equal(chat.input.value,'','unchanged submitted draft clears after success');
 }finally{chat.restore()}
});
test('actual chat form retains failed draft and durable client ID for an explicit retry',async()=>{
 const chat=mountedChatForm();try{
  const {socket,calls}=chat.connection('a');chat.attach(socket);chat.input.value='retry this message';
  const failed=chat.submit();calls[0].reject(new Error('Belum dikonfirmasi.'));await failed;
  assert.equal(chat.input.value,'retry this message');assert.match(chat.nodes['#chat-status'].textContent,/Belum dikonfirmasi/);assert.equal(chat.send.disabled,false);
  const retry=chat.submit();assert.equal(calls[1].id,calls[0].id,'retry must reuse durable identity');
  calls[1].resolve(message('22',{clientMessageId:calls[1].id,text:calls[1].text}));await retry;assert.equal(chat.input.value,'');
 }finally{chat.restore()}
});
test('actual chat form ignores old ACKs after switching attachments while a new draft is pending',async()=>{
 const chat=mountedChatForm();try{
  const a=chat.connection('a'),b=chat.connection('b');chat.attach(a.socket);chat.input.value='old world';const old=chat.submit();
  chat.attach(b.socket);chat.input.value='new world draft';const current=chat.submit();
  a.calls[0].resolve(message('23',{clientMessageId:a.calls[0].id,text:'old world'}));await old;
  assert.equal(chat.input.value,'new world draft');assert.equal(chat.send.disabled,true,'old ACK must not enable the pending new-world form');assert.equal(chat.nodes['#chat-messages'].children.length,0,'old-world message must not render');
  b.calls[0].resolve(message('24',{worldId:'b',clientMessageId:b.calls[0].id,text:'new world draft'}));await current;assert.equal(chat.input.value,'');assert.equal(chat.send.disabled,false);assert.equal(chat.nodes['#chat-messages'].children.length,1);
 }finally{chat.restore()}
});

import {renderToStaticMarkup} from 'react-dom/server';
import React from 'react';
import {LobbyView} from '../src/features/lobby/LobbyPanel';
test('production lobby view truthfully disables Start without a concrete engine',()=>{
 const lobby={id:'l',arenaId:'race',worldId:'a',hostId:'u',state:'lobby',gameType:'race',capacity:5,engineAvailable:false,botChosen:false,players:[{userId:'u',actorId:'actor',actorType:'human',displayName:'<Pumpkin>',ready:true,connected:true}],spectators:[]};
 const html=renderToStaticMarkup(React.createElement(LobbyView,{lobbies:[lobby],arenaId:'race',selfId:'u',online:true,pending:false,message:'',act:()=>{}}));
 assert.match(html,/Mesin permainan belum tersedia/);assert.match(html,/<button[^>]*disabled=""[^>]*>Mulai/);assert.match(html,/&lt;Pumpkin&gt;/);
 const offline=renderToStaticMarkup(React.createElement(LobbyView,{lobbies:[],arenaId:'race',selfId:'',online:false,pending:false,message:'',act:()=>{}}));assert.match(offline,/Sambungkan ke dunia privat/);
});
test('lobby wire commands reject identity spoofing and socket dispatches resync events',async()=>{
 assert.equal(validateCommand({v:1,commandId:'list',seq:1,worldId:'a',type:'lobby.list',payload:{}}).type,'lobby.list');
 assert.throws(()=>validateCommand({v:1,commandId:'spoof',seq:1,worldId:'a',type:'lobby.join',payload:{arenaId:'race',userId:'other'}}));
 const all:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;sent:string[]=[];onmessage:any;onclose:any;constructor(public url:string){all.push(this)}send(s:string){this.sent.push(s)}close(){this.readyState=3}}
 const old=globalThis.WebSocket;globalThis.WebSocket=FakeSocket as any;
 try{const socket=new WorldSocket('a',()=>{},()=>{},'http://village.test'),events:string[]=[];socket.onEvent=t=>events.push(t);socket.connect();const event=(type:string,payload:any)=>({data:JSON.stringify({v:1,eventId:'e',serverTick:0,serverTimeMs:1000,type,payload})});all[0].onmessage(event('world.snapshot',snapshot(1000,32)));
 const pending=socket.gameCommand('lobby.list',{});const command=JSON.parse(all[0].sent.at(-1));all[0].onmessage(event('command.ack',{commandId:command.commandId,result:{worldId:'a',lobbies:[]}}));assert.deepEqual(await pending,{worldId:'a',lobbies:[]});
 all[0].onmessage(event('lobby.updated',{worldId:'a',lobbies:[]}));assert.deepEqual(events,['world.snapshot','lobby.updated']);
 const closed=socket.close();const leave=JSON.parse(all[0].sent.at(-1));all[0].onmessage(event('command.ack',{commandId:leave.commandId}));await closed;
 }finally{globalThis.WebSocket=old}
});

import {MatchDetail,type MatchRecord} from '../src/features/history/MatchDetail';
test('history does not invent a score or duration for an unknown-start aborted match',()=>{
 const match:MatchRecord={matchId:'m',worldId:'w',worldName:'<Autumn>',gameType:'duck',status:'aborted_server',preparedAt:'2026-01-01T00:00:00Z',startsAt:null,finishedAt:null,version:{mapId:'autumn',mapVersion:1,rulesId:'beta',rulesVersion:1,protocolVersion:1},participants:[{actorId:'a',actorType:'human',displayName:'<Visitor>',avatarId:'pumpkin'}]};
 const html=renderToStaticMarkup(React.createElement(MatchDetail,{match}));assert.match(html,/Waktu mulai tidak diketahui/);assert.doesNotMatch(html,/Durasi:|Skor|Tersimpan/);assert.match(html,/&lt;Visitor&gt;/);
 const saved={...match,status:'completed',startsAt:'2026-01-01T00:00:00Z',result:{startsAt:1000,finishedAt:61000,participants:[{...match.participants[0],botAssisted:true,outcome:'finished',score:25,duck:{hits:2,shots:4}}]}};
 const result=renderToStaticMarkup(React.createElement(MatchDetail,{match:saved}));assert.match(result,/Tersimpan/);assert.match(result,/60/);assert.match(result,/50.0%/);assert.match(result,/Bot assistance/);
});

test('actual socket dispatch forwards race and duck snapshots while fencing replaced sockets and worlds',()=>{
 const all:any[]=[];class FakeSocket{readyState=1;bufferedAmount=0;onmessage:any;onclose:any;constructor(public url:string){all.push(this)}send(){}close(){this.readyState=3}}
 const old=globalThis.WebSocket;globalThis.WebSocket=FakeSocket as any;
 try{const socket=new WorldSocket('a',()=>{},()=>{},'http://village.test'),events:string[]=[];socket.onEvent=t=>events.push(t);socket.connect();const event=(type:string,worldId='a')=>({data:JSON.stringify({v:1,eventId:'e',serverTick:0,serverTimeMs:1000,type,payload:{worldId,matchId:'m'}})});
 for(const kind of ['race.snapshot','race.finished','duck.snapshot','duck.finished'])all[0].onmessage(event(kind));assert.deepEqual(events,['race.snapshot','race.finished','duck.snapshot','duck.finished']);all[0].onmessage(event('duck.snapshot','other'));assert.equal(events.length,4);socket.connect();all[0].onmessage(event('duck.snapshot'));assert.equal(events.length,4);all[1].onmessage(event('duck.snapshot'));assert.equal(events.length,5);
 }finally{globalThis.WebSocket=old}
});

import {visibleVillageProp} from '../src/world/controller';
test('large Warkop canopy stays visible at mobile viewport edges at every zoom',()=>{
 for(const zoom of [.65,1,1.5]){
  const camera={x:195-1060*zoom,y:422-460*zoom};
  assert.ok(visibleVillageProp({x:624,y:416},camera,zoom,{width:390,height:844}));
  assert.equal(visibleVillageProp({x:2400,y:416},camera,zoom,{width:390,height:844}),false);
 }
});
