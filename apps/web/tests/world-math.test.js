import test from 'node:test';
import assert from 'node:assert/strict';
import { project, unproject, findPath, routeToStation, tileKey, isPlayable } from '../src/world-math.js';

test('isometric projection round trips fractional coordinates', () => {
  for (const [x, y] of [[0, 0], [8, 8], [15, 1], [3.25, 12.75]]) {
    const screen = project(x, y);
    assert.deepEqual(unproject(screen.x, screen.y), { x, y });
  }
});

test('ground navigation walks around an occupied tile without leaving the island', () => {
  const blocked = new Set(['8,8']);
  const path = findPath({ x: 7, y: 8 }, { x: 9, y: 8 }, blocked);
  assert.equal(path.length, 4);
  let last = { x: 7, y: 8 };
  for (const tile of path) {
    assert.ok(isPlayable(tile.x, tile.y));
    assert.ok(!blocked.has(tileKey(tile.x, tile.y)));
    assert.equal(Math.abs(last.x - tile.x) + Math.abs(last.y - tile.y), 1);
    last = tile;
  }
  assert.deepEqual(last, { x: 9, y: 8 });
  assert.deepEqual(findPath(last, { x: 64, y: 3 }, blocked), []);
});

test('station navigation stops at the closest reachable entrance', () => {
  const station = { x: 4, y: 4 };
  const blocked = new Set(['4,4', '5,4']);
  const route = routeToStation({ x: 8, y: 4 }, station, blocked);
  assert.ok(route.path.length > 0);
  assert.equal(Math.abs(route.target.x - 4) + Math.abs(route.target.y - 4), 1);
  assert.ok(route.path.every(({ x, y }) => !blocked.has(tileKey(x, y))));
  assert.deepEqual(routeToStation({ x: 4, y: 5 }, station, blocked).path, []);
  assert.equal(routeToStation({ x: 8, y: 4 }, station, new Set(['4,4', '3,4', '5,4', '4,3', '4,5'])), null);
});
