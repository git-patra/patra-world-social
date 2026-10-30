import test from 'node:test';
import assert from 'node:assert/strict';
import {ActivityStore} from '../src/activities/store';
import {validateActivitySnapshot} from '../src/activities/contracts';
import {validateCommand} from '../src/world/avatars';
import {audio} from '../src/audio/service';
import {activityInput} from '../src/activities/input-router';
import {clawActors} from '../src/world/scenes/claw';
import {clawSpawn} from '../src/activities/claw';
import type {Claw} from '../src/activities/contracts';
import protocolSchema from '../../../contracts/protocol.schema.json';
const claw=(generation=1,tick=1)=>({worldId:'world',areaId:'claw-1',incarnation:'runtime',generation,serverTimeMs:1000,tick,kind:'claw',payload:{claw:{operatorId:'',operatorName:'',leaseUntil:0,x:.5,y:.5,phase:'idle',phaseAt:0,dropId:0,prizeId:''}}});
const socket=()=>({worldId:'world',selfId:'me',status:'online',serverNow:()=>1000,async gameCommand(type:string,_payload?:any):Promise<any>{return type==='lobby.list'?{lobbies:[]}:{activities:[]}}});
test('claw_snapshot_and_intent_are_exact_bounded_contracts',()=>{assert.equal(validateActivitySnapshot(claw()).kind,'claw');const c={v:2,worldId:'world',seq:1,commandId:'enter',type:'activity.claw',payload:{incarnation:'runtime',areaGeneration:0,generation:1,machineId:'claw-1',action:'claw.enter'}};assert.equal(validateCommand(c).type,'activity.claw');for(const extra of [{x:.28},{prizeId:'bear'},{dx:1}])assert.throws(()=>validateCommand({...c,payload:{...c.payload,...extra}}));for(const extra of [{x:2},{prizeId:'invented'},{score:1},{operatorId:'me'}])assert.throws(()=>validateActivitySnapshot({...claw(),payload:{claw:{...claw().payload.claw,...extra}}}));});
test('guest_claw_lease_drop_animation_and_match_cleanup_use_application_clock',async()=>{const store=new ActivityStore();assert.ok((store.snapshot() as any).claw,'guest cabinet missing');assert.equal((store as any).interactClaw({x:0,y:0}),false);assert.equal((store as any).interactClaw({x:41,y:24}),true);assert.equal(store.snapshot().inputMode,'claw');assert.equal((store as any).interactFootball({x:28,y:46}),false);for(let n=0;n<6;n++){(store as any).clawControls({x:-1,y:-1});store.tick(50*(n+1))};assert.equal((store as any).dropClaw(),true);const s=(store.snapshot() as any).claw;assert.equal(s.prizeId,'bear');assert.equal(s.dropId,1);assert.equal((store as any).dropClaw(),false);store.tick(800);assert.equal((store.snapshot() as any).claw.phase,'lifting');store.tick(1300);assert.equal((store.snapshot() as any).claw.phase,'releasing');store.tick(1800);assert.equal((store.snapshot() as any).claw.prizeId,'');store.tick(20000);assert.equal(store.snapshot().inputMode,'walk');(store as any).interactClaw({x:41,y:24});await store.startLocal('pingpong',21000,1,'test-match');assert.equal((store.snapshot() as any).claw.operatorId,'');store.dispose()});
test('claw_spectator_fences_resync_audio_and_operator_input',async()=>{const store=new ActivityStore(),transport=socket(),sent:any[]=[];transport.gameCommand=async(type,p)=>{sent.push([type,p]);return type==='lobby.list'?{lobbies:[]}:{activities:[]}};await store.attach(transport);store.resync([claw()] as any);const occupied=claw(2);Object.assign(occupied.payload.claw,{operatorId:'peer',operatorName:'Peer',leaseUntil:21000,phase:'aiming'});store.receive('activity.lifecycle',occupied);store.setPosition({x:41,y:24});assert.equal(store.snapshot().inputMode,'walk');assert.equal((store as any).interactClaw(),false);assert.equal((store as any).dropClaw(),false);assert.ok(!sent.some(([t])=>t==='activity.claw'));const sounds:any[]=[],play=audio.play;audio.play=e=>{sounds.push(e);return true};try{const drop=claw(3);Object.assign(drop.payload.claw,{...occupied.payload.claw,phase:'dropping',phaseAt:1000,dropId:1,prizeId:'bear',x:.28,y:.28});store.receive('activity.lifecycle',drop);store.receive('activity.lifecycle',drop);assert.equal(sounds.length,1);store.resync([{...drop,generation:4}] as any);assert.equal(sounds.length,1);store.receive('activity.lifecycle',{...drop,generation:5,serverTimeMs:-1,payload:{claw:{...drop.payload.claw,dropId:2}}});assert.equal(sounds.length,1);store.resync([{...claw(),incarnation:'next'}] as any);store.receive('activity.lifecycle',{...drop,generation:100});store.resync([{...drop,generation:101}] as any);assert.equal((store.snapshot() as any).claw.operatorId,'')}finally{audio.play=play;store.dispose()}});
test('claw_modal_release_sends_neutral_and_guest_repeated_cycles_stay_bounded',()=>{const store=new ActivityStore();(store as any).interactClaw({x:41,y:24});const off=activityInput.register('claw',input=>(store as any).clawControls(input));activityInput.setOwner('claw');activityInput.direction(1,1);activityInput.block('claw-test',true);store.tick(50);assert.equal((store.snapshot() as any).claw.x,.5);activityInput.block('claw-test',false);activityInput.setOwner('walk');off();for(let n=0;n<20;n++){(store as any).interactClaw({x:41,y:24});(store as any).interactClaw({x:41,y:24})};assert.equal(store.snapshot().games.length,0);assert.equal(store.snapshot().lobbies.length,0);store.dispose()});

