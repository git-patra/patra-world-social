import test from 'node:test';
import assert from 'node:assert/strict';
import { ActivityStore } from '../src/activities/store';
import { validateActivitySnapshot } from '../src/activities/contracts';
import { validateCommand } from '../src/world/avatars';
import { audio } from '../src/audio/service';

const ball = (generation = 1, tick = 20) => ({
    worldId: 'world', areaId: 'penalty', incarnation: 'runtime', generation,
    serverTimeMs: 1000, tick, kind: 'football',
    payload: { ball: { x: 28, y: 46, vx: 0, vy: 0, visible: true } }
});
const area = (areaId = 'penalty', generation = 0, phase = 'OPEN') => ({
    ...ball(generation), areaId, kind: 'area',
    payload: { phase, matchId: phase === 'OPEN' ? '' : 'match', startsAt: 0, clearAt: 4000, resetAt: 0 }
});
const socket = () => ({
    worldId: 'world', selfId: 'me', status: 'online', serverNow: () => 1000,
    async gameCommand(type: string, _payload?: any): Promise<any> {
        return type === 'lobby.list' ? { lobbies: [] } : { activities: [] };
    }
});

test('football_snapshot_and_kick_are_exact_bounded_contracts', () => {
    assert.equal(validateActivitySnapshot(ball()).kind, 'football');
    for (const b of [{ ...ball().payload.ball, score: 1 }, { ...ball().payload.ball, vx: 9 }, { ...ball().payload.ball, x: 0 }])
        assert.throws(() => validateActivitySnapshot({ ...ball(), payload: { ball: b } }));
    const command = { v: 2, commandId: 'kick', seq: 1, worldId: 'world', type: 'activity.kick', payload: { incarnation: 'runtime', areaGeneration: 0, generation: 1, action: 'kick', dx: 0, dy: -1 } };
    assert.equal(validateCommand(command).type, 'activity.kick');
    for (const extra of [{ x: 28 }, { score: 2 }, { vehicleId: 'car-2' }])
        assert.throws(() => validateCommand({ ...command, payload: { ...command.payload, ...extra } }));
    assert.throws(() => validateCommand({ ...command, payload: { ...command.payload, dy: 0 } }));
});

test('guest_ball_kick_proximity_cooldown_friction_and_penalty_start_are_local', async () => {
    const store = new ActivityStore();
    assert.ok((store.snapshot() as any).football, 'guest world is missing its one football');
    assert.equal((store as any).interactFootball({ x: 5, y: 5 }), false);
    assert.equal((store as any).interactFootball({ x: 28, y: 46 }), true);
    assert.equal((store as any).interactFootball({ x: 28, y: 46 }), false);
    store.tick(50);
    const moving = (store.snapshot() as any).football;
    assert.ok(moving.y < 46);
    assert.ok(Math.hypot(moving.vx, moving.vy) < 8);
    assert.equal(store.snapshot().games.length, 0);
    const pending = store.startLocal('penalty', 1000, undefined, 'local-penalty');
    assert.equal((store.snapshot() as any).football.visible, false, 'hide before lazy penalty startup');
    assert.equal((store as any).interactFootball({ x: 28, y: 46 }), false);
    await pending;
    store.clearLocal();
    assert.deepEqual((store.snapshot() as any).football, { x: 28, y: 46, vx: 0, vy: 0, visible: true });
    store.dispose();
});

test('football_revisions_preserve_masks_and_reject_stale_runtime_resync', async () => {
    const store = new ActivityStore();
    await store.attach(socket());
    store.resync([area(), area('race'), ball()] as any);
    const kicked = ball(2);
    kicked.payload.ball.vy = -8;
    store.receive('activity.lifecycle', kicked);
    assert.equal(store.snapshot().areas.length, 2);
    assert.equal((store.snapshot() as any).football.vy, -8);
    store.receive('activity.snapshot', { ...ball(), tick: 1000 });
    assert.equal((store.snapshot() as any).football.vy, -8);
    store.receive('activity.lifecycle', area('penalty', 1, 'CLEARING'));
    assert.equal((store.snapshot() as any).nearbyFootball, false);
    assert.equal((store as any).interactFootball({ x: 28, y: 46 }), false);
    store.resync([{ ...ball(), incarnation: 'next-runtime' }] as any);
    store.receive('activity.lifecycle', { ...kicked, generation: 100 });
    store.resync([{ ...kicked, generation: 101 }] as any);
    assert.equal((store.snapshot() as any).football.vy, 0);
    store.dispose();
});

