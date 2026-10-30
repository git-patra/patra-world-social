import test from 'node:test';
import assert from 'node:assert/strict';
import {ActivityStore} from '../src/activities/store';
import {InputRouter} from '../src/activities/input-router';
import {LocalRace,type RaceSnapshot} from '../src/features/race/local';
const actors=[{actorId:'a',actorType:'human',userId:'me',displayName:'Me',avatarId:'pumpkin'}];
const race=(id='match')=>new LocalRace(id,actors,3000).snapshot();
const socket=()=>({worldId:'world',selfId:'me',status:'online',serverNow:()=>1000,commands:[] as string[],async gameCommand(type:string){this.commands.push(type);return type==='lobby.list'?{lobbies:[]}:type==='activity.snapshot'?{activities:[]}:race()}});
const lobby=(id='match',kind='race')=>({id:'lobby',arenaId:kind,gameType:kind,matchId:id,state:'active',worldId:'world',players:[{userId:'me',actorId:'a'}],spectators:[]});
const area=(generation=1,phase='ACTIVE')=>({worldId:'world',areaId:'race',incarnation:'runtime',generation,serverTimeMs:1000,tick:generation,kind:'area',payload:{phase,matchId:'match',startsAt:0,clearAt:0,resetAt:0}});
test('closing_menu_does_not_unsubscribe_game',async()=>{const store=new ActivityStore();const s=socket();await store.attach(s);store.receive('lobby.updated',{worldId:'world',lobbies:[lobby()]});const close=store.subscribe(()=>{});close();const next={...race(),frame:17};store.receive('race.snapshot',next);assert.equal((store.snapshot().games[0]?.snapshot as RaceSnapshot)?.frame,17);assert.equal(store.snapshot().active?.matchId,'match');store.dispose()});
test('one_input_owner_only',()=>{const router=new InputRouter();const calls:string[]=[];router.register('walk',()=>calls.push('walk'));router.register('race',()=>calls.push('race'));router.setOwner('race');calls.length=0;router.direction(0,-1);assert.deepEqual(calls,['race']);router.setOwner('walk');calls.length=0;router.direction(1,0);assert.deepEqual(calls,['walk'])});
test('modal_and_text_neutralize_active_mode',()=>{const router=new InputRouter();const frames:any[]=[];router.register('race',input=>frames.push({...input}));router.setOwner('race');router.direction(0,-1);router.block('modal',true);assert.deepEqual(frames.at(-1),{x:0,y:0,throttle:0,brake:0,steer:0});const count=frames.length;router.direction(1,0);assert.equal(frames.length,count);router.block('modal',false);router.direction(1,0);router.block('text',true);assert.deepEqual(frames.at(-1),{x:0,y:0,throttle:0,brake:0,steer:0})});
test('passive_spectator_has_no_input_or_history',async()=>{const store=new ActivityStore();const s=socket();await store.attach(s);store.receive('lobby.updated',{worldId:'world',lobbies:[{...lobby(),players:[]}]});store.receive('race.snapshot',race());store.controls({throttle:1,brake:0,steer:0});store.tick(4000);assert.equal(store.snapshot().active,null);assert.equal(store.snapshot().games.length,1);assert.equal(s.commands.some(c=>c==='lobby.join'||c==='race.input'),false);store.dispose()});
test('resync_swaps_generation_without_replaying_effects',async()=>{let effects=0;const store=new ActivityStore(()=>effects++);const s=socket();await store.attach(s);store.resync([area(1)] as any);store.receive('activity.lifecycle',area(2,'RESETTING'));assert.equal(effects,1);store.resync([area(3,'OPEN')] as any);store.receive('activity.lifecycle',area(2));assert.equal(store.snapshot().areas[0].generation,3);assert.equal(effects,1);store.dispose()});
test('local_online_local_clears_ownership_and_two_public_matches_are_independent',async()=>{const store=new ActivityStore();await store.startLocal('race',1000,42,'local-race');assert.equal(store.snapshot().active?.matchId,'local-race');const s=socket();await store.attach(s);assert.equal(store.snapshot().games.length,0);store.receive('lobby.updated',{worldId:'world',lobbies:[lobby('one'),lobby('two','penalty')]});store.receive('race.snapshot',race('one'));store.receive('penalty.snapshot',{matchId:'two',phase:'choosing',actors:[],scores:[0,0],chosen:[],turns:[]});assert.equal(store.snapshot().games.length,2);await store.attach(null);assert.equal(store.snapshot().games.length,0);assert.equal(store.snapshot().active,null);store.dispose()});
test('real_local_match_runs_while_all_menu_subscribers_are_closed',async()=>{const store=new ActivityStore();await store.startLocal('race',1000,42,'durable-local');const close=store.subscribe(()=>{});close();store.tick(5000);assert.equal(store.snapshot().games[0].snapshot.phase,'racing');assert.equal((store.snapshot().games[0].snapshot as RaceSnapshot).startsAt,4000);store.dispose()});

