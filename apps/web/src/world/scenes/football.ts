import type { Football } from '../../activities/contracts';
import type { SceneActor } from '../../activities/scene';

export function footballActors(ball: Football): SceneActor[] {
    if (!ball.visible)
        return [];
    const x = (ball.x + .5) * 40, y = (ball.y + .5) * 40 - 320;
    return [{
        id: 'football:ball', x, y, label: '',
        draw(ctx) {
            ctx.fillStyle = '#344c4340';
            ctx.beginPath();
            ctx.ellipse(x, y + 3, 7, 3, 0, 0, Math.PI * 2);
            ctx.fill();
            ctx.fillStyle = '#fff0cc';
            ctx.beginPath();
            ctx.arc(x, y - 2, 6, 0, Math.PI * 2);
            ctx.fill();
            ctx.fillStyle = '#405847';
            ctx.fillRect(x - 2, y - 4, 3, 3);
        }
    }];
}