test('own_ride_entry_sound_ignores_second_rider_and_resync', async () => {
    const { vehiclePool } = await import('../src/activities/vehicles');
    const store = new ActivityStore();
    await store.attach(socket());
    const pool = (generation: number) => ({ ...ball(generation), areaId: 'race', kind: 'vehicles', payload: { vehicles: vehiclePool() } });
    const sounds: any[] = [];
    const play = audio.play;
    audio.play = event => { sounds.push(event); return true; };
    try {
        store.resync([area('race'), pool(1)] as any);
        const own = pool(2);
        own.payload.vehicles[0].riderId = 'me';
        own.payload.vehicles[0].generation = 2;
        store.receive('activity.lifecycle', own);
        assert.equal(sounds.length, 1);
        const peer = structuredClone(own);
        peer.generation = 3;
        peer.payload.vehicles[1].riderId = 'peer';
        peer.payload.vehicles[1].generation = 2;
        store.receive('activity.lifecycle', structuredClone(peer));
        assert.equal(sounds.length, 1, 'another rider repeats the self entry cue');
        peer.generation = 4;
        peer.payload.vehicles[1].riderId = '';
        peer.payload.vehicles[1].generation = 3;
        store.receive('activity.lifecycle', structuredClone(peer));
        store.resync([{ ...peer, generation: 5 }] as any);
        store.receive('activity.lifecycle', own);
        assert.equal(sounds.length, 1);
    } finally {
        audio.play = play;
        store.dispose();
    }
});

test('casual_scene_has_one_ball_and_hides_it_before_penalty_projection', async () => {
    const { ActivityScene } = await import('../src/activities/scene');
    const store = new ActivityStore();
    let changed!: () => void;
    let ready = new Promise<void>(r => { changed = r; });
    const scene = new ActivityScene(() => changed());
    scene.set(store.snapshot());
    await ready;
    await Promise.resolve();
    assert.equal(scene.actors(0).filter(a => a.id === 'football:ball').length, 1);
    const pending = store.startLocal('penalty', 1000, undefined, 'one-match-ball');
    scene.set(store.snapshot());
    assert.equal(scene.actors(0).filter(a => a.id === 'football:ball').length, 0);
    await pending;
    ready = new Promise<void>(r => { changed = r; });
    scene.set(store.snapshot());
    await ready;
    assert.equal(scene.actors(1000).filter(a => a.id.endsWith(':ball')).length, 1);
    store.dispose();
});

test('late_spectator_reconnect_before_reveal_stays_passive_and_can_walk_away', async () => {
    const store = new ActivityStore();
    const transport = socket(), sent: string[] = [];
    transport.gameCommand = async type => {
        sent.push(type);
        if (type === 'lobby.list')
            return { lobbies: [{ id: 'lobby', arenaId: 'penalty', gameType: 'penalty', worldId: 'world', state: 'playing', matchId: 'match', players: [{ userId: 'a', actorId: 'a' }, { userId: 'b', actorId: 'b' }], spectators: ['me'] }] };
        if (type === 'penalty.snapshot')
            return { matchId: 'match', phase: 'choosing', turn: 0, turnId: 'turn', choiceDeadlineAt: 3000, nextTurnAt: 4500, kickerId: 'a', keeperId: 'b', scores: [0, 0], chosen: [true, false], actors: [{ userId: 'a', actorId: 'a', displayName: 'A' }, { userId: 'b', actorId: 'b', displayName: 'B' }], turns: [] };
        return { activities: [] };
    };
    await store.attach(transport);
    store.setPosition({ x: 23, y: 46 });
    assert.equal(store.snapshot().active, null);
    assert.equal((store.snapshot() as any).spectating?.kind, 'penalty');
    assert.equal(store.snapshot().inputMode, 'walk');
    await store.choose(2);
    assert.ok(!sent.includes('penalty.choose'));
    store.setPosition({ x: 22, y: 46 });
    assert.equal((store.snapshot() as any).spectating, null);
    await store.attach(transport);
    store.setPosition({ x: 23, y: 46 });
    const snapshot = (store.snapshot() as any).spectating.snapshot;
    assert.equal(snapshot.chosen[0], true);
    assert.equal(snapshot.turns.length, 0);
    assert.equal('kickZone' in snapshot, false);
    store.dispose();
});

