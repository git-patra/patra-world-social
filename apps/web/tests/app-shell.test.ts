import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import vm from 'node:vm';
import {createWorld} from '../src/world/controller';
import {activities} from '../src/activities/store';
import {InputRouter} from '../src/activities/input-router';
import {PointerHold,acceptsMovement} from '../src/world/input';
import {createWorldArt} from '../src/world/village-art';
import {projectVillage,unprojectVillage,villageLocation} from '../src/world/village';
import {stations} from '../src/data.js';
import {mountMinimap} from '../src/world/minimap';
import {openCenteredDialog,closeDialogThen,mountDialogViewport,setDialogFocusReturn} from '../src/ui/dialogs';
import layout from '../../../contracts/kampung-senja-layout.v1.json';

// Exercise the production event handlers with real movement/input code. Feature
// mounts are isolated because these tests concern shell event ordering, not auth,
// React rendering, browser layout, or external services.
function shellFixture(pixelRatio=1){
 activities.clearLocal();
 const activitySubscriptions:(()=>void)[]=[];
 const original={window:globalThis.window,document:globalThis.document,Element:globalThis.Element,MutationObserver:globalThis.MutationObserver,HTMLElement:globalThis.HTMLElement,ResizeObserver:globalThis.ResizeObserver,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame,setTimeout:globalThis.setTimeout,clearTimeout:globalThis.clearTimeout,setInterval:globalThis.setInterval,clearInterval:globalThis.clearInterval};
 const gradient={addColorStop(){}};
 const context=new Proxy({measureText:()=>({width:40}),createRadialGradient:()=>gradient,createLinearGradient:()=>gradient},{get:(o,k)=>k in o?o[k as keyof typeof o]:()=>{},set:()=>true});
 let now=100,nextID=1,world!:ReturnType<typeof createWorld>;
 const timers=new Map<number,{at:number;repeat:number;run:()=>void}>(),frames=new Map<number,FrameRequestCallback>(),nodes=new Map<string,Element>();
 const windowEvents=new Map<string,Function[]>(),documentEvents=new Map<string,Function[]>(),observers=new Map<Element,Set<()=>void>>();
 const emit=(events:Map<string,Function[]>,kind:string,event:unknown)=>events.get(kind)?.forEach(fn=>fn(event));
 const add=(events:Map<string,Function[]>,kind:string,fn:Function)=>events.set(kind,[...events.get(kind)??[],fn]);
 class Element{
  opener:Element|null=null;onclick:any;isConnected=true;width=768;height=576;open=false;hidden=false;dataset:Record<string,string>={};style={};textContent='';innerHTML='';parent:Element|null=null;
  classList={add(){},toggle(){}};listeners=new Map<string,Function[]>();
  constructor(public id=''){}
  addEventListener(kind:string,fn:Function){add(this.listeners,kind,fn)}
  removeEventListener(kind:string,fn:Function){this.listeners.set(kind,(this.listeners.get(kind)??[]).filter(listener=>listener!==fn))}
  append(){}
  remove(){}
  setAttribute(){}
  getContext(){return context}
  getBoundingClientRect(){return{left:0,top:0,width:1000,height:700}}
  contains(node:Element|null):boolean{return node===this||Boolean(node?.parent&&this.contains(node.parent))}
  focus(){if(this.parent&&dialogs.includes(this.parent)&&!this.parent.open)return;if(dialogs.some(d=>d.open&&!d.contains(this)))return;document.activeElement=this}
  scrollIntoView(){}
  closest(selector:string):Element|null{
   if(selector==='[data-project]'&&this.dataset.project)return this;
   if(selector==='#places')return this.parent?.id==='#places'?this.parent:null;
   if(selector==='button'&&(this.dataset.move||this.id.startsWith('#')))return this;
   return null;
  }
  notifyOpen(){queueMicrotask(()=>observers.get(this)?.forEach(fn=>fn()))}
  showModal(){this.opener=document.activeElement;this.open=true;document.activeElement=this;this.notifyOpen()}
  close(){this.open=false;if(this.contains(document.activeElement)){document.activeElement=get('body');this.opener?.focus()}this.notifyOpen();queueMicrotask(()=>emit(this.listeners,'close',{target:this}))}
  click(detail=0,position={x:0,y:0}){const event={target:this,detail,clientX:position.x,clientY:position.y,preventDefault(){},stopPropagation(){}};this.focus();this.onclick?.(event);emit(this.listeners,'click',event);if(this.parent)emit(this.parent.listeners,'click',event);emit(documentEvents,'click',event)}
 }
 const get=(selector:string)=>{if(!nodes.has(selector))nodes.set(selector,new Element(selector));return nodes.get(selector)!};
 const dialogs=['map','menu','project','worlds','account','history','social','podium','settings'].map(id=>get(`#${id}-dialog`));dialogs.push(get('#chat-panel'));
 for(const id of ['settings-button','history-button','podium-button','locations-button'])get(`#${id}`).parent=get('#menu-dialog');
 const quickPlay=get('#quick-play-duck');quickPlay.parent=get('#menu-dialog');quickPlay.dataset.project='duck';
 const pad=get('right-pad');pad.dataset.move='1,0';
 const document={activeElement:get('#world'),hidden:false,body:get('body'),querySelector:(selector:string)=>selector==='dialog[open]'?dialogs.find(d=>d.open):get(selector),querySelectorAll:(selector:string)=>selector==='dialog'?dialogs:selector==='[data-move]'?[pad]:[],createElement:()=>new Element(),addEventListener:(kind:string,fn:Function)=>add(documentEvents,kind,fn),removeEventListener:(kind:string,fn:Function)=>documentEvents.set(kind,(documentEvents.get(kind)??[]).filter(f=>f!==fn))};
 const window={devicePixelRatio:pixelRatio,addEventListener:(kind:string,fn:Function)=>add(windowEvents,kind,fn),removeEventListener:(kind:string,fn:Function)=>windowEvents.set(kind,(windowEvents.get(kind)??[]).filter(f=>f!==fn)),dispatchEvent(){}};
 const schedule=(run:()=>void,delay=0,repeat=0)=>{const id=nextID++;timers.set(id,{run,at:now+delay,repeat});return id};
 const clock={setTimeout:(run:()=>void,delay=0)=>schedule(run,delay),clearTimeout:(id:number)=>timers.delete(id),setInterval:(run:()=>void,delay=0)=>schedule(run,delay,delay),clearInterval:(id:number)=>timers.delete(id)};
 const requestAnimationFrame=(fn:FrameRequestCallback)=>{const id=nextID++;frames.set(id,fn);return id};
 Object.assign(globalThis,{window,document,HTMLElement:Element,ResizeObserver:class{observe(){}disconnect(){}},requestAnimationFrame,cancelAnimationFrame:(id:number)=>frames.delete(id),...clock});
 const environment:Record<string,unknown>={window,document,HTMLElement:Element,Event,CustomEvent,requestAnimationFrame,queueMicrotask,...clock,matchMedia:()=>({matches:false,addEventListener(){}}),MutationObserver:class{constructor(private callback:()=>void){}disconnect(){}observe(element:Element){if(!observers.has(element))observers.set(element,new Set());observers.get(element)!.add(this.callback)}},IntersectionObserver:class{observe(){}},createWorld:(...args:Parameters<typeof createWorld>)=>(world=createWorld(...args)),PointerHold,acceptsMovement,villageLocation,stations,mountMinimap,openCenteredDialog,closeDialogThen,mountDialogViewport,setDialogFocusReturn,mountAudio:()=>()=>{},activePreferences:{subscribe(){},start(){},toggleMotion:async()=>{}}};
 Object.assign(globalThis,{Element,MutationObserver:environment.MutationObserver});
 environment.activities={snapshot:activities.snapshot,subscribe:(listener:()=>void)=>{const close=activities.subscribe(listener);activitySubscriptions.push(close);return close}};
 environment.activityInput=new InputRouter();
 environment.mountActivities=()=>()=>{};environment.mountActivityHUD=()=>()=>{};
 for(const name of ['mountSettings','mountGuestLifecycle','mountGuestUICleanup','mountVoice','mountVoicePanel','mountPenalty','mountAuth','mountChampionBoard','mountHistory','mountOnline','mountSocial','mountLobbies','mountLobbyPanel','mountChat','mountWorlds'])environment[name]=()=>{};
 environment.mountSettings=()=>{get('#settings-button').onclick=()=>get('#settings-dialog').showModal()};
 environment.mountHistory=()=>{const trigger=get('#history-button'),panel=get('#history-dialog');trigger.onclick=()=>panel.showModal();panel.addEventListener('close',()=>trigger.focus())};
 // The bundler runs tests from a temporary directory; cwd is the web workspace.
 const app=readFileSync(resolve('src/app.js'),'utf8').replace(/^import .*;\n/gm,'');
 vm.runInNewContext(app,environment,{filename:'src/app.js'});
 function tick(ms=50){
  now+=ms;
  for(const [id,timer]of [...timers])if(timer.at<=now){if(timer.repeat)timer.at=now+timer.repeat;else timers.delete(id);timer.run()}
  const pending=[...frames.values()];frames.clear();pending.forEach(fn=>fn(now));
 }
 return{world,get,pad,tick,
  routerKey:(type:string,key:string)=>emit(windowEvents,type,{type,key,target:get('body'),preventDefault(){},stopImmediatePropagation(){}}),
  padDown:(pointerId:number)=>emit(pad.listeners,'pointerdown',{pointerId,currentTarget:pad,preventDefault(){}}),
  pointerEnd:(pointerId:number)=>emit(windowEvents,'pointerup',{pointerId}),
  repeatInput:()=>(environment.activityInput as InputRouter).repeat(),visibility:(hidden:boolean)=>{document.hidden=hidden;emit(documentEvents,'visibilitychange',{})},frameCount:()=>frames.size,keydown:(key:string,repeat=false)=>emit(documentEvents,'keydown',{key,repeat,target:pad,preventDefault(){}}),keyup:(key:string)=>emit(windowEvents,'keyup',{key}),choose(id:string){const choice=new Element();choice.dataset.project=id;choice.parent=get('#places');choice.click(1)},restore(){for(const close of activitySubscriptions)close();activities.clearLocal();world.destroy();Object.assign(globalThis,original)}};
}

