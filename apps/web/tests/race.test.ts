import test from 'node:test';import assert from 'node:assert/strict';
import golden from '../../../contracts/race.golden.json';
import {LocalRace,crossGate,track,rules,prepare,stepKart,type Kart} from '../src/features/race/local';
const actors=Array.from({length:5},(_,i)=>({actorId:String(i),actorType:i===0?'human':'bot',userId:i===0?'guest':undefined,displayName:i===0?'Guest':'BOT',avatarId:'pumpkin'}));
test('shared checkpoint gates reject reverse and shortcut laps',()=>{for(const c of golden.crossings){const k={x:c.to.x,y:c.to.y,nextCheckpoint:c.next,laps:c.laps,lapTimesMs:[],lapStarted:0} as Kart;crossGate(k,c.from,1000);assert.equal(k.laps,c.wantLaps,c.name);assert.equal(k.nextCheckpoint,c.wantNext,c.name)}});
test('guest countdown, stale input expiry and full bot race match server semantics',()=>{const e=new LocalRace('m',actors,4000);e.input('0',1,{throttle:1,brake:0,steer:0},2000);e.tick(3999);assert.equal(e.snapshot().karts[0].x,18432);e.input('0',2,{throttle:1,brake:0,steer:0},4000);for(let t=4000;t<=304000;t+=10)e.tick(t);const s=e.snapshot();assert.equal(s.phase,'finished');assert.equal(s.karts.filter(k=>k.actorType==='bot'&&k.finished).length,4);assert.equal(s.karts[0].dnf,true);assert.ok(s.serverTimeMs<=304000);assert.equal(e.input('0',3,{throttle:1,brake:0,steer:0},304000),false)});
test('bounded catchup never replays expired throttle',()=>{const e=new LocalRace('m',actors.map(a=>({...a,actorType:'human'})),4000);e.input('0',1,{throttle:1,brake:0,steer:0},4000);e.tick(100000);assert.equal(e.snapshot().karts[0].speed,0);assert.ok(e.snapshot().droppedSteps>0)});
test('fixed point acceleration steering and brake shared vectors',()=>{const e=new LocalRace('m',actors,4000),k=e.karts[0];let at=4000;for(const f of golden.physics){for(let i=0;i<f.ticks;i++){at+=34;stepKart(k,f.controls,at)}assert.deepEqual({x:k.x,y:k.y,heading:k.heading,speed:k.speed},f.want)}});
test('prepared grid is stable and permutes identities',()=>{const a=prepare(actors,42);assert.deepEqual(a.map(a=>a.actorId),['1','4','3','0','2']);assert.deepEqual(actors.map(a=>a.actorId),['0','1','2','3','4'])});
import {GuestRaceSession} from '../src/features/race/session';
import React from 'react';import {renderToStaticMarkup} from 'react-dom/server';import {RaceView} from '../src/features/race/Race';
test('guest session keeps race identity on reopen and never extends absolute deadline',()=>{const session=new GuestRaceSession();session.start(1000,42,'guest-match');session.input({throttle:1,steer:0,brake:0},4000);const reopen=session.snapshot(6000)!;assert.equal(reopen.matchId,'guest-match');assert.equal(reopen.deadlineAt,304000);assert.equal(session.snapshot(304000)?.phase,'finished');session.clear();assert.equal(session.snapshot(304001),null)});
test('race controls are unavailable to spectators and saved result claims need server confirmation',()=>{const e=new LocalRace('m',actors,4000);const html=renderToStaticMarkup(React.createElement(RaceView,{snapshot:e.snapshot(),selfId:'spectator',online:true,now:1000,saved:false,disabled:false,controls:{throttle:0,brake:0,steer:0},message:'',onControl:()=>{},onReturn:()=>{}}));assert.doesNotMatch(html,/aria-label="Gas/);assert.match(html,/Menonton/);e.tick(304000);const result=renderToStaticMarkup(React.createElement(RaceView,{snapshot:e.snapshot(),selfId:'guest',online:true,now:304000,saved:false,disabled:false,controls:{throttle:0,brake:0,steer:0},message:'',onControl:()=>{},onReturn:()=>{}}));assert.match(result,/Hasil sementara/);assert.match(result,/disabled=""/)});
import {loadRules} from '../src/world/map';import ruleData from '../../../contracts/rules.v1.json';
test('race rules cannot omit a physics coefficient',()=>{const r=structuredClone(ruleData) as any;delete r.race.acceleration;assert.throws(()=>loadRules(r))});
test('finished race clock freezes at authoritative finish',()=>{const e=new LocalRace('m',actors,4000);const s={...e.snapshot(),phase:'finished',finishedAt:5000};const html=renderToStaticMarkup(React.createElement(RaceView,{snapshot:s,selfId:'guest',online:false,now:90000,saved:false,disabled:false,controls:{throttle:0,brake:0,steer:0},message:'',onControl:()=>{},onReturn:()=>{}}));assert.match(html,/>1.00s</)});

test('remounted guest controller accepts its first controls after a high prior sequence', () => {
 const session = new GuestRaceSession();
 session.start(1000, 42, 'same-race');
 // The component calls this session boundary from its mounted input timer.
 let controller: ((c: {throttle:number;brake:number;steer:number}, at:number) => boolean) | null =
  (c, at) => session.input(c, at);
 for (let i = 0; i < 600; i++) {
  assert.equal(controller({throttle:0, brake:0, steer:0}, 4000 + i * 50), true);
 }
 const before = session.snapshot(34000)!;
 const kartBefore = before.karts.find(k => k.userId === 'guest')!;
 controller = null; // effect cleanup on modal close
 const reopened = session.snapshot(35000)!;
 controller = (c, at) => session.input(c, at); // fresh controller on remount
 assert.equal(controller({throttle:1, brake:0, steer:1}, 35000), true);
 const after = session.snapshot(35067)!;
 const kartAfter = after.karts.find(k => k.userId === 'guest')!;
 assert.equal(after.matchId, before.matchId);
 assert.equal(after.deadlineAt, before.deadlineAt);
 assert.equal(kartAfter.laps, kartBefore.laps);
 assert.equal(kartAfter.nextCheckpoint, kartBefore.nextCheckpoint);
 assert.equal(reopened.karts.find(k => k.userId === 'guest')!.x, kartBefore.x);
 assert.ok(kartAfter.x > kartBefore.x);
 assert.ok(kartAfter.heading > kartBefore.heading);
 // Moving the counter into the session must not relax simulator replay checks.
 assert.equal(session.engine!.input('guest', 600, {throttle:0, brake:1, steer:-1}, 35067), false);
});