test('late_local_lazy_load_cannot_restore_a_session_after_attachment_reset',async()=>{const store=new ActivityStore();const pending=store.startLocal('race',1000,42,'obsolete');await store.attach(null);await pending;assert.equal(store.snapshot().active,null);assert.equal(store.snapshot().games.length,0);store.dispose()});

test('passive_public_finished_snapshot_does_not_write_device_history',async()=>{const {IDBFactory}=await import('fake-indexeddb');const {guestHistory}=await import('../src/guest/history');const previous=globalThis.indexedDB;globalThis.indexedDB=new IDBFactory();const store=new ActivityStore();try{const s=socket();await store.attach(s);store.receive('lobby.updated',{worldId:'world',lobbies:[{...lobby(),players:[]}]});store.receive('race.finished',{...race(),phase:'finished',finishedAt:4000});store.tick(5000);assert.deepEqual(await guestHistory.list(),[]);assert.equal(store.snapshot().active,null)}finally{store.dispose();globalThis.indexedDB=previous}});

test('public_scene_replaces_walking_actor_and_uses_the_exact_physics_art_transform',async()=>{const {ActivityScene}=await import('../src/activities/scene');const {racePoint}=await import('../src/world/scenes/race');const store=new ActivityStore();await store.attach(socket());store.receive('lobby.updated',{worldId:'world',lobbies:[lobby()]});store.receive('race.snapshot',race());let changed!:()=>void;const ready=new Promise<void>(resolve=>{changed=resolve});const scene=new ActivityScene(changed);scene.set(store.snapshot());assert.equal(scene.replaces('me'),true);await ready;const actors=scene.actors(1000);assert.equal(actors.length,1);assert.equal(scene.ownActor(actors)?.userId,'me');assert.deepEqual(racePoint({x:2048,y:2048}),{x:1520,y:1184});store.dispose()});

test('installed_input_owner_releases_keyboard_on_blur_visibility_and_text',()=>{
 const old={window:globalThis.window,document:globalThis.document,Element:globalThis.Element,MutationObserver:globalThis.MutationObserver};
 class Surface extends EventTarget{constructor(public editing=false){super()}closest(){return this.editing?this:null}}
 const events=new Surface(),doc=Object.assign(new Surface(),{hidden:false,body:{dataset:{}},querySelector:()=>null});
 Object.assign(globalThis,{window:events,document:doc,Element:Surface,MutationObserver:class{observe(){}disconnect(){}}});
 const router=new InputRouter(),frames:any[]=[];router.register('race',input=>frames.push(input));router.setOwner('race');const cleanup=router.install();
 try{
  const key=()=>events.dispatchEvent(Object.assign(new Event('keydown',{cancelable:true}),{key:'w'}));
  for(const kind of ['blur']){key();assert.equal(frames.at(-1).throttle,1);events.dispatchEvent(new Event(kind));assert.equal(frames.at(-1).throttle,0)}
  key();doc.hidden=true;doc.dispatchEvent(new Event('visibilitychange'));assert.equal(frames.at(-1).throttle,0);const count=frames.length;key();assert.equal(frames.length,count);
  doc.hidden=false;doc.dispatchEvent(new Event('visibilitychange'));key();const focus=new Event('focusin');Object.defineProperty(focus,'target',{value:new Surface(true)});doc.dispatchEvent(focus);assert.equal(frames.at(-1).throttle,0);assert.equal(router.mode(),'modal');
  cleanup();const after=frames.length;events.dispatchEvent(new Event('blur'));assert.equal(frames.length,after);
 }finally{cleanup();Object.assign(globalThis,old)}
});