test('actual Map selection retains its online destination after dialog close cleanup',async()=>{
 const shell=shellFixture();try{
  const commands:unknown[][]=[];
  shell.world.setOnline({input:(dx,dy)=>commands.push(['input',dx,dy]),destination:(x,y)=>commands.push(['destination',x,y])});
  shell.get('#map-dialog').showModal();await Promise.resolve();commands.length=0;
  const before=shell.world.getState().player;
  shell.choose('race');await Promise.resolve();shell.tick();await Promise.resolve();
  assert.deepEqual(commands.filter(c=>c[0]==='destination'),[['destination',45,45]]);
  assert.deepEqual(commands.at(-1),['destination',45,45],'dialog close must not send a zero input after the destination and cancel the server route');
  assert.deepEqual(shell.world.getState().player,before,'online Map selection waits for authoritative positions');
 }finally{shell.restore()}
});

for(const key of ['Enter',' '])test(`semantic direction activation with ${JSON.stringify(key)} is bounded after release`,async()=>{
 const shell=shellFixture();try{
  shell.tick();const start=shell.world.getState().player;
  shell.pad.focus();shell.keydown(key);shell.pad.click();shell.keyup(key);
  for(let i=0;i<6;i++)shell.tick();
  const stopped=shell.world.getState().player;
  assert.ok(stopped.x>start.x,'semantic activation still moves the avatar');
  for(let i=0;i<10;i++)shell.tick();
  assert.deepEqual(shell.world.getState().player,stopped,'a keyboard or assistive click must not leave continuous movement running');
 }finally{shell.restore()}
});

