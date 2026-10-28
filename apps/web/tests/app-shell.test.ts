import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {resolve} from 'node:path';
import vm from 'node:vm';
import {createWorld} from '../src/world/controller';
import {PointerHold,acceptsMovement} from '../src/world/input';
import {createWorldArt} from '../src/world/village-art';
import {projectVillage,unprojectVillage,villageLocation} from '../src/world/village';
import {stations} from '../src/data.js';
import layout from '../../../contracts/kampung-senja-layout.v1.json';

// Exercise the production event handlers with real movement/input code. Feature
// mounts are isolated because these tests concern shell event ordering, not auth,
// React rendering, browser layout, or external services.
function shellFixture(){
 const original={window:globalThis.window,document:globalThis.document,HTMLElement:globalThis.HTMLElement,ResizeObserver:globalThis.ResizeObserver,requestAnimationFrame:globalThis.requestAnimationFrame,cancelAnimationFrame:globalThis.cancelAnimationFrame,setTimeout:globalThis.setTimeout,clearTimeout:globalThis.clearTimeout,setInterval:globalThis.setInterval,clearInterval:globalThis.clearInterval};
 const gradient={addColorStop(){}};
 const context=new Proxy({measureText:()=>({width:40}),createRadialGradient:()=>gradient,createLinearGradient:()=>gradient},{get:(o,k)=>k in o?o[k as keyof typeof o]:()=>{},set:()=>true});
 let now=100,nextID=1,world!:ReturnType<typeof createWorld>;
 const timers=new Map<number,{at:number;repeat:number;run:()=>void}>(),frames=new Map<number,FrameRequestCallback>(),nodes=new Map<string,Element>();
 const windowEvents=new Map<string,Function[]>(),documentEvents=new Map<string,Function[]>(),observers=new Map<Element,Set<()=>void>>();
 const emit=(events:Map<string,Function[]>,kind:string,event:unknown)=>events.get(kind)?.forEach(fn=>fn(event));
 const add=(events:Map<string,Function[]>,kind:string,fn:Function)=>events.set(kind,[...events.get(kind)??[],fn]);
 class Element{
  open=false;hidden=false;dataset:Record<string,string>={};style={};textContent='';innerHTML='';parent:Element|null=null;
  classList={add(){},toggle(){}};listeners=new Map<string,Function[]>();
  constructor(public id=''){}
  addEventListener(kind:string,fn:Function){add(this.listeners,kind,fn)}
  removeEventListener(){}
  setAttribute(){}
  getContext(){return context}
  getBoundingClientRect(){return{left:0,top:0,width:1000,height:700}}
  focus(){document.activeElement=this}
  scrollIntoView(){}
  closest(selector:string):Element|null{
   if(selector==='[data-project]'&&this.dataset.project)return this;
   if(selector==='#places')return this.parent?.id==='#places'?this.parent:null;
   if(selector==='button'&&this.dataset.move)return this;
   return null;
  }
  notifyOpen(){queueMicrotask(()=>observers.get(this)?.forEach(fn=>fn()))}
  showModal(){this.open=true;this.notifyOpen()}
  close(){this.open=false;this.notifyOpen()}
  click(detail=0){const event={target:this,detail,preventDefault(){}};emit(this.listeners,'click',event);emit(documentEvents,'click',event)}
 }
 const get=(selector:string)=>{if(!nodes.has(selector))nodes.set(selector,new Element(selector));return nodes.get(selector)!};
 const dialogs=['map','menu','project','worlds','account','history','social','podium','settings'].map(id=>get(`#${id}-dialog`));
 const pad=get('right-pad');pad.dataset.move='1,0';
 const document={activeElement:get('#world'),hidden:false,body:{dataset:{},classList:{toggle(){}}},querySelector:(selector:string)=>selector==='dialog[open]'?dialogs.find(d=>d.open):get(selector),querySelectorAll:(selector:string)=>selector==='dialog'?dialogs:selector==='[data-move]'?[pad]:[],createElement:()=>new Element(),addEventListener:(kind:string,fn:Function)=>add(documentEvents,kind,fn)};
 const window={devicePixelRatio:1,addEventListener:(kind:string,fn:Function)=>add(windowEvents,kind,fn),dispatchEvent(){}};
 const schedule=(run:()=>void,delay=0,repeat=0)=>{const id=nextID++;timers.set(id,{run,at:now+delay,repeat});return id};
 const clock={setTimeout:(run:()=>void,delay=0)=>schedule(run,delay),clearTimeout:(id:number)=>timers.delete(id),setInterval:(run:()=>void,delay=0)=>schedule(run,delay,delay),clearInterval:(id:number)=>timers.delete(id)};
 const requestAnimationFrame=(fn:FrameRequestCallback)=>{const id=nextID++;frames.set(id,fn);return id};
 Object.assign(globalThis,{window,document,HTMLElement:Element,ResizeObserver:class{observe(){}disconnect(){}},requestAnimationFrame,cancelAnimationFrame:(id:number)=>frames.delete(id),...clock});
 const environment:Record<string,unknown>={window,document,HTMLElement:Element,Event,CustomEvent,requestAnimationFrame,...clock,matchMedia:()=>({matches:false,addEventListener(){}}),MutationObserver:class{constructor(private callback:()=>void){}observe(element:Element){if(!observers.has(element))observers.set(element,new Set());observers.get(element)!.add(this.callback)}},IntersectionObserver:class{observe(){}},createWorld:(...args:Parameters<typeof createWorld>)=>(world=createWorld(...args)),PointerHold,acceptsMovement,villageLocation,stations,activePreferences:{subscribe(){},start(){},toggleMotion:async()=>{}}};
 for(const name of ['mountSettings','mountGuestLifecycle','mountGuestUICleanup','mountVoice','mountVoicePanel','mountPenalty','mountAuth','mountChampionBoard','mountHistory','mountOnline','mountSocial','mountLobbies','mountLobbyPanel','mountChat','mountWorlds'])environment[name]=()=>{};
 // The bundler runs tests from a temporary directory; cwd is the web workspace.
 const app=readFileSync(resolve('src/app.js'),'utf8').replace(/^import .*;\n/gm,'');
 vm.runInNewContext(app,environment,{filename:'src/app.js'});
 function tick(ms=50){
  now+=ms;
  for(const [id,timer]of [...timers])if(timer.at<=now){if(timer.repeat)timer.at=now+timer.repeat;else timers.delete(id);timer.run()}
  const pending=[...frames.values()];frames.clear();pending.forEach(fn=>fn(now));
 }
 return{world,get,pad,tick,keydown:(key:string)=>emit(documentEvents,'keydown',{key,target:pad,preventDefault(){}}),keyup:(key:string)=>emit(windowEvents,'keyup',{key}),choose(id:string){const choice=new Element();choice.dataset.project=id;choice.parent=get('#places');choice.click(1)},restore(){world.destroy();Object.assign(globalThis,original)}};
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