test('HUD_projection_is_limited_to_ten_notifications_per_second',async()=>{const store=new ActivityStore();await store.startLocal('race',1000,42,'throttled');let notifications=0;const close=store.subscribe(()=>notifications++);for(let at=1001;at<=2000;at++)store.tick(at);assert.equal(notifications,10);close();store.dispose()});

test('explicit_guest_exit_neutralizes_unfinished_match_without_history_and_allows_next_game',async()=>{const {IDBFactory}=await import('fake-indexeddb');const {guestHistory}=await import('../src/guest/history');const previous=globalThis.indexedDB;globalThis.indexedDB=new IDBFactory();const store=new ActivityStore();try{await store.startLocal('race',1000,42,'unfinished-race');store.controls({throttle:1,steer:1,brake:0});store.tick(4100);await store.action('leave');assert.equal(store.snapshot().active,null);store.tick(500000);assert.deepEqual(await guestHistory.list(),[]);await store.startLocal('penalty',500001,undefined,'next-duel');assert.equal(store.snapshot().active?.kind,'penalty');assert.equal(store.snapshot().active?.matchId,'next-duel')}finally{store.dispose();globalThis.indexedDB=previous}});

test('spatial_reopen_removes_finished_projection_while_result_claim_stays_explicit',async()=>{const {ActivityScene}=await import('../src/activities/scene');const store=new ActivityStore();await store.attach(socket());store.receive('lobby.updated',{worldId:'world',lobbies:[{...lobby(),state:'results'}]});store.receive('race.finished',{...race(),phase:'finished'});let loaded!:()=>void;const ready=new Promise<void>(resolve=>{loaded=resolve});const scene=new ActivityScene(loaded);scene.set(store.snapshot());await ready;assert.equal(scene.actors(1000).length,1);store.resync([{...area(2,'OPEN'),payload:{...area().payload,phase:'OPEN',matchId:''}}] as any);scene.set(store.snapshot());assert.equal(scene.actors(1001).length,0);assert.equal(scene.replaces('me'),false);assert.equal(store.snapshot().inputMode,'walk');assert.equal(store.snapshot().active?.matchId,'match');store.dispose()});

test('finished_participant_stops_sending_controls_while_other_karts_continue',async()=>{const store=new ActivityStore(),s=socket();await store.attach(s);store.receive('lobby.updated',{worldId:'world',lobbies:[lobby()]});const snapshot={...race(),phase:'racing'};snapshot.karts[0].finished=true;store.receive('race.snapshot',snapshot);s.commands.length=0;store.controls({throttle:1,steer:0,brake:0});store.tick(4000);assert.equal(s.commands.includes('race.input'),false);store.dispose()});

test('penalty_scene_never_animates_a_guessed_ball_before_authoritative_reveal',async()=>{const {sceneActors}=await import('../src/world/scenes/penalty');const game={kind:'penalty',matchId:'p',snapshot:{matchId:'p',phase:'choosing',choiceDeadlineAt:3000,actors:[],turns:[]}};const ball=sceneActors(game as any,9000).find(a=>a.id==='penalty:ball');assert.equal(ball?.y,1498)});