test('approved layout solids match runtime art and top-down projection round-trips',()=>{
 const shell=shellFixture();try{
  assert.deepEqual(createWorldArt().solids,layout.solids);
  for(const point of [{x:0,y:0},{x:63,y:63},{x:24.25,y:32.75}])assert.deepEqual(unprojectVillage(projectVillage(point)),point);
 }finally{shell.restore()}
});


test('minimap_click_never_routes_ground_behind_it',async()=>{
 const shell=shellFixture();try{
  const commands:unknown[][]=[];shell.world.setOnline({input:(...input)=>commands.push(['input',...input]),destination:(...point)=>commands.push(['destination',...point])});
  const before=shell.world.getState().player;shell.get('#minimap-button').click();await Promise.resolve();
  assert.equal(shell.get('#map-dialog').open,true);assert.equal(commands.filter(c=>c[0]==='destination').length,0);assert.deepEqual(shell.world.getState().player,before);
  shell.get('#close-map').click();await Promise.resolve();assert.equal((globalThis.document as any).activeElement,shell.get('#minimap-button'));
 }finally{shell.restore()}
});

test('dialog_close_cleanup_precedes_destination',async()=>{
 const shell=shellFixture();try{
  const commands:unknown[][]=[];shell.world.setOnline({input:(...input)=>commands.push(['input',...input]),destination:(...point)=>commands.push(['destination',...point])});
  shell.get('#list-button').click();await Promise.resolve();commands.length=0;
  shell.choose('race');assert.equal(commands.filter(c=>c[0]==='destination').length,0);await Promise.resolve();shell.tick();
  assert.deepEqual(commands.at(-1),['destination',45,45]);assert.equal(shell.get('#map-dialog').open,false);
 }finally{shell.restore()}
});

