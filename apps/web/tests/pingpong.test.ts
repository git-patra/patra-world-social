import test from 'node:test';
import assert from 'node:assert/strict';
import { LocalPingpong, won } from '../src/features/pingpong/local';
import { ActivityStore } from '../src/activities/store';
import { InputRouter } from '../src/activities/input-router';
import { Completion } from '../src/guest/completion';
import { validateCommand } from '../src/world/avatars';
import golden from '../../../contracts/pingpong.golden.json';
const actors = [{ actorId: 'a', userId: 'u', actorType: 'human', displayName: 'A', avatarId: 'pumpkin' }, { actorId: 'b', userId: 'v', actorType: 'human', displayName: 'B', avatarId: 'pumpkin' }];
const fixture = () => new LocalPingpong('m', actors, 1000);
test('pingpong Go/TS shared serve, movement, miss and paddle bounce goldens', () => { for (const c of golden.cases) {
    const e = fixture();
    assert.equal(e.serve('u', 1000), true);
    e.input('u', 1, c.axis, 1000);
    e.input('v', 1, 'peerAxis' in c ? c.peerAxis : 0, 1000);
    for (let i = 1; i <= c.steps; i++)
        e.tick(1000 + i * 50);
    const s = e.snapshot();
    assert.equal(s.phase, c.phase, c.name);
    assert.deepEqual(s.scores, c.scores);
    const values = [...s.paddles, ...Object.values(s.ball)], expected = [...c.paddles, ...Object.values(c.ball)];
    values.forEach((v, i) => assert.ok(Math.abs(v - expected[i]) < 1e-9, `${c.name} ${i}: ${v} vs ${expected[i]}`));
} });
test('pingpong actor authority, neutral grace, bounded catchup and immutable deadline', () => { const e = fixture(); assert.equal(e.input('spectator', 1, 1, 1000), false); assert.equal(e.serve('v', 1000), false); e.serve('u', 1000); for (let i = 1; i < 100; i++)
    e.input('u', i, 1, 1000); e.tick(1050); assert.equal(e.snapshot().paddles[0], .55); e.presence('u', 'disconnect', 2000); e.tick(16999); assert.equal(e.snapshot().result, undefined); e.tick(17000); assert.equal(e.snapshot().result?.participants[0].outcome, 'forfeit'); e.presence('u', 'resume', 18000); assert.equal(e.snapshot().result?.finishedAt, 17000); const deadline = fixture(); deadline.serve('u', 1000); deadline.tick(121000); const s = deadline.snapshot(); assert.equal(s.result?.finishedAt, 121000); assert.equal(s.result?.pingpong.reason, 'deadline'); assert.ok(s.ball.x <= .63, 'catchup capped at 4 steps'); assert.equal(won(7, 6), false); assert.equal(won(7, 5), true); assert.equal(won(11, 10), true); });
test('pingpong strict intents reject positions, score and stale serve fields', () => { const base = { v: 2, worldId: 'w', commandId: 'c', seq: 1, type: 'pingpong.input', payload: { matchId: 'm', axis: 1 } }; assert.equal(validateCommand(base).type, 'pingpong.input'); for (const extra of [{ score: 7 }, { x: .5 }, { axis: 2 }])
    assert.throws(() => validateCommand({ ...base, payload: { ...base.payload, ...extra } })); assert.throws(() => validateCommand({ ...base, type: 'pingpong.serve', payload: { matchId: 'm' } })); });