test('installed_pointer_owners_keep_other_thumb_and_keyboard_holds_until_their_own_release',()=>{
 const old={window:globalThis.window,document:globalThis.document,Element:globalThis.Element,MutationObserver:globalThis.MutationObserver};
 class Surface extends EventTarget{closest(){return null}}
 const events=new Surface(),doc=Object.assign(new Surface(),{hidden:false,body:{dataset:{}},querySelector:()=>null});
 Object.assign(globalThis,{window:events,document:doc,Element:Surface,MutationObserver:class{observe(){}disconnect(){}}});
 const router=new InputRouter(),frames:any[]=[];router.register('race',input=>frames.push(input));router.setOwner('race');const cleanup=router.install();
 const key=(type:string,key:string)=>events.dispatchEvent(Object.assign(new Event(type,{cancelable:true}),{key}));
 const pointer=(type:string,pointerId:number)=>events.dispatchEvent(Object.assign(new Event(type),{pointerId}));
 try{
  // The same two independent pointer IDs used by the Gas and steering HUD buttons.
  router.pointerControl(11,'throttle',1);router.pointerControl(22,'steer',-1);
  pointer('pointerup',22);
  assert.equal(frames.at(-1).throttle,1,'releasing steering must preserve the gas thumb');
  assert.equal(frames.at(-1).steer,0,'only the released steering pointer becomes neutral');
  pointer('lostpointercapture',11);assert.equal(frames.at(-1).throttle,0,'lost gas capture releases its own hold');
  router.pointerControl(11,'throttle',1);router.pointerControl(22,'steer',1);pointer('pointercancel',22);assert.equal(frames.at(-1).throttle,1);assert.equal(frames.at(-1).steer,0);
  events.dispatchEvent(new Event('blur'));assert.equal(frames.at(-1).throttle,0);pointer('pointerup',11);router.repeat();assert.equal(frames.at(-1).throttle,0);
  key('keydown','w');pointer('pointerup',99);assert.equal(frames.at(-1).throttle,1,'unrelated pointer release must preserve W');
  key('keydown','d');key('keyup','d');assert.equal(frames.at(-1).throttle,1,'W remains in the actual held-key set');
  key('keyup','w');assert.equal(frames.at(-1).throttle,0);
  for(const reason of ['modal','disconnected']){key('keydown','w');router.pointerControl(11,'throttle',1);router.pointerControl(22,'steer',1);router.block(reason,true);assert.deepEqual(frames.at(-1),{x:0,y:0,throttle:0,brake:0,steer:0});router.block(reason,false);pointer('pointerup',11);key('keydown','d');assert.equal(frames.at(-1).throttle,0,'cleared W and gas cannot revive after a context reset');key('keyup','d')}
  router.pointerControl(11,'throttle',1);doc.hidden=true;doc.dispatchEvent(new Event('visibilitychange'));assert.equal(frames.at(-1).throttle,0);doc.hidden=false;doc.dispatchEvent(new Event('visibilitychange'));pointer('lostpointercapture',11);router.repeat();assert.equal(frames.at(-1).throttle,0);
 }finally{cleanup();Object.assign(globalThis,old)}
});

const deferred=<T,>()=>{let resolve!:(value:T)=>void;const promise=new Promise<T>(done=>{resolve=done});return{promise,resolve}};
const penalty=(turn:number,phase='choosing',chosen=[false,false])=>({matchId:'duel',phase,turn,turnId:`duel:${turn}`,choiceOpensAt:turn*4500,choiceDeadlineAt:turn*4500+3000,animationEndsAt:turn*4500+4500,actors:[],scores:[turn-1,0],chosen,turns:[]});

test('ordered_lifecycle_accepts_reserved_to_active_in_the_same_owner_tick',async()=>{
 let effects=0;const store=new ActivityStore(()=>effects++);await store.attach(socket());
 store.resync([{...area(1,'RESERVED'),tick:20}] as any);
 store.receive('activity.lifecycle',{...area(1,'ACTIVE'),tick:20});
 assert.equal(store.snapshot().areas[0].payload.phase,'ACTIVE');assert.equal(effects,1);
 store.receive('activity.snapshot',{...area(1,'RESERVED'),tick:21});
 assert.equal(store.snapshot().areas[0].payload.phase,'ACTIVE','a later publish cannot regress a lease phase');store.dispose();
});