test('map_click_respects_letterbox_and_dynamic_closed_zone',async()=>{
 const shell=shellFixture();try{
  const commands:unknown[][]=[];shell.world.setOnline({input:(...input)=>commands.push(['input',...input]),destination:(...point)=>commands.push(['destination',...point])});
  shell.world.setClosedZones([{id:'race',bounds:{x:40,y:40,width:12,height:12},entrance:{x:39,y:45}}]);
  const rect={left:0,top:0,width:1000,height:700};
  assert.equal(shell.world.mapDestination({x:1,y:350},rect),null,'horizontal letterbox cannot become a world destination');
  const p=projectVillage({x:45,y:45});const scale=700/1920;
  const destination=shell.world.mapDestination({x:(1000-2560*scale)/2+p.x*scale,y:p.y*scale},rect);
  assert.deepEqual(destination,{x:39,y:45});shell.world.walkToPoint(destination!);assert.deepEqual(commands.at(-1),['destination',39,45]);
  shell.world.setClosedZones([]);assert.deepEqual(shell.world.mapDestination({x:(1000-2560*scale)/2+p.x*scale,y:p.y*scale},rect),{x:45,y:45});
  const count=commands.length;for(const point of [{x:NaN,y:2},{x:Infinity,y:2},{x:-1,y:2},{x:64,y:2}])shell.world.walkToPoint(point);assert.equal(commands.length,count);
 }finally{shell.restore()}
});


test('closeDialogThen waits for close cleanup before scheduling the destination frame',async()=>{
 const shell=shellFixture();try{
  const events:string[]=[];const dialog=shell.get('#map-dialog');dialog.showModal();
  dialog.addEventListener('close',()=>events.push('cleanup'));
  closeDialogThen(dialog as any,()=>events.push('destination'));shell.tick();
  assert.deepEqual(events,[],'close task must run before destination frame can be queued');
  await Promise.resolve();shell.tick();assert.deepEqual(events,['cleanup','destination']);
 }finally{shell.restore()}
});


for(const dpr of [1,2])for(const zoom of [.65,1,1.5])test(`full map click uses CSS geometry at DPR ${dpr}, zoom ${zoom}`,async()=>{
 const shell=shellFixture(dpr);try{
  const commands:unknown[][]=[];shell.world.setOnline({input:(...input)=>commands.push(['input',...input]),destination:(...point)=>commands.push(['destination',...point])});shell.world.setZoom(zoom);
  shell.get('#list-button').click();await Promise.resolve();commands.length=0;
  const p=projectVillage({x:45,y:45}),scale=700/1920;
  shell.get('#village-map').click(1,{x:(1000-2560*scale)/2+p.x*scale,y:p.y*scale});
  await Promise.resolve();shell.tick();assert.deepEqual(commands.at(-1),['destination',45,45]);
  assert.deepEqual(shell.world.getState().player,{x:32,y:32});
 }finally{shell.restore()}
});


for(const [entry,panel] of [['#settings-button','#settings-dialog'],['#history-button','#history-dialog'],['#quick-play-duck','#project-dialog']])test(`menu child ${entry} returns focus to visible Menu without stealing from the child`,async()=>{
 const shell=shellFixture();try{
  shell.get('#menu-button').click();await Promise.resolve();shell.get(entry).click();await Promise.resolve();
  assert.equal(shell.get('#menu-dialog').open,false);assert.equal(shell.get(panel).open,true);
  assert.ok(shell.get(panel).contains((globalThis.document as any).activeElement),'parent close must leave focus in the newly open child');
  shell.get(panel).close();await Promise.resolve();
  assert.equal((globalThis.document as any).activeElement,shell.get('#menu-button'),'child close must restore a visible opener outside the closed menu');
 }finally{shell.restore()}
});

test('hidden world cancels its only frame and resumes without jumping',()=>{
 const shell=shellFixture();try{
  shell.tick();shell.world.move(1,0);shell.tick();const before=shell.world.getState().player;
  shell.visibility(true);assert.equal(shell.frameCount(),0);
  shell.tick(30000);assert.deepEqual(shell.world.getState().player,before);
  shell.visibility(false);assert.equal(shell.frameCount(),1);shell.tick();assert.deepEqual(shell.world.getState().player,before);
 }finally{shell.restore()}
});

