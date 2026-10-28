import type { ActivityGame } from '../../activities/store';
import type { SceneActor } from '../../activities/scene';
import type { PingpongSnapshot } from '../../features/pingpong/local';
/** Same world canvas/RAF; interpolation only, never client scoring. */
export function sceneActors(game: ActivityGame, _at: number, previous?: ActivityGame, fraction = 1): SceneActor[] {
    const s = game.snapshot as PingpongSnapshot, old = previous?.snapshot as PingpongSnapshot | undefined;
    const blend = (a: number, b: number) => a + (b - a) * fraction;
    const actors: SceneActor[] = s.actors.map((a, i) => {
        const paddle = old && old.rallyId === s.rallyId ? blend(old.paddles[i], s.paddles[i]) : s.paddles[i], x = 1745 + (i === 0 ? 8 : 150), y = 543 + paddle * 51;
        return { id: `pingpong:${a.actorId}`, userId: a.userId, x, y: 622, label: a.displayName, draw(ctx) {
                ctx.fillStyle = i ? '#d8ad67' : '#c97857';
                ctx.fillRect(x - 3, y - 7, 6, 14);
                ctx.fillStyle = '#e3c294';
                ctx.fillRect(x - 2, y + 7, 4, 6);
            } };
    });
    const ball = old && old.rallyId === s.rallyId && old.phase === s.phase ? { x: blend(old.ball.x, s.ball.x), y: blend(old.ball.y, s.ball.y) } : s.ball;
    actors.push({ id: 'pingpong:ball', x: 1745 + ball.x * 158, y: 623, label: '', draw(ctx) {
            ctx.fillStyle = '#fff0d4';
            ctx.beginPath();
            ctx.arc(1745 + ball.x * 158, 543 + ball.y * 51, 3, 0, Math.PI * 2);
            ctx.fill();
        } });
    return actors;
}
