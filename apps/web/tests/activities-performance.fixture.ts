import {createWorld} from '../src/world/controller';
import type {Avatar} from '../src/world/avatars';
// Synthetic scene only: no auth, sockets, API calls, or production state.
const canvas=document.querySelector<HTMLCanvasElement>('canvas')!;
const output=document.querySelector<HTMLOutputElement>('output')!;
const report={fixture:'20 deterministic avatars, fixed 1280×720 CSS viewport',userAgent:navigator.userAgent,dpr:devicePixelRatio,frames:0,p50ms:0,p95ms:0,p99ms:0,fps:0,heapBytes:null as number|null,elapsedMs:0,done:false};
const originalRAF=window.requestAnimationFrame.bind(window);
const costs:number[]=[],intervals:number[]=[];
window.requestAnimationFrame=callback=>originalRAF(time=>{const start=performance.now();callback(time);costs.push(performance.now()-start)});
const scene=createWorld(canvas,{stations:[]});
scene.setOnline({input(){},destination(){}});
let begin=0,previous=0,stopped=false;
const avatars:Avatar[]=Array.from({length:20},(_,i)=>({id:`fixture-${i}`,name:`Player ${i+1}`,avatarId:'pumpkin',x:28+i%5,y:28+Math.floor(i/5),status:'available',seq:0}));
function frame(now:number){
 if(stopped)return;
 if(!begin)begin=now;
 if(previous)intervals.push(now-previous);previous=now;
 scene.setPublicState({avatars:avatars.map((a,i)=>({...a,x:a.x+Math.sin(now/900+i)*2,y:a.y+Math.cos(now/1100+i)*2}))});
 report.frames++;report.elapsedMs=Math.round(now-begin);
 if(now-begin>=15000){
  stopped=true;scene.destroy();window.requestAnimationFrame=originalRAF;costs.sort((a,b)=>a-b);
  const percentile=(p:number)=>Number((costs[Math.min(costs.length-1,Math.floor(costs.length*p))]||0).toFixed(3));
  report.p50ms=percentile(.5);report.p95ms=percentile(.95);report.p99ms=percentile(.99);
  report.fps=Number((intervals.length*1000/intervals.reduce((a,b)=>a+b,0)).toFixed(2));
  report.heapBytes=(performance as Performance&{memory?:{usedJSHeapSize:number}}).memory?.usedJSHeapSize??null;
  report.done=true;output.textContent=JSON.stringify(report,null,2);return;
 }
 output.textContent=`Profiling synthetic scene… ${Math.floor((now-begin)/1000)} / 15 seconds`;
 originalRAF(frame);
}
originalRAF(frame);
