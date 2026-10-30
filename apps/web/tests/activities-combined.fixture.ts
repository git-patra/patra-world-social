import {createWorld} from '../src/world/controller';
import {ActivityStore} from '../src/activities/store';
import {AudioService} from '../src/audio/service';
import {combinedFixture} from './activities-combined-data';
const canvas=document.querySelector('canvas')!,output=document.querySelector('output')!;
const nativeRAF=window.requestAnimationFrame.bind(window),nativeCancel=window.cancelAnimationFrame.bind(window);
const nativeInterval=window.setInterval.bind(window),nativeClear=window.clearInterval.bind(window);
const nativeTimeout=window.setTimeout.bind(window),nativeClearTimeout=window.clearTimeout.bind(window);
const add=EventTarget.prototype.addEventListener,remove=EventTarget.prototype.removeEventListener;
const percentile=(a:number[],p:number)=>Number((a[Math.min(a.length-1,Math.floor(a.length*p))]??0).toFixed(3));
const summary=(a:number[])=>{const b=[...a].sort((a,b)=>a-b);return{samples:b.length,p50ms:percentile(b,.5),p95ms:percentile(b,.95),p99ms:percentile(b,.99),maxMs:b.at(-1)??0}};
let running=false;
async function run(durationMs:number,cycles:number,spectatorCadence=false){
 if(running)return;running=true;
 const begin=performance.now(),render:number[]=[],intervals:number[]=[],setup:number[]=[],updates:number[]=[],heaps:{cycle:number;bytes:number|null}[]=[],cleanups:unknown[]=[];
 const pendingRAF=new Set<number>(),timers=new Set<number>(),timeouts=new Set<number>();
 let store:ActivityStore|null=null,unsubscribe:(()=>void)|null=null,unsubscribeScene:(()=>void)|null=null;
 let closing:Promise<void>|null=null;
 let listeners=0,contexts=0,oscillators=0,hiddenMs=0,hiddenAt=document.hidden?begin:0,previous=0,lastUpdate=0,current=-1,scene:ReturnType<typeof createWorld>|null=null,sound:AudioService|null=null,entities={};
 // Weak keys only: instrumentation must never keep destroyed canvases/listeners alive.
 const registered=new WeakMap<EventTarget,WeakMap<object,Set<string>>>();
 EventTarget.prototype.addEventListener=function(type,listener,options){if(listener){let byListener=registered.get(this);if(!byListener){byListener=new WeakMap();registered.set(this,byListener)}let keys=byListener.get(listener);if(!keys){keys=new Set();byListener.set(listener,keys)}const key=type+':'+Boolean(typeof options==='boolean'?options:options?.capture);if(!keys.has(key)){keys.add(key);listeners++}}return add.call(this,type,listener,options)};
 EventTarget.prototype.removeEventListener=function(type,listener,options){if(listener){const keys=registered.get(this)?.get(listener),key=type+':'+Boolean(typeof options==='boolean'?options:options?.capture);if(keys?.delete(key))listeners--}return remove.call(this,type,listener,options)};
 window.requestAnimationFrame=callback=>{const id=nativeRAF(at=>{pendingRAF.delete(id);const start=performance.now();callback(at);render.push(performance.now()-start)});pendingRAF.add(id);return id};
 window.cancelAnimationFrame=id=>{pendingRAF.delete(id);nativeCancel(id)};
 window.setInterval=((fn:TimerHandler,ms?:number,...args:unknown[])=>{const id=nativeInterval(fn,ms,...args);timers.add(id);return id}) as typeof window.setInterval;
 window.clearInterval=id=>{timers.delete(id);nativeClear(id)};
 window.setTimeout=((fn:TimerHandler,ms?:number,...args:unknown[])=>{const id=nativeTimeout(()=>{timeouts.delete(id);if(typeof fn==='function')fn(...args)},ms);timeouts.add(id);return id}) as typeof window.setTimeout;
 window.clearTimeout=id=>{timeouts.delete(id);nativeClearTimeout(id)};
 const heap=()=> (performance as Performance&{memory?:{usedJSHeapSize:number}}).memory?.usedJSHeapSize??null;
 const counts=()=>({raf:pendingRAF.size,intervals:timers.size,timeouts:timeouts.size,listeners,audioContexts:contexts,oscillators});
 const visibility=()=>{if(document.hidden)hiddenAt=performance.now();else if(hiddenAt){hiddenMs+=performance.now()-hiddenAt;hiddenAt=0}};
 add.call(document,'visibilitychange',visibility);
 async function dispose(){unsubscribe?.();unsubscribeScene?.();unsubscribe=unsubscribeScene=null;store?.dispose();store=null;scene?.destroy();scene=null;sound?.dispose();sound=null;await closing;closing=null;cleanups.push({cycle:current+1,...counts()});heaps.push({cycle:current+1,bytes:heap()})}
 function report(done:boolean){return{fixture:spectatorCadence?'combined-spectator-cadence-v1':'combined-public-snapshots-v1',snapshotCadencesMs:spectatorCadence?[300,600]:[100],build:import.meta.env.PROD?'Vite production fixture (production constructors)':'Vite development module fixture (production constructors)',userAgent:navigator.userAgent,dpr:devicePixelRatio,viewport:{width:canvas.clientWidth,height:canvas.clientHeight},requestedDurationMs:durationMs,elapsedMs:Math.round(performance.now()-begin),cyclesRequested:cycles,cyclesCompleted:cleanups.length,scenario:current%2?'B':'A',entities,renderCallbackCPU:summary(render),frameIntervals:summary(intervals),stallsOver50ms:intervals.filter(v=>v>50).length,setupCPU:summary(setup),snapshotUpdateCPU:summary(updates),hiddenMs:hiddenMs+(hiddenAt?performance.now()-hiddenAt:0),counts:counts(),cleanupCounts:cleanups,heapSamples:heaps,heapLimit:'Raw usedJSHeapSize includes numeric measurement samples; no forced GC or retained-heap conclusion',audioLimit:'Short production AudioService cue per lifecycle; no voice/media',done}}
 function restore(){window.requestAnimationFrame=nativeRAF;window.cancelAnimationFrame=nativeCancel;window.setInterval=nativeInterval;window.clearInterval=nativeClear;window.setTimeout=nativeTimeout;window.clearTimeout=nativeClearTimeout;EventTarget.prototype.addEventListener=add;EventTarget.prototype.removeEventListener=remove;remove.call(document,'visibilitychange',visibility);running=false}
 const safeFrame=(at:number)=>{void frame(at).catch(async error=>{try{await dispose()}finally{restore();output.textContent=JSON.stringify({failed:true,error:String(error),elapsedMs:performance.now()-begin,cleanupCounts:cleanups},null,2)}})};
 async function frame(at:number){
  if(previous)intervals.push(at-previous);previous=at;
  const elapsed=Math.max(0,at-begin),next=Math.min(cycles-1,Math.floor(elapsed/(durationMs/cycles)));
  if(next!==current){if(scene)await dispose();current=next;const start=performance.now();store=new ActivityStore();unsubscribe=store.subscribe(()=>{});unsubscribeScene=store.subscribeScene(()=>{});store.start();scene=createWorld(canvas,{stations:[]});scene.setOnline({input(){},destination(){}});scene.setZoom(.65);sound=new AudioService(()=>{const ctx=new AudioContext();contexts++;const close=ctx.close.bind(ctx);ctx.close=()=>closing=close().then(()=>{contexts--});const create=ctx.createOscillator.bind(ctx);ctx.createOscillator=()=>{const osc=create();oscillators++;add.call(osc,'ended',()=>{oscillators--},{once:true});return osc};return ctx});sound.setEnabled(true);await sound.unlock();sound.play({eventId:`cycle-${current}`,kind:'ride',mode:'fixture'});setup.push(performance.now()-start)}
  const updateCadence=spectatorCadence?(current%2?600:300):100;
  if(at-lastUpdate>=updateCadence){const start=performance.now(),data=combinedFixture(current%2?'B':'A',at,current,spectatorCadence);scene!.setPublicState({avatars:data.avatars});scene!.setActivityState(data.state);entities=data.entities;updates.push(performance.now()-start);lastUpdate=at}
  if(elapsed>=durationMs){await dispose();restore();output.textContent=JSON.stringify(report(true),null,2);running=false;return}
  if(Math.floor(elapsed/1000)!==Math.floor((elapsed-(intervals.at(-1)??0))/1000))output.textContent=JSON.stringify({running:true,elapsedMs:Math.round(elapsed),requestedDurationMs:durationMs,cycle:current+1,snapshotCadenceMs:updateCadence,scenario:current%2?'B':'A',entities,counts:counts(),cleanupCounts:cleanups,heapSamples:heaps},null,2);nativeRAF(safeFrame);
 }
 heaps.push({cycle:0,bytes:heap()});nativeRAF(safeFrame);
}
document.querySelector('#long')!.addEventListener('click',()=>void run(900000,20));
document.querySelector('#smoke')!.addEventListener('click',()=>void run(20000,4));

document.querySelector('#spectator')!.addEventListener('click',()=>void run(20000,4,true));