function drawClaw(b:Claw,at=0){
 const lines:number[][]=[],plush:number[][]=[];
 const ctx=new Proxy({lineTo:(x:number,y:number)=>lines.push([x,y]),arc:(x:number,y:number,r:number)=>{if(r===5)plush.push([x,y])}},{get:(o,k)=>k in o?o[k as keyof typeof o]:()=>{},set:()=>true});
 clawActors(b,at)[0].draw(ctx as unknown as CanvasRenderingContext2D,at);
 return {grip:lines[0],tips:[lines[1],lines[3]],plush};
}
test('claw_renderer_aims_and_picks_up_at_the_visible_preset_in_both_axes',()=>{
 for(const [id,x,y] of [['bear',.28,.28],['bunny',.72,.30],['fox',.50,.75]] as const){
  const b:Claw={...clawSpawn(),phase:'aiming',x,y},aim=drawClaw(b),preset=aim.plush[['bear','bunny','fox'].indexOf(id)];
  assert.equal(aim.grip[0],preset[0],`${id}: horizontal aim`);
  assert.equal(aim.tips[0][1],preset[1],`${id}: vertical aim`);
  const dropping={...b,phase:'dropping' as const,prizeId:id},bottom=drawClaw(dropping,500),pickup=drawClaw({...dropping,phase:'lifting'},500),raised=drawClaw({...dropping,phase:'lifting'},1000);
  assert.deepEqual(bottom.grip,preset,`${id}: drop reaches visible preset`);
  assert.deepEqual(pickup.plush.at(-1),preset,`${id}: pickup starts without jumping`);
  assert.ok(raised.plush.at(-1)![1]<preset[1],`${id}: picked plush lifts`);
  assert.deepEqual(drawClaw({...dropping,phase:'releasing'},1000).plush.at(-1),raised.plush.at(-1),`${id}: release starts without jumping`);
 }
});
test('claw_renderer_keeps_vertical_aim_visible_through_every_drop_phase',()=>{
 for(const [phase,at] of [['aiming',0],['dropping',250],['lifting',750],['releasing',1250]] as const){
  const b:Claw={...clawSpawn(),phase,y:0};
  assert.ok(drawClaw({...b,y:1},at).grip[1]>drawClaw(b,at).grip[1],`${phase}: Y changes grip position`);
 }
});