test('neutral input heartbeat cannot cancel an online map route',async()=>{
 const shell=shellFixture();try{
  const commands:unknown[][]=[];shell.world.setOnline({input:(...input)=>commands.push(['input',...input]),destination:(...point)=>commands.push(['destination',...point])});
  shell.get('#list-button').click();await Promise.resolve();shell.choose('race');await Promise.resolve();shell.tick();
  const count=commands.length;shell.repeatInput();shell.repeatInput();assert.equal(commands.length,count);assert.deepEqual(commands.at(-1),['destination',45,45]);
 }finally{shell.restore()}
});

test('activity_location_display_tracks_own_actor_without_changing_walking_authority',async()=>{
 const {ActivityStore}=await import('../src/activities/store');const shell=shellFixture(),store=new ActivityStore();try{
  const before=shell.world.getState().player;await store.startLocal('race',1000,42,'display-projection');shell.world.setActivityState(store.snapshot());await import('../src/world/scenes/race');await Promise.resolve();shell.tick();
  assert.deepEqual(shell.world.getState().player,before);assert.notEqual(shell.get('#world').dataset.position,`${before.x.toFixed(2)},${before.y.toFixed(2)}`);assert.equal(shell.get('#current-location').textContent,layout.destinations.find(d=>d.id==='race')!.name);
  await store.action('leave');shell.world.setActivityState(store.snapshot());assert.deepEqual(shell.world.getState().player,{x:45,y:49});
 }finally{store.dispose();shell.restore()}
});

test('world_pad_release_preserves_held_keyboard_and_ignores_unrelated_pointerup',()=>{
 const shell=shellFixture();try{
  shell.tick();shell.routerKey('keydown','w');shell.padDown(7);shell.tick();const diagonal=shell.world.getState().player;
  shell.pointerEnd(99);shell.tick();const stillDiagonal=shell.world.getState().player;
  assert.ok(stillDiagonal.x>diagonal.x&&stillDiagonal.y<diagonal.y,'unrelated pointer does not end either owned input');
  shell.pointerEnd(7);shell.tick();const keyboardOnly=shell.world.getState().player;
  assert.equal(keyboardOnly.x,stillDiagonal.x);assert.ok(keyboardOnly.y<stillDiagonal.y,'ending the pad pointer preserves held W');
  shell.routerKey('keyup','w');shell.tick();assert.deepEqual(shell.world.getState().player,keyboardOnly);
 }finally{shell.restore()}
});

test('contextual_dock_keeps_action_text_separate_from_its_E_keycap', () => {
 const shell=shellFixture();
 try {
  // Cross the 100ms subscriber gate; fractional performance.now() can make an exact +100 subtract below 100.
  activities.setPosition({x:28,y:46});
  activities.tick(activities.snapshot().now+101);
  assert.equal(shell.get('#interact-button span').textContent,'Kick');
  activities.interactRide({x:39,y:46});
  assert.equal(shell.get('#interact-button span').textContent,'Turun');
  activities.interactRide();
  activities.setPosition({x:41,y:46});
  activities.tick(activities.snapshot().now+101);
  assert.equal(shell.get('#interact-button span').textContent,'Naik Sepeda');
 } finally {shell.restore()}
});

test('held_E_does_not_repeat_discrete_world_interaction', () => {
 const shell=shellFixture();
 try {
  let interactions=0;
  shell.world.interact=()=>{interactions++};
  shell.keydown('e');
  shell.keydown('e',true);
  assert.equal(interactions,1);
 } finally {shell.restore()}
});

test('football_cooldown_does_not_fall_through_to_opening_the_penalty_menu', () => {
 const shell=shellFixture();
 try {
  shell.world.walkToPoint({x:28,y:46});
  for(let i=0;i<200;i++)shell.tick();
  assert.deepEqual(shell.world.getState().player,{x:28,y:46});
  shell.world.interact();
  shell.world.interact();
  assert.equal(shell.get('#project-dialog').open,false,'cooldown is still the ball context');
 } finally {shell.restore()}
});

test('claw_map_destination_walks_to_cabinet_entrance_after_close',async()=>{
 assert.ok(stations.some(s=>s.id==='claw'),'map must list Capit Boneka');
 const shell=shellFixture();try{const commands:unknown[][]=[];shell.world.setOnline({input:(...a)=>commands.push(['input',...a]),destination:(...p)=>commands.push(['destination',...p])});shell.get('#list-button').click();await Promise.resolve();commands.length=0;shell.choose('claw');await Promise.resolve();shell.tick();assert.deepEqual(commands.at(-1),['destination',41,24]);assert.equal(shell.get('#map-dialog').open,false)}finally{shell.restore()}
});
