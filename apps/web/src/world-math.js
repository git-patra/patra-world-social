export const GRID_SIZE = 64;
export const TILE_WIDTH = 30;
export const TILE_HEIGHT = 15;

export const tileKey = (x, y) => `${x},${y}`;
export const project = (x, y) => ({ x: (x - y) * TILE_WIDTH, y: (x + y) * TILE_HEIGHT });
export const unproject = (x, y) => ({ x: (x / TILE_WIDTH + y / TILE_HEIGHT) / 2, y: (y / TILE_HEIGHT - x / TILE_WIDTH) / 2 });
export const isPlayable = (x, y) => Number.isInteger(x) && Number.isInteger(y) && x >= 0 && x < GRID_SIZE && y >= 0 && y < GRID_SIZE;

/** Breadth-first search on a small unweighted grid. Occupied tiles are never entered. */
export function findPath(start, goal, blocked = new Set()) {
  if (!isPlayable(goal.x, goal.y) || blocked.has(tileKey(goal.x, goal.y))) return [];
  const origin = { x: Math.round(start.x), y: Math.round(start.y) };
  if (origin.x === goal.x && origin.y === goal.y) return [];
  const queue = [origin];
  const visited = new Map([[tileKey(origin.x, origin.y), null]]);
  let cursor = 0;
  while (cursor < queue.length) {
    const current = queue[cursor++];
    for (const [dx, dy] of [[1, 0], [0, 1], [-1, 0], [0, -1]]) {
      const next = { x: current.x + dx, y: current.y + dy };
      const key = tileKey(next.x, next.y);
      if (!isPlayable(next.x, next.y) || blocked.has(key) || visited.has(key)) continue;
      visited.set(key, current);
      if (next.x === goal.x && next.y === goal.y) {
        const route = [next];
        let previous = current;
        while (previous && (previous.x !== origin.x || previous.y !== origin.y)) {
          route.push(previous);
          previous = visited.get(tileKey(previous.x, previous.y));
        }
        return route.reverse();
      }
      queue.push(next);
    }
  }
  return [];
}

export function routeToStation(start, station, blocked) {
  const choices = [[1, 0], [0, 1], [-1, 0], [0, -1]].map(([dx, dy]) => ({ x: station.x + dx, y: station.y + dy }));
  let best = null;
  for (const target of choices) {
    if (!isPlayable(target.x, target.y) || blocked.has(tileKey(target.x, target.y))) continue;
    const sameTile = Math.round(start.x) === target.x && Math.round(start.y) === target.y;
    const path = findPath(start, target, blocked);
    if ((sameTile || path.length) && (!best || path.length < best.path.length)) best = { target, path };
  }
  return best;
}
