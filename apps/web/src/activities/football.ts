import { footballInside, footballRules, type Football } from './contracts';

export const footballSpawn = (visible = true): Football => ({ x: 28, y: 46, vx: 0, vy: 0, visible });
export const nearFootball = (ball: Football | null, p: { x: number; y: number }) => Boolean(ball?.visible && Math.hypot(ball.x - p.x, ball.y - p.y) <= footballRules.kickRadius);

/** Guest ball shares the application activity clock and the owner's bounded physics. */
export class LocalFootball {
    ball = footballSpawn();
    revision = 1;
    private at = 0;
    private activeAt = 0;
    private lastKick = -Infinity;
    reset(visible = true, at = this.at) {
        this.ball = footballSpawn(visible);
        this.revision++;
        this.at = this.activeAt = at;
        this.lastKick = -Infinity;
    }
    kick(p: { x: number; y: number }, dx: number, dy: number, at: number) {
        const length = Math.hypot(dx, dy);
        if (!nearFootball(this.ball, p) || at - this.lastKick < footballRules.cooldownMs || !Number.isFinite(length) || length === 0 || Math.abs(dx) > 1 || Math.abs(dy) > 1)
            return false;
        this.ball = { ...this.ball, vx: dx / length * footballRules.speed, vy: dy / length * footballRules.speed };
        this.activeAt = this.lastKick = at;
        this.revision++;
        return true;
    }
    tick(at: number) {
        if (!this.ball.visible)
            return;
        if (!footballInside(this.ball.x, this.ball.y) || at - this.activeAt >= footballRules.idleMs) {
            this.reset(true, at);
            return;
        }
        let steps = 0;
        while (this.at + 50 <= at && steps++ < 4) {
            this.at += 50;
            const b = { ...this.ball }, dx = b.vx * .05, dy = b.vy * .05;
            const samples = Math.ceil(Math.hypot(dx, dy) / .05), x = b.x, y = b.y;
            for (let j = 1; j <= samples; j++) {
                const nx = x + dx * j / samples, ny = y + dy * j / samples;
                if (!footballInside(nx, ny)) {
                    b.vx = b.vy = 0;
                    break;
                }
                b.x = nx;
                b.y = ny;
            }
            b.vx *= footballRules.friction;
            b.vy *= footballRules.friction;
            if (Math.hypot(b.vx, b.vy) < .05)
                b.vx = b.vy = 0;
            this.ball = b;
        }
        if (this.at + 50 <= at)
            this.at = at;
    }
}
