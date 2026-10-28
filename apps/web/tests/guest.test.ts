import test from 'node:test';import assert from 'node:assert/strict';
import {retained,type LocalResult} from '../src/guest/history';
import {Completion} from '../src/guest/completion';
import {GuestSession} from '../src/guest/session';
import {guestDuck} from '../src/features/duck/session';
import {guestRace} from '../src/features/race/session';
import {guestSession} from '../src/features/penalty/session';
test('retention keeps latest 100 distinct finished device records',()=>{const rows=Array.from({length:105},(_,i)=>({id:String(i),finishedAt:i,scope:'device'} as LocalResult));const r=retained([...rows,rows[0]]);assert.equal(r.length,100);assert.equal(r[0].id,'104');assert.equal(r.at(-1)?.id,'5')});
test('completion deduplicates while pending and after persistence; retries failure',async()=>{let writes=0;const c=new Completion('one','duck',async r=>{assert.equal(r.scope,'device');writes++;if(writes===1)throw Error('storage unavailable')});c.finish(60000,'score',{});c.finish(60000,'score',{});await new Promise(r=>setImmediate(r));assert.equal(writes,1);c.finish(60000,'score',{});await new Promise(r=>setImmediate(r));c.finish(60000,'score',{});assert.equal(writes,2)});
test('mode cleanup removes all three engines and cannot resume stale monotonic clocks',()=>{GuestSession.start('race',{at:0});GuestSession.start('penalty',{at:0});GuestSession.start('duck',{at:0});assert.ok(guestRace.snapshot(100));assert.ok(guestSession.snapshot(100));assert.ok(guestDuck.snapshot(100));GuestSession.clear();assert.equal(guestRace.snapshot(900000),null);assert.equal(guestSession.snapshot(900000),null);assert.equal(guestDuck.snapshot(900000),null)});

import {IDBFactory,IDBObjectStore} from 'fake-indexeddb';
import {GuestHistory,defaults} from '../src/guest/history';
const record=(i:number):LocalResult=>({id:`session-${i}`,gameType:'duck',finishedAt:i,durationMs:60000,scope:'device',version:{mapId:'fixture',mapVersion:1,rulesId:'duck-v1',rulesVersion:1,protocolVersion:1},summary:`Skor ${i}`,detail:{score:i,hits:1,shots:2}});
test('actual IndexedDB transactions retain newest 100 under concurrent append and reject duplicate replacement',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('retention',()=>factory);
 await Promise.all(Array.from({length:120},(_,i)=>h.append(record(i))));
 let rows=await h.list();assert.equal(rows.length,100);assert.equal(rows[0].id,'session-119');assert.equal(rows.at(-1)?.id,'session-20');
 await h.append({...record(119),finishedAt:9999,summary:'Forged replacement'});
 rows=await h.list();assert.equal(rows[0].finishedAt,119);assert.equal(rows[0].summary,'Skor 119');assert.equal(rows.length,100);
 const reopened=new GuestHistory('retention',()=>factory);assert.deepEqual(await reopened.list(),rows);
});
test('actual IndexedDB settings and results survive fresh instance; atomic reset removes both',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('settings',()=>factory),p={...defaults,displayName:'Local fixture',avatar:'moon' as const,audio:false,controls:'touch' as const,reducedMotion:true};
 await h.append(record(1));await h.saveSettings(p);
 const reopened=new GuestHistory('settings',()=>factory);assert.deepEqual(await reopened.settings(),p);assert.equal((await reopened.list()).length,1);
 await reopened.reset();const again=new GuestHistory('settings',()=>factory);assert.deepEqual(await again.list(),[]);assert.deepEqual(await again.settings(),defaults);
});
test('reset generation fences append waiting for IndexedDB open',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('pending',()=>factory);
 const append=h.append(record(1));const reset=h.reset();await Promise.all([append,reset]);assert.deepEqual(await h.list(),[]);assert.deepEqual(await h.settings(),defaults);
 await h.append(record(2));assert.equal((await h.list())[0].id,'session-2');
});
test('failed IndexedDB writes reject, preserve existing records/settings, and permit retry',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('error',()=>factory);await h.append(record(1));await h.saveSettings({...defaults,displayName:'Keep me'});
 await assert.rejects(h.append({...record(2),detail:()=>{}}));assert.deepEqual((await h.list()).map(r=>r.id),['session-1']);assert.equal((await h.settings()).displayName,'Keep me');
 await h.append(record(2));assert.deepEqual((await h.list()).map(r=>r.id),['session-2','session-1']);
 const denied=new GuestHistory('denied',()=>{throw Error('Storage denied')});await assert.rejects(denied.list(),/Storage denied/);await assert.rejects(denied.reset(),/Storage denied/);
});

test('reset failure aborts both-store transaction without partially deleting history',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('reset-abort',()=>factory);await h.append(record(1));await h.saveSettings({...defaults,displayName:'Keep on rollback'});
 const clear=IDBObjectStore.prototype.clear;
 IDBObjectStore.prototype.clear=function(){if(this.name==='settings')throw Error('Injected second-store failure');return clear.call(this)};
 try{await assert.rejects(h.reset(),/Injected second-store failure/)}finally{IDBObjectStore.prototype.clear=clear}
 assert.equal((await h.list()).length,1);assert.equal((await h.settings()).displayName,'Keep on rollback');await h.reset();assert.deepEqual(await h.list(),[]);assert.deepEqual(await h.settings(),defaults);
});

test('device Motion transaction preserves concurrent settings and reset fences pending mutation',async()=>{
 const factory=new IDBFactory(),h=new GuestHistory('motion',()=>factory),p={...defaults,displayName:'Concurrent profile',avatar:'moon' as const,audio:false,controls:'touch' as const};
 await Promise.all([h.saveSettings(p),h.setMotion(true)]);assert.deepEqual(await h.settings(),{...p,reducedMotion:true});
 const motion=h.setMotion(false),reset=h.reset();await Promise.all([motion,reset]);assert.deepEqual(await h.settings(),defaults);
 const save=h.saveSettings(p),resetAgain=h.reset();await Promise.all([save,resetAgain]);assert.deepEqual(await h.settings(),defaults);
});