test('slow_resync_merges_against_newer_live_generation_without_replaying_effects',async()=>{
 let effects=0;const store=new ActivityStore(()=>effects++),s=socket();s.status='connecting';await store.attach(s);s.status='online';
 store.resync([area(1)] as any);const metadata=deferred<unknown>(),lobbies=deferred<unknown>();
 s.gameCommand=async(type:string)=>{s.commands.push(type);return(type==='activity.snapshot'?metadata:lobbies).promise as any};
 const reload=store.reload();metadata.resolve({activities:[area(1)]});
 store.receive('activity.lifecycle',area(2,'RESETTING'));lobbies.resolve({worldId:'world',lobbies:[]});await reload;
 assert.equal(store.snapshot().areas[0].generation,2);assert.equal(store.snapshot().areas[0].payload.phase,'RESETTING');assert.equal(effects,1);store.dispose();
});

test('newer_resync_incarnation_fences_older_pending_reads_and_old_runtime_events',async()=>{
 const store=new ActivityStore(),s=socket();s.status='connecting';await store.attach(s);s.status='online';store.resync([area(8)] as any);
 const oldMetadata=deferred<unknown>(),oldLobbies=deferred<unknown>();let reads=0;
 s.gameCommand=async(type:string)=>{s.commands.push(type);if(reads++<2)return(type==='activity.snapshot'?oldMetadata:oldLobbies).promise as any;return(type==='activity.snapshot'?{activities:[{...area(1),incarnation:'runtime-new'}]}:{worldId:'world',lobbies:[]}) as any};
 const oldReload=store.reload();await store.reload();assert.equal(store.snapshot().areas[0].incarnation,'runtime-new');
 oldMetadata.resolve({activities:[area(8)]});oldLobbies.resolve({worldId:'world',lobbies:[]});await oldReload;
 store.receive('activity.lifecycle',area(9,'RESETTING'));store.resync([area(10)] as any);
 assert.equal(store.snapshot().areas[0].incarnation,'runtime-new');assert.equal(store.snapshot().areas[0].generation,1);store.dispose();
});

test('penalty_public_progress_never_rewinds_turn_reveal_or_locked_chosen_flags',async()=>{
 const store=new ActivityStore();await store.attach(socket());store.receive('lobby.updated',{worldId:'world',lobbies:[lobby('duel','penalty')]});
 store.receive('penalty.turn_opened',penalty(1));store.receive('penalty.snapshot',penalty(1,'choosing',[true,false]));store.receive('penalty.snapshot',penalty(1));
 assert.deepEqual((store.snapshot().games[0].snapshot as any).chosen,[true,false]);
 store.receive('penalty.revealed',penalty(1,'animation',[true,true]));store.receive('penalty.snapshot',penalty(1));
 assert.equal(store.snapshot().games[0].snapshot.phase,'animation');
 store.receive('penalty.turn_opened',penalty(2));store.receive('penalty.revealed',penalty(1,'animation',[true,true]));
 assert.equal((store.snapshot().games[0].snapshot as any).turn,2);assert.equal(store.snapshot().games[0].snapshot.phase,'choosing');assert.deepEqual((store.snapshot().games[0].snapshot as any).scores,[1,0]);store.dispose();
});

