import test from 'node:test';import assert from 'node:assert/strict';
import {LocalDuck,spawn,rng,rules} from '../src/features/duck/local';
import {GuestDuckSession} from '../src/features/duck/session';
import fixture from '../../../contracts/duck.golden.json';
test('shared Duck spawn and collision fixtures',()=>{const random=rng(fixture.seed);for(const expected of fixture.targets){const t=spawn(random,expected.id,expected.born);assert.deepEqual(t,expected)};for(const c of fixture.collisions){const e=new LocalDuck(4000);e.tick(5000);e.targets=fixture.overlap.map(t=>({...t}));assert.equal(e.shoot(c.aim,5000)?.targetId,c.targetId)}});
test('local guest sixty seconds, cooldown, dead target and persistent counter',()=>{const e=new LocalDuck(4000);assert.equal(e.shoot({x:.5,y:.5},3999),null);e.tick(5000);e.targets=fixture.overlap.map(t=>({...t}));assert.equal(e.shoot({x:.5,y:.5},5000)?.points,500);assert.equal(e.shoot({x:.5,y:.5},5249),null);assert.equal(e.shoot({x:.5,y:.5},5250)?.points,100);assert.equal(e.shoot({x:.5,y:.5},5500)?.points,0);e.tick(64000);assert.equal(e.snapshot().phase,'finished');assert.equal(e.snapshot().score,600);const session=new GuestDuckSession();session.start(1000);session.shoot({x:0,y:0},5000);assert.equal(session.snapshot(5100)?.shots,1);session.shoot({x:0,y:0},5300);assert.equal(session.snapshot(5300)?.shots,2);assert.equal(rules.durationMs,60000)});

import {DuckView} from '../src/features/duck/Duck';
function nodes(node:any):any[]{if(!node||typeof node!=='object')return[];const children=Array.isArray(node.props?.children)?node.props.children:[node.props?.children];return[node,...children.flatMap((v:any)=>Array.isArray(v)?v.flatMap(nodes):nodes(v))]}
test('online view aims through delivered geometry at200ms RTT and55ms after arrival',()=>{
 const p=10000,s:any={matchId:'online',worldId:'world',actor:{userId:'u'},phase:'playing',startsAt:4000,deadlineAt:64000,serverTimeMs:p,frame:180,droppedSteps:0,targets:[{id:1,kind:'fast',x:.4,y:.5,radius:.045,vx:.3,expiresAt:12000}],score:0,hits:0,shots:0};let payload:any;
 const tree=DuckView({s,aim:{x:0,y:0},now:p+155,online:true,canShoot:true,saved:false,message:'',onAim:()=>{},onShoot:a=>payload=a,onReturn:()=>{}}),all=nodes(tree),target=all.find(n=>n.props?.['data-target']===1),svg=all.find(n=>n.type==='svg');
 const [x,y]=target.props.transform.match(/[\d.]+/g).map(Number);
 svg.props.onPointerDown({preventDefault(){},currentTarget:{getBoundingClientRect:()=>({left:0,top:0,width:1000,height:1000})},clientX:x,clientY:y});
 assert.ok(Math.abs(payload.x-(.4+.3*.155))<1e-9,'visible center must track server-selected evaluation time, not old snapshot');
});

import {OnlineDuckController,presentedTargets} from '../src/features/duck/online';
import {clickPresented} from './duck-online-driver';
test('actual online controller/view follows measured50/100/200+jitter and capped320ms profiles',async()=>{
 for(const rtt of[50,100,200,320])for(const jitter of[0,12]){
  const p=10000,offset=9000,rewind=Math.min(150,rtt/2);let now=1000+rtt/2;let payload:any;const commands:string[]=[];
  const controller=new OnlineDuckController('w','m',async(type,v)=>{commands.push(type);if(type==='duck.clock')return{nonce:'single-use',sentAt:10000};if(type==='duck.clock_reply'){assert.equal(v.clientTimeMs,now);return{measured:true,calibration:{offsetMs:offset,rttMs:rtt,rewindMs:rewind,measuredAt:10000+rtt,expiresAt:25000+rtt}}}payload=v;return{targetId:0,points:0}},()=>now);
  await controller.calibrate();const s:any={matchId:'m',worldId:'w',actor:{userId:'u'},phase:'playing',startsAt:4000,deadlineAt:64000,serverTimeMs:p+1000,frame:210,targets:[{id:1,kind:'fast',x:.4,y:.5,radius:.045,vx:.3,expiresAt:p+3000}],score:0,hits:0,shots:0};
  now=s.serverTimeMs+rtt/2+jitter-offset;controller.receive(s);now+=55;
  const v=await clickPresented(controller,1);
  const serverReceipt=now+offset+rtt/2+jitter,selected=serverReceipt-rewind;
  assert.equal(v.clicked,true);assert.ok(Math.abs(payload.x-(.4+.3*(selected-s.serverTimeMs)/1000))<=.003601,'center aim stays aligned with actual server-selected time under12ms jitter');assert.equal(payload.clientTimeMs,now);assert.ok(!('rewindMs'in payload));
  await clickPresented(controller,1,'outside');assert.ok(Math.abs(payload.y-.5)>.045);assert.deepEqual(commands.slice(0,2),['duck.clock','duck.clock_reply']);
 }
});
test('online presentation hides expired/dead/stale targets and resets calibration on epoch',async()=>{
 let now=1100;const replies:any[]=[];const c=new OnlineDuckController('w','m',async(type)=>type==='duck.clock'?{nonce:'n'}:type==='duck.clock_reply'?{measured:true,calibration:{offsetMs:9000,rttMs:200,rewindMs:100,measuredAt:10200,expiresAt:25200}}:replies.shift()??{targetId:1,points:200},()=>now);
 await c.calibrate();const s:any={matchId:'m',worldId:'w',actor:{userId:'u'},phase:'playing',startsAt:4000,deadlineAt:64000,serverTimeMs:11000,frame:210,targets:[{id:1,kind:'fast',x:.4,y:.5,radius:.045,vx:.3,expiresAt:11200}],score:0,hits:0,shots:0};now=2100;c.receive(s);now=2155;assert.equal((await clickPresented(c,1)).clicked,true);c.receive(s);assert.equal((await clickPresented(c,1)).clicked,false,'acknowledged dead ID cannot revive in older snapshot');
 c.reset();c.receive(s);assert.equal(c.frame().calibrated,false);await c.calibrate();now=2300;assert.equal((await clickPresented(c,1)).clicked,false,'expired target hidden');
 c.reset();now=2100;c.receive({...s,targets:[{...s.targets[0],expiresAt:14000}]});await c.calibrate();now=2601;assert.equal(c.frame().stale,true);assert.equal((await clickPresented(c,1)).clicked,false,'prediction cannot exceed500ms');
 now=16300;assert.equal(c.frame().calibrated,false,'expired calibration not reused');c.reset();assert.equal(c.frame().snapshot,null);
 assert.equal(presentedTargets({...s,phase:'countdown'},11100).length,0);assert.equal(presentedTargets({...s,phase:'finished'},11100).length,0);
});
