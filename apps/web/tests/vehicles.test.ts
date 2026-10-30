import test from 'node:test';
import assert from 'node:assert/strict';
import {validateActivitySnapshot} from '../src/activities/contracts';
import {ActivityStore} from '../src/activities/store';
const pool=()=>({worldId:'world',areaId:'race',incarnation:'runtime',generation:1,serverTimeMs:1000,tick:20,kind:'vehicles',payload:{vehicles:[{id:'bicycle-0',kind:'bicycle',generation:1,riderId:'',x:39,y:46,heading:0},{id:'bicycle-1',kind:'bicycle',generation:1,riderId:'',x:41,y:46,heading:0},{id:'car-2',kind:'car',generation:1,riderId:'',x:43,y:46,heading:0},{id:'car-3',kind:'car',generation:1,riderId:'',x:45,y:46,heading:0}]}});
test('vehicle_snapshot_is_exact_and_keeps_all_four_area_masks',async()=>{
 const snapshot=pool();assert.equal(validateActivitySnapshot(snapshot).kind,'vehicles');
 const socket={worldId:'world',selfId:'me',status:'online',serverNow:()=>1000,async gameCommand(type:string){return type==='lobby.list'?{lobbies:[]}:{activities:[]}}};
 const store=new ActivityStore();await store.attach(socket);
 const areas=['race','penalty','pingpong','blaster'].map(areaId=>({...snapshot,kind:'area',areaId,payload:{phase:'OPEN',matchId:'',startsAt:0,clearAt:0,resetAt:0}}));
 store.resync([...areas,snapshot] as any);assert.equal(store.snapshot().areas.length,4);assert.equal((store.snapshot() as any).vehicles.length,4);store.dispose();
});
test('guest_ride_requires_proximity_moves_on_store_clock_and_race_clears_before_countdown',async()=>{
 const store=new ActivityStore();
 assert.equal((store as any).interactRide({x:10,y:10}),false);
 assert.equal((store as any).interactRide({x:39,y:46}),true);
 assert.equal(store.snapshot().inputMode,'ride');
 const start=store.snapshot().now;
 for(let step=1;step<=20;step++){(store as any).rideControls({x:1,y:0});store.tick(start+step*50)}
 assert.ok(Math.abs((store.snapshot() as any).vehicles[0].x-45)<.001);
 assert.equal(store.snapshot().games.length,0,'casual ride creates no fake match');
 const pending=store.startLocal('race',start+1100,42,'race-clear');
 assert.equal((store.snapshot() as any).vehicles.some((v:any)=>v.riderId),false,'riders clear before lazy race startup');
 await pending;assert.equal((store as any).interactRide({x:39,y:46}),false);assert.equal(store.snapshot().active?.kind,'race');store.clearLocal();assert.equal((store as any).interactRide({x:39,y:46}),true);store.dispose();
});
test('vehicle_generations_do_not_replace_masks_or_resurrect_old_riders',async()=>{
 const store=new ActivityStore();const sent:{type:string,payload:any}[]=[];const socket={worldId:'world',selfId:'me',status:'online',serverNow:()=>1000,async gameCommand(type:string,payload:any){sent.push({type,payload});return type==='lobby.list'?{lobbies:[]}:{activities:[]}}};await store.attach(socket);
 const open={...pool(),kind:'area',generation:0,payload:{phase:'OPEN',matchId:'',startsAt:0,clearAt:0,resetAt:0}};
 const mounted=pool();mounted.generation=2;mounted.payload.vehicles[0].riderId='me';mounted.payload.vehicles[0].generation=2;
 store.resync([open,mounted] as any);assert.equal(store.snapshot().inputMode,'ride');store.rideControls({x:1,y:0});store.tick(1001);await Promise.resolve();assert.equal(sent.at(-1)?.type,'activity.ride');assert.equal(sent.at(-1)?.payload.dx,1);
 const released=pool();released.generation=3;released.payload.vehicles[0].generation=3;store.receive('activity.lifecycle',released);store.receive('activity.snapshot',{...mounted,tick:1000});assert.equal(store.snapshot().inputMode,'walk');assert.equal(store.snapshot().vehicles[0].riderId,'');assert.equal(store.snapshot().areas.length,1);
 store.receive('activity.lifecycle',{...open,generation:1,payload:{phase:'CLEARING',matchId:'match',startsAt:0,clearAt:4000,resetAt:0}});assert.equal(store.interactRide({x:39,y:46}),false);
 const fresh={...released,incarnation:'runtime-new',generation:1};store.resync([fresh] as any);store.receive('activity.snapshot',{...released,generation:20});store.resync([{...released,generation:21}] as any);assert.equal(store.snapshot().vehicles[0].riderId,'');store.dispose();
});
test('guest_dismount_is_immediate_neutral_and_never_completes_a_match',async()=>{
 const store=new ActivityStore();store.interactRide({x:43,y:46});const start=store.snapshot().now;store.rideControls({x:1,y:0});store.tick(start+50);const x=store.snapshot().vehicles[2].x;assert.ok(x>43);store.interactRide();assert.equal(store.snapshot().inputMode,'walk');store.tick(start+10000);assert.equal(store.snapshot().vehicles[2].x,x);assert.equal(store.snapshot().games.length,0);assert.equal(store.snapshot().active,null);store.dispose();
});
test('casual_scene_uses_world_projection_and_replaces_only_the_current_rider',async()=>{
 const {ActivityScene}=await import('../src/activities/scene');const store=new ActivityStore();store.interactRide({x:39,y:46});let resolve!:()=>void;const ready=new Promise<void>(r=>resolve=r);const scene=new ActivityScene(resolve);scene.set(store.snapshot());await ready;assert.equal(scene.actors(0).filter(a=>a.id.startsWith('vehicle:')).length,4);assert.equal(scene.replaces('guest'),true);assert.equal(scene.replaces('spectator'),false);assert.equal(scene.ownActor(scene.actors(0))?.x,1580);store.interactRide();scene.set(store.snapshot());assert.equal(scene.replaces('guest'),false);store.dispose();
});
test('guest_vehicle_catchup_is_capped_at_four_steps',async()=>{
 const {LocalVehicles}=await import('../src/activities/vehicles');const local=new LocalVehicles();local.mount('car-2',{x:43,y:46},1000);local.controls(1,0,1000);local.tick(1250);assert.ok(local.vehicles[2].x<=44.600001,'catch-up cannot simulate five ride steps');const x=local.vehicles[2].x;local.tick(2000);assert.equal(local.vehicles[2].x,x,'expired input cannot resume after a stall');
});

test('delayed_own_ride_entry_updates_state_without_replaying_audio',async()=>{
 const {audio}=await import('../src/audio/service');const original=audio.play;const heard:string[]=[];audio.play=e=>{heard.push(e.kind);return true};
 try {for(const age of [-251,-250,1000,1001]){const store=new ActivityStore();await store.attach({worldId:'world',selfId:'me',status:'online',serverNow:()=>5000,async gameCommand(type:string){return type==='lobby.list'?{lobbies:[]}:{activities:[]}}});store.resync([pool()] as any);const mounted=pool();mounted.generation=2;mounted.serverTimeMs=5000-age;mounted.payload.vehicles[0].riderId='me';mounted.payload.vehicles[0].generation=2;const before=heard.length;store.receive('activity.lifecycle',mounted);assert.equal(store.snapshot().vehicles[0].riderId,'me');assert.equal(heard.length-before,age>=-250&&age<=1000?1:0);store.dispose()}}finally{audio.play=original}
});