import {ActivityScene} from '../src/activities/scene';
import type {ActivityState} from '../src/activities/store';
import {racePoint} from '../src/world/scenes/race';
function interpolationState(at:number,x:number,options:{id?:string;phase?:string;generation?:number;incarnation?:string;actorId?:string}={}):ActivityState{
 const id=options.id??'interpolated',snapshot={...race(id),phase:options.phase??'racing',serverTimeMs:at,karts:race(id).karts.map(k=>({...k,actorId:options.actorId??k.actorId,x,y:2048}))};
 return{games:[{kind:'race',matchId:id,snapshot}],areas:[{...area(options.generation??1),incarnation:options.incarnation??'runtime',payload:{phase:'ACTIVE',matchId:id,startsAt:1000,clearAt:0,resetAt:0}}] as any,fishing:[],nearbyFishing:null,claw:null,nearbyClaw:false,football:null,nearbyFootball:false,vehicles:[],nearbyRide:null,active:null,spectating:null,inputMode:'walk',lobbies:[],selfId:'watcher',online:true,connected:true,now:at,message:''};
}
for(const cadence of [300,600,1600])test(`production_scene_interpolates_${cadence}ms_spectator_updates`,async t=>{
 let now=1000;t.mock.method(performance,'now',()=>now);
 let ready!:()=>void;const loaded=new Promise<void>(resolve=>ready=resolve),scene=new ActivityScene(()=>ready());scene.set(interpolationState(1000,2048));await loaded;
 now+=cadence;scene.set(interpolationState(now,3048));
 assert.equal(scene.actors(now)[0].x,racePoint({x:2048,y:2048}).x,'receipt starts from previous public pose');
 now+=Math.min(cadence,600)/2;assert.equal(scene.actors(now)[0].x,racePoint({x:2548,y:2048}).x,'half interval produces an interpolated pose');
 now+=Math.min(cadence,600)/2;assert.equal(scene.actors(now)[0].x,racePoint({x:3048,y:2048}).x,'reaches newest public pose without extrapolation');
 now+=5000;assert.equal(scene.actors(now)[0].x,racePoint({x:3048,y:2048}).x,'stale pose never extrapolates');
});
for(const boundary of ['stale-server','stale-receipt','phase','match','actor','generation','incarnation'] as const)test(`production_scene_snaps_${boundary}_boundary`,async t=>{
 let now=1000;t.mock.method(performance,'now',()=>now);
 let ready!:()=>void;const loaded=new Promise<void>(resolve=>ready=resolve),scene=new ActivityScene(()=>ready());scene.set(interpolationState(1000,2048));await loaded;
 now=boundary==='stale-receipt'?3501:1100;
 const next=interpolationState(boundary==='stale-server'?3501:1100,3048,{phase:boundary==='phase'?'finished':'racing',id:boundary==='match'?'next':'interpolated',actorId:boundary==='actor'?'replacement':'a',generation:boundary==='generation'?2:1,incarnation:boundary==='incarnation'?'new-runtime':'runtime'});
 scene.set(next);assert.equal(scene.actors(now)[0].x,racePoint({x:3048,y:2048}).x,'identity/phase/stale boundaries must not blend obsolete positions');
});

test('spectator_quick_B_renders_four_riders_without_changing_historical_fixture_defaults',async()=>{
 const {combinedFixture}=await import('./activities-combined-data');
 const historical=combinedFixture('B',1000,1),quick=combinedFixture('B',1000,1,true);
 assert.deepEqual(historical.state.areas,[],'historical100ms fixture stays unchanged');
 const scene=new ActivityScene(()=>{});scene.set(quick.state);
 await new Promise<void>(resolve=>setImmediate(resolve));
 const vehicles=scene.actors(1000).filter(a=>a.id.startsWith('vehicle:'));
 assert.equal(vehicles.length,4);assert.deepEqual(vehicles.map(v=>v.userId),['fixture-0','fixture-1','fixture-2','fixture-3']);
 assert.ok(scene.replaces('fixture-0'),'rider replaces the walking avatar');
 assert.equal(quick.state.areas.find(a=>a.areaId==='race')?.payload.phase,'OPEN');
 assert.deepEqual(combinedFixture('A',1000,0,true).state.areas,[],'closed-race workload does not acquire casual OPEN state');
});