test('pingpong guest engine survives view subscriptions and controls release with independent pointers', async () => { const store = new ActivityStore(); await store.startLocal('pingpong', 1000, 1, 'test'); const unsubscribe = store.subscribe(() => { }); unsubscribe(); await store.serve(); store.tick(1050); assert.equal(store.snapshot().active?.snapshot.phase, 'rally'); const router = new InputRouter(); router.register('pingpong', input => store.paddle(input.steer)); router.setOwner('pingpong'); router.pointerControl(1, 'steer', -1); router.pointerControl(2, 'steer', 1); router.releasePointer(2); store.tick(1100); const p = (store.snapshot().active!.snapshot as ReturnType<LocalPingpong['snapshot']>).paddles[0]; assert.equal(p, .45); router.block('modal', true); store.tick(1150); assert.equal((store.snapshot().active!.snapshot as ReturnType<LocalPingpong['snapshot']>).paddles[0], p); store.clearLocal(); assert.equal(store.snapshot().inputMode, 'walk'); store.dispose(); });
test('pingpong completion retries storage and preserves rules identity without duplicate rows', async () => { let calls = 0, record: any; const completion = new Completion('receipt', 'pingpong', async (r) => { calls++; if (calls === 1)
    throw Error('offline'); record = r; }); completion.finish(120000, '0 : 0', {}); await new Promise(r => setTimeout(r, 0)); completion.finish(120000, '0 : 0', {}); await new Promise(r => setTimeout(r, 0)); completion.finish(120000, '0 : 0', {}); assert.equal(calls, 2); assert.equal(record.version.rulesId, 'pingpong-v1'); });
test('pingpong stale snapshots cannot rewind results and passive spectators keep walking', async () => { const engine = fixture(), lobby = { id: 'l', arenaId: 'pingpong', gameType: 'pingpong', worldId: 'w', state: 'running', matchId: 'm', players: actors.map(a => ({ userId: a.userId, actorId: a.actorId })), spectators: [] }; const socket = { worldId: 'w', selfId: 'watcher', status: 'online', serverNow: () => 1000, async gameCommand(kind: string): Promise<any> { return kind === 'lobby.list' ? { worldId: 'w', lobbies: [lobby] } : kind === 'pingpong.snapshot' ? engine.snapshot() : { activities: [] }; } }; const store = new ActivityStore(); await store.attach(socket); store.setPosition({ x: 42, y: 22 }); assert.equal(store.snapshot().spectating?.kind, 'pingpong'); assert.equal(store.snapshot().inputMode, 'walk'); const old = engine.snapshot(); engine.tick(121000); store.receive('pingpong.snapshot', engine.snapshot()); store.receive('pingpong.snapshot', old); assert.equal(store.snapshot().games[0].snapshot.phase, 'results'); store.dispose(); });

test('pingpong sounds only fresh live paddle/score transitions and spectator HUD has no controls', async () => {
    const {audio}=await import('../src/audio/service');
    const {PingpongHUD}=await import('../src/features/pingpong/PingpongHUD');
    const React=await import('react');
    const {renderToStaticMarkup}=await import('react-dom/server');
    const engine=fixture();engine.serve('u',1000);engine.input('v',1,1,1000);
    let now=1000;
    const lobby={id:'l',arenaId:'pingpong',gameType:'pingpong',worldId:'w',state:'running',matchId:'m',players:actors.map(a=>({userId:a.userId,actorId:a.actorId})),spectators:[]};
    const socket={worldId:'w',selfId:'watcher',status:'online',serverNow:()=>now,async gameCommand(kind:string):Promise<any>{return kind==='lobby.list'?{worldId:'w',lobbies:[lobby]}:kind==='pingpong.snapshot'?engine.snapshot():{activities:[]}}};
    const calls:any[]=[],original=audio.play;audio.play=(event)=>{calls.push(event);return true};
    const store=new ActivityStore();
    try {
        await store.attach(socket);assert.equal(calls.length,0,'resync has no cue');
        for(let i=1;i<=15;i++)engine.tick(1000+i*50);
        now=1750;store.receive('pingpong.snapshot',engine.snapshot());assert.equal(calls.length,1);assert.equal(calls[0].kind,'paddle');
        store.receive('pingpong.snapshot',engine.snapshot());assert.equal(calls.length,1,'duplicate cue');
        const delayed={...engine.snapshot(),frame:20,ball:{...engine.snapshot().ball,vx:.6}};now=5000;store.receive('pingpong.snapshot',delayed);assert.equal(calls.length,1,'delayed cue');
        const html=renderToStaticMarkup(React.createElement(PingpongHUD,{store,snapshot:engine.snapshot(),spectator:true}));assert.match(html,/Penonton pingpong/);assert.doesNotMatch(html,/<button/);
    } finally {audio.play=original;store.dispose()}
});
