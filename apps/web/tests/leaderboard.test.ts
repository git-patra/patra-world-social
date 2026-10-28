import test from 'node:test';import assert from 'node:assert/strict';
import {BoardController} from '../src/features/leaderboard/controller';
import {ChampionBoardView} from '../src/features/leaderboard/ChampionBoard';
import {renderToStaticMarkup} from 'react-dom/server';
import {validateEvent} from '../src/world/avatars';
const empty:any={worldId:'w',ruleset:'duck-v1@1',top:[],personalBest:null,lastResult:null,changes:[],administrativeChanges:[],rulesets:[]};
test('empty board has three empty ranks and no invented winners; summaries link only to world podium',()=>{
 let html=renderToStaticMarkup(ChampionBoardView({board:empty,busy:false,error:'',reload(){},select(){}}));assert.equal((html.match(/Belum ada skor sah/g)||[]).length,3);assert.match(html,/Belum ada pergantian/);assert.match(html,/Belum ada personal best/);
 const entry={userId:'u',displayName:'Synthetic',resultId:'r',score:300,hits:2,shots:3,achievedAt:100};html=renderToStaticMarkup(ChampionBoardView({board:{...empty,top:[entry],personalBest:entry,changes:[{...entry,id:1,changedAt:'2026-01-01T00:00:00Z'}]},busy:false,error:'',reload(){},select(){}}));assert.match(html,/#podium\/w\/duck-v1%401\/r/);assert.doesNotMatch(html,/\/history\//);assert.match(html,/66\.7%/);
});
test('committed event invalidates once, wrong world/rules and temporary results cannot promote a score; reconnect reloads',async()=>{
 let calls=0;const accepted:any[]=[];const c=new BoardController('w','duck-v1@1',async()=>{calls++;return {...empty,top:[{userId:'u',resultId:'r',score:300}]};},v=>accepted.push(v));
 await c.load();assert.equal(calls,1);assert.equal(c.event('leaderboard.changed',{worldId:'other',ruleset:'duck-v1@1',resultId:'r'}),false);assert.equal(c.event('leaderboard.changed',{worldId:'w',ruleset:'old',resultId:'r'}),false);assert.equal(c.event('match.result_status',{status:'finalizing',matchId:'r'}),false);
 assert.equal(c.event('leaderboard.changed',{worldId:'w',ruleset:'duck-v1@1',resultId:'r'}),true);assert.equal(c.event('leaderboard.changed',{worldId:'w',ruleset:'duck-v1@1',resultId:'r'}),false);await c.settled;assert.equal(calls,2);assert.equal(accepted.at(-1).top.length,1);await c.load();assert.equal(calls,3);c.dispose();
});
test('late board response cannot restore data after world switch/logout; retry replaces stale requests',async()=>{
 let finish:any;const accepted:any[]=[];const c=new BoardController('w','duck-v1@1',()=>new Promise(r=>finish=r),v=>accepted.push(v));const load=c.load();c.dispose();finish(empty);await load;assert.equal(accepted.length,0);
 let first:any,calls=0;const d=new BoardController('w','duck-v1@1',()=>++calls===1?new Promise(r=>first=r):Promise.resolve({...empty,top:[{score:200}]}),v=>accepted.push(v));const prior=d.load();await d.load();first(empty);await prior;assert.equal(accepted.length,1);assert.equal(accepted[0].top[0].score,200);d.dispose();
});
test('public invalidation protocol rejects expanded private payload',()=>{const event:any={v:1,eventId:'e',serverTick:1,serverTimeMs:1,type:'leaderboard.changed',payload:{worldId:'w',ruleset:'duck-v1@1',resultId:'r'}};assert.equal(validateEvent(event).type,'leaderboard.changed');assert.throws(()=>validateEvent({...event,payload:{...event.payload,score:100}}));});

import {ChampionBoard} from '../src/features/leaderboard/ChampionBoard';
import {componentLifecycle} from './component-lifecycle';
function lifecycleFixture(world='w',attachedWorld='w'){
 const saved={window:globalThis.window,fetch:globalThis.fetch};const events=new EventTarget();const requests:{url:string;signal:AbortSignal;finish:(body:any,status?:number)=>void}[]=[];
 Object.assign(globalThis,{window:events,fetch:(url:string,options:any)=>new Promise<Response>(resolve=>requests.push({url,signal:options.signal,finish:(body,status=200)=>resolve(new Response(JSON.stringify(body),{status}))}))});
 const socket:any={worldId:attachedWorld,status:'online'};let match='r';const component=componentLifecycle(()=>ChampionBoard({world,rules:'duck-v1@1',match,socket,onRules(){}}));component.flush();
 const entry={userId:'u',displayName:'Private member',resultId:'r',score:300,hits:1,shots:2,achievedAt:100};const board={...empty,worldId:world,top:[entry]},detail={...entry,status:'completed',complete:true,version:{mapId:'m',mapVersion:1,rulesId:'duck-v1',rulesVersion:1,protocolVersion:1}};
 return {component,requests,socket,board,detail,html:()=>renderToStaticMarkup(component.tree),status(state:string){socket.status=state;events.dispatchEvent(new CustomEvent('patra-world-status',{detail:{socket,state}}));component.flush()},focus(){events.dispatchEvent(new Event('focus'));component.flush()},retry(){const view=component.tree.props.children[0];view.props.reload();component.flush()},replaceMatch(id:string){match=id;component.replace()},cleanup(){component.dispose();Object.assign(globalThis,saved)}};
}
test('production component cancels board and detail together on denial and fences late responses through explicit retry',async()=>{
 const f=lifecycleFixture();try{
 assert.equal(f.requests.length,2);const old=[...f.requests];f.status('access_denied');assert.ok(old.every(r=>r.signal.aborted),'both private reads must abort on denial');assert.doesNotMatch(f.html(),/Memuat podium/);
 f.focus();assert.equal(f.requests.length,2,'focus must not revive denied lifetime');f.retry();assert.equal(f.requests.length,4);old[0].finish(f.board);old[1].finish(f.detail);await f.component.settle();assert.doesNotMatch(f.html(),/Private member|Ringkasan hasil dunia/);
 f.requests[2].finish(f.board);f.requests[3].finish(f.detail);await f.component.settle();assert.match(f.html(),/Private member/);assert.match(f.html(),/Ringkasan hasil dunia/);
 }finally{f.cleanup()}
});
test('production loaded detail is removed when a board refresh returns401 or404 and only a valid online recheck restores it',async()=>{
 for(const status of[401,404]){const f=lifecycleFixture();try{
 f.requests[0].finish(f.board);f.requests[1].finish(f.detail);await f.component.settle();assert.match(f.html(),/Ringkasan hasil dunia/);
 f.focus();assert.equal(f.requests.length,3);f.requests[2].finish({error:'denied'},status);await f.component.settle();assert.doesNotMatch(f.html(),/Private member|Ringkasan hasil dunia|Memuat podium/);f.focus();assert.equal(f.requests.length,3);
 f.status('online');assert.equal(f.requests.length,3,'repeated online status must not revive denied reads');f.status('reconnecting');f.status('online');assert.equal(f.requests.length,5,'matching reconnect must recheck denied access');f.requests[3].finish(f.board);f.requests[4].finish(f.detail);await f.component.settle();assert.match(f.html(),/Ringkasan hasil dunia/);
 }finally{f.cleanup()}}
});
test('production detail denial fences pending board and ignores late transient failures after a fresh retry',async()=>{
 const f=lifecycleFixture();try{
 f.requests[1].finish({error:'denied'},404);await f.component.settle();assert.ok(f.requests[0].signal.aborted);f.retry();assert.equal(f.requests.length,4);
 f.requests[2].finish(f.board);f.requests[3].finish(f.detail);await f.component.settle();f.requests[0].finish({error:'temporary'},503);await f.component.settle();assert.match(f.html(),/Private member/);assert.match(f.html(),/Ringkasan hasil dunia/);assert.doesNotMatch(f.html(),/Podium belum dapat dimuat/);
 }finally{f.cleanup()}
});
test('production deep link ignores denial from another world and replaces detail request on route change',async()=>{
 const f=lifecycleFixture('linked-world','attached-world');try{
 f.status('access_denied');assert.ok(f.requests.every(r=>!r.signal.aborted));f.requests[0].finish(f.board);f.requests[1].finish(f.detail);await f.component.settle();assert.match(f.html(),/Ringkasan hasil dunia/);f.status('access_denied');assert.match(f.html(),/Ringkasan hasil dunia/,'unrelated socket denial cannot erase an authorized deep link');
 f.replaceMatch('new-result');assert.ok(f.requests[1].signal.aborted,'route replacement cleans up prior detail');assert.doesNotMatch(f.html(),/Ringkasan hasil dunia/);const detailRequest=f.requests.find(r=>r.url.endsWith('/new-result'));assert.ok(detailRequest);detailRequest.finish({...f.detail,resultId:'new-result'});await f.component.settle();assert.match(f.html(),/Ringkasan hasil dunia/);
 }finally{f.cleanup()}
});