test('socket_full_resync_requires_four_masks_vehicle_pool_football_and_claw', async () => {
    const { WorldSocket } = await import('../src/net/socket');
    const { vehiclePool } = await import('../src/activities/vehicles');
    const connection = new WorldSocket('world', () => {}, () => {}, 'http://village.test');
    const masks = ['race', 'penalty', 'pingpong', 'blaster'].map(id => area(id));
    const vehicles = { ...ball(), areaId: 'race', kind: 'vehicles', payload: { vehicles: vehiclePool() } };
    const claw={...ball(),areaId:'claw-1',kind:'claw',payload:{claw:{operatorId:'',operatorName:'',leaseUntil:0,x:.5,y:.5,phase:'idle',phaseAt:0,dropId:0,prizeId:''}}};
    let values: any[] = [...masks, vehicles, ball(),claw];
    connection.gameCommand = async () => ({ activities: values });
    assert.equal((await connection.activitySnapshots()).length, 7);
    values = [...masks, vehicles,ball()];
    await assert.rejects(connection.activitySnapshots(), /Invalid activity resync/);
    values = [...masks, vehicles, vehicles,claw];
    await assert.rejects(connection.activitySnapshots(), /Invalid activity resync/);
    values = [...masks.slice(1), masks[1], vehicles, ball(),claw];
    await assert.rejects(connection.activitySnapshots(), /Invalid activity resync/);
});

test('guest_football_bounds_idle_and_catchup_follow_the_owner_caps', async () => {
    const { LocalFootball } = await import('../src/activities/football');
    const { footballInside } = await import('../src/activities/contracts');
    const local = new LocalFootball();
    assert.equal(local.kick({ x: 28, y: 46 }, 1, 0, 0), true);
    local.tick(250);
    assert.ok(local.ball.x <= 29.3757, 'a stall advances at most four decaying steps');
    let speed = Math.hypot(local.ball.vx, local.ball.vy);
    for (let at = 300; at <= 5000; at += 50) {
        local.tick(at);
        const next = Math.hypot(local.ball.vx, local.ball.vy);
        assert.ok(next <= speed);
        assert.ok(footballInside(local.ball.x, local.ball.y));
        speed = next;
    }
    local.tick(16000);
    assert.deepEqual(local.ball, { x: 28, y: 46, vx: 0, vy: 0, visible: true });
    local.ball.x = 100;
    local.tick(16050);
    assert.deepEqual(local.ball, { x: 28, y: 46, vx: 0, vy: 0, visible: true });
});

import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { ActivityHUD } from '../src/activities/ActivityHUD';

test('football_HUD_exposes_discrete_kick_and_spectator_HUD_has_no_choice_buttons', async () => {
    const store = new ActivityStore();
    store.setPosition({ x: 28, y: 46 });
    const html = () => renderToStaticMarkup(React.createElement(ActivityHUD, { store }));
    assert.match(html(), /Kick · E/);
    assert.doesNotMatch(html(), /Zona [1-4]/);
    const transport = socket();
    transport.gameCommand = async type => type === 'lobby.list' ? { lobbies: [{ id: 'lobby', arenaId: 'penalty', gameType: 'penalty', worldId: 'world', state: 'playing', matchId: 'match', players: [], spectators: ['me'] }] } : type === 'penalty.snapshot' ? { matchId: 'match', phase: 'choosing', choiceDeadlineAt: 3000, actors: [], chosen: [], turns: [], scores: [1, 2] } : { activities: [] };
    await store.attach(transport);
    store.setPosition({ x: 23, y: 46 });
    assert.match(html(), /Penonton penalti/);
    assert.match(html(), /1 : 2/);
    assert.doesNotMatch(html(), /<button|Zona [1-4]/);
    store.dispose();
});

test('football_audio_plays_only_fresh_live_kicks_and_never_resync_or_stale_replay', async () => {
    const store = new ActivityStore(), transport = socket();
    transport.serverNow = () => 3000;
    await store.attach(transport);
    const sounds: string[] = [], play = audio.play;
    audio.play = event => { sounds.push(event.eventId); return true; };
    try {
        store.resync([area(), ball()] as any);
        const kick = { ...ball(2), serverTimeMs: 3000, payload: { ball: { ...ball().payload.ball, vy: -8 } } };
        store.receive('activity.lifecycle', kick);
        store.receive('activity.lifecycle', kick);
        assert.equal(sounds.length, 1);
        store.resync([{ ...kick, generation: 3 }] as any);
        assert.equal(sounds.length, 1);
        store.receive('activity.lifecycle', { ...kick, generation: 4, serverTimeMs: 1000 });
        assert.equal(sounds.length, 1, 'a delayed kick cue should be dropped');
    } finally {
        audio.play = play;
        store.dispose();
    }
});