const flushReads=()=>new Promise<void>(resolve=>setImmediate(resolve));
test('lobby_metadata_reads_only_missing_matches_and_explicit_resync_refreshes_all',async()=>{
 const store=new ActivityStore(),commands:{type:string;matchId?:string}[]=[];
 let lobbies=['race','penalty','pingpong','blaster'].map(kind=>lobby(kind,kind));
 const s={worldId:'world',selfId:'observer',status:'online',serverNow:()=>1000,async gameCommand(type:string,payload:any){commands.push({type,matchId:payload.matchId});if(type==='activity.snapshot')return{activities:[]};if(type==='lobby.list')return{worldId:'world',lobbies};return{...race(payload.matchId),actors:[],chosen:[],turn:1}}};
 try{
  await store.attach(s);assert.equal(store.snapshot().games.length,4);commands.length=0;
  for(let i=0;i<3;i++){store.receive('lobby.updated',{worldId:'world',lobbies:lobbies.map(l=>({...l,spectators:[`watcher-${i}`]}))});await flushReads()}
  assert.equal(commands.length,0,'three ordinary updates must not produce 12 snapshot control reads');
  lobbies=lobbies.map(l=>l.gameType==='race'?{...l,matchId:'new-race'}:l);store.receive('lobby.updated',{worldId:'world',lobbies});await flushReads();
  assert.deepEqual(commands,[{type:'race.snapshot',matchId:'new-race'}]);assert.ok(store.snapshot().games.some(g=>g.matchId==='new-race'));
  commands.length=0;await store.reload();assert.equal(commands.length,6,'explicit reconnect/resync reads metadata, lobbies and every match');
 }finally{store.dispose()}
});
test('lobby_missing_reads_deduplicate_pending_retry_failure_and_fence_old_identity',async(t)=>{
 let now=1000;t.mock.method(performance,'now',()=>now);
 const store=new ActivityStore(),reads:{id:string;resolve:(value:any)=>void;reject:(error:Error)=>void}[]=[];
 const s={worldId:'world',selfId:'observer',status:'online',serverNow:()=>1000,async gameCommand(type:string,payload:any):Promise<any>{if(type==='activity.snapshot')return{activities:[]};if(type==='lobby.list')return{lobbies:[]};return new Promise((resolve,reject)=>reads.push({id:payload.matchId,resolve,reject}))}};
 const update=(id:string)=>store.receive('lobby.updated',{worldId:'world',lobbies:[lobby(id)]});
 try{
  await store.attach(s);update('old');update('old');update('old');assert.equal(reads.length,1,'in-flight missing match is read once');
  reads[0].reject(new Error('transient'));await flushReads();for(let i=0;i<20;i++)update('old');assert.equal(reads.length,1,'failed-read storm is bounded by retry cooldown');
  now+=1000;update('old');assert.equal(reads.length,2,'metadata retries a failed read after cooldown');
  update('new');assert.equal(reads.length,3);reads[1].resolve(race('old'));reads[2].resolve(race('new'));await flushReads();assert.deepEqual(store.snapshot().games.map(g=>g.matchId),['new']);
 }finally{store.dispose()}
});
test('explicit_resync_supersedes_pending_reads_and_bypasses_failure_cooldown',async()=>{
 const store=new ActivityStore(),reads:{resolve:(value:any)=>void;reject:(error:Error)=>void}[]=[];
 let lobbies:ReturnType<typeof lobby>[]=[];
 const s={worldId:'world',selfId:'observer',status:'online',serverNow:()=>1000,async gameCommand(type:string):Promise<any>{if(type==='activity.snapshot')return{activities:[]};if(type==='lobby.list')return{lobbies};return new Promise((resolve,reject)=>reads.push({resolve,reject}))}};
 try{
  await store.attach(s);lobbies=[lobby('same')];store.receive('lobby.updated',{worldId:'world',lobbies});
  const resync=store.reload();await flushReads();assert.equal(reads.length,2,'full resync supersedes pending missing read');
  reads[0].resolve({...race('same'),frame:100});reads[1].reject(new Error('transient'));await resync;assert.equal(store.snapshot().games.length,0,'old read remains fenced');
  const retry=store.reload();await flushReads();assert.equal(reads.length,3,'explicit resync bypasses ordinary failure cooldown');reads[2].resolve(race('same'));await retry;assert.equal(store.snapshot().games.length,1);
 }finally{store.dispose()}
});
