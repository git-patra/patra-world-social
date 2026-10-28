import {test} from 'node:test';
import assert from 'node:assert/strict';
import golden from '../../../contracts/activities.golden.json';
import {validateActivitySnapshot,activityContract} from '../src/activities/contracts';
import {validateEvent,validateCommand} from '../src/world/avatars';
test('activity snapshot goldens enforce bounded discriminated payloads',()=>{for(const x of golden.valid)assert.deepEqual(validateActivitySnapshot(x),x);for(const x of golden.invalid)assert.throws(()=>validateActivitySnapshot(x));assert.throws(()=>validateActivitySnapshot(golden.valid[0],'other'));});
test('activity resync and lifecycle require protocol 2 and exact keys',()=>{const c={v:2,commandId:'c',seq:1,worldId:'world',type:'activity.snapshot',payload:{}};assert.equal(validateCommand(c).type,'activity.snapshot');assert.throws(()=>validateCommand({...c,v:1}));assert.throws(()=>validateCommand({...c,payload:{action:'shoot'}}));const e={v:2,eventId:'e',serverTick:1,serverTimeMs:1,type:'activity.lifecycle',payload:golden.valid[0]};assert.equal(validateEvent(e).type,'activity.lifecycle');assert.throws(()=>validateEvent({...e,v:1}));});
test('safe returns stay outside every sterile area',()=>{for(const a of activityContract.areas)for(const b of activityContract.areas){const p=a.safeReturn,r=b.bounds;assert.ok(!(p.x>=r.x&&p.x<r.x+r.width&&p.y>=r.y&&p.y<r.y+r.height));}});