// Evaluate only the actual claw payload branch; fail on new keywords rather than silently ignoring them.
function matchesClawSchema(s:any,value:any):boolean{
 const supported=['type','const','enum','minimum','maximum','minLength','maxLength','properties','required','additionalProperties','not','allOf','anyOf','if','then','else'];
 for(const key of Object.keys(s))assert.ok(supported.includes(key),`unsupported claw schema keyword: ${key}`);
 if(s.type&&!(s.type==='object'?value!==null&&typeof value==='object'&&!Array.isArray(value):s.type==='integer'?Number.isInteger(value):typeof value===s.type))return false;
 if('const'in s&&value!==s.const||s.enum&&!s.enum.includes(value))return false;
 if(typeof value==='number'&&(s.minimum!==undefined&&value<s.minimum||s.maximum!==undefined&&value>s.maximum))return false;
 if(typeof value==='string'&&(s.minLength!==undefined&&value.length<s.minLength||s.maxLength!==undefined&&value.length>s.maxLength))return false;
 if(value!==null&&typeof value==='object'&&!Array.isArray(value)){
  if(s.required?.some((key:string)=>!Object.hasOwn(value,key)))return false;
  if(s.additionalProperties===false&&Object.keys(value).some(key=>!Object.hasOwn(s.properties??{},key)))return false;
  for(const [key,child] of Object.entries(s.properties??{}))if(Object.hasOwn(value,key)&&!matchesClawSchema(child,value[key]))return false;
 }
 if(s.not&&matchesClawSchema(s.not,value))return false;
 if(s.allOf&&!s.allOf.every((child:any)=>matchesClawSchema(child,value)))return false;
 if(s.anyOf&&!s.anyOf.some((child:any)=>matchesClawSchema(child,value)))return false;
 if(s.if){const branch=matchesClawSchema(s.if,value)?s.then:s.else;if(branch&&!matchesClawSchema(branch,value))return false}
 return true;
}
const clawPayloadSchema=(protocolSchema.allOf as any[]).find(s=>s.if?.properties?.type?.const==='activity.claw').then.properties.payload;
const clawPayload={incarnation:'runtime',areaGeneration:0,generation:1,machineId:'claw-1'};
for(const [name,command] of [
 ['enter',{action:'claw.enter'}],['leave',{action:'claw.leave'}],['drop',{action:'claw.drop'}],
 ['directional move',{action:'claw.move',dx:-1,dy:1}],['neutral move',{action:'claw.move',dx:0,dy:0}],
] as const)test(`claw_json_schema_accepts_${name.replaceAll(' ','_')}`,()=>assert.equal(matchesClawSchema(clawPayloadSchema,{...clawPayload,...command}),true));
test('claw_json_schema_rejects_extra_missing_and_unbounded_fields',()=>{
 for(const fields of [
  {action:'claw.enter',dx:1},{action:'claw.leave',dy:0},{action:'claw.drop',dx:0,dy:0},
  {action:'claw.drop',x:.28},{action:'claw.drop',prizeId:'bear'},{action:'claw.move',dx:0,dy:0,score:1},
  {action:'claw.move',dx:0},{action:'claw.move',dy:0},{action:'claw.move',dx:2,dy:0},{action:'claw.move',dx:0,dy:-2},
  {action:'invented'},{action:'claw.drop',machineId:'claw-2'},
 ])assert.equal(matchesClawSchema(clawPayloadSchema,{...clawPayload,...fields}),false,JSON.stringify(fields));
 for(const key of Object.keys(clawPayload)){
  const payload:Record<string,unknown>={...clawPayload,action:'claw.drop'};delete payload[key];
  assert.equal(matchesClawSchema(clawPayloadSchema,payload),false,`missing ${key}`);
 }
});
