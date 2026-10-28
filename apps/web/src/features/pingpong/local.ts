import type { Actor, Participant } from '../history/MatchDetail';
import rules from '../../../../../contracts/pingpong-rules.v1.json';
export type PingpongResult = {
    startsAt: number;
    finishedAt: number;
    participants: Participant[];
    pingpong: {
        scores: number[];
        rallies: number;
        reason: string;
    };
};
export type PingpongSnapshot = {
    matchId: string;
    phase: string;
    frame: number;
    serverTimeMs: number;
    startsAt: number;
    deadlineAt: number;
    rallyId: number;
    server: number;
    actors: Actor[];
    paddles: number[];
    scores: number[];
    ball: {
        x: number;
        y: number;
        vx: number;
        vy: number;
    };
    result?: PingpongResult;
};
const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));
export const won = (a: number, b: number) => a >= rules.capScore || a >= rules.targetScore && a - b >= rules.winBy;
/** Guest mirror of the authoritative 20Hz model. No canvas, timers or transport. */
export class LocalPingpong {
    private at: number;
    private frame = 0;
    private rally = 1;
    private server = 0;
    private paddles = [.5, .5];
    private scores = [0, 0];
    private ball = { x: .5, y: .5, vx: 0, vy: 0 };
    private playing = false;
    private axis = [0, 0];
    private expires = [0, 0];
    private seq = [0, 0];
    private disconnected = [0, 0];
    private result?: PingpongResult;
    constructor(private id: string, private actors: Actor[], private started: number) {
        this.at = started;
        this.actors = structuredClone(actors);
    }
    private actor(user: string) {
        return this.actors.findIndex(a => a.actorType === 'human' && a.userId === user);
    }
    input(user: string, seq: number, axis: number, at: number) {
        this.tick(at);
        const i = this.actor(user);
        if (i < 0 || this.result || this.disconnected[i] || seq <= this.seq[i] || !Number.isFinite(axis) || axis < -1 || axis > 1)
            return false;
        this.seq[i] = seq;
        this.axis[i] = axis;
        this.expires[i] = at + rules.inputExpiryMs;
        return true;
    }
    serve(user: string, at: number, rally = this.rally) {
        this.tick(at);
        const i = this.actor(user);
        if (i < 0 || i !== this.server || this.result || this.playing || this.disconnected[i] || at < this.started || rally !== this.rally)
            return false;
        this.startServe();
        return true;
    }
    private startServe() {
        this.playing = true;
        this.ball = { x: .5, y: .5, vx: rules.ballSpeedX * (this.server === 0 ? 1 : -1), vy: rules.serveSpeedY * (this.rally % 2 === 0 ? -1 : 1) };
    }
    private step() {
        for (let i = 0; i < 2; i++) {
            let axis = this.axis[i], speed = rules.paddleSpeed;
            if (this.expires[i] < this.at || this.disconnected[i])
                axis = 0;
            if (this.actors[i].actorType === 'bot') {
                axis = this.ball.y > this.paddles[i] + rules.botDeadzone ? 1 : this.ball.y < this.paddles[i] - rules.botDeadzone ? -1 : 0;
                speed = rules.botSpeed;
            }
            this.paddles[i] = clamp(this.paddles[i] + axis * speed * .05, .12, .88);
        }
        if (!this.playing) {
            if (this.actors[this.server].actorType === 'bot')
                this.startServe();
            return;
        }
        const old = { ...this.ball };
        this.ball.x += this.ball.vx * .05;
        this.ball.y += this.ball.vy * .05;
        if (this.ball.y < .02) {
            this.ball.y = .04 - this.ball.y;
            this.ball.vy = Math.abs(this.ball.vy);
        }
        if (this.ball.y > .98) {
            this.ball.y = 1.96 - this.ball.y;
            this.ball.vy = -Math.abs(this.ball.vy);
        }
        rules.paddleX.forEach((x, i) => {
            const cross = i === 0 && old.x >= x && this.ball.x <= x || i === 1 && old.x <= x && this.ball.x >= x;
            if (cross && Math.abs(this.ball.y - this.paddles[i]) <= .14) {
                this.ball.x = 2 * x - this.ball.x;
                this.ball.vx = -this.ball.vx;
                this.ball.vy = clamp((this.ball.y - this.paddles[i]) * 3, -.45, .45);
            }
        });
        const scorer = this.ball.x < 0 ? 1 : this.ball.x > 1 ? 0 : -1;
        if (scorer >= 0) {
            this.scores[scorer]++;
            if (won(this.scores[scorer], this.scores[1 - scorer])) {
                this.finish(this.at, -1, 'score');
                return;
            }
            this.server = 1 - scorer;
            this.rally++;
            this.playing = false;
            this.ball = { x: .5, y: .5, vx: 0, vy: 0 };
        }
    }
    tick(now: number) {
        if (this.result)
            return;
        let cutoff = this.started + rules.deadlineMs, loser = -1;
        this.disconnected.forEach((at, i) => {
            if (at > 0 && at + rules.disconnectGraceMs < cutoff) {
                cutoff = at + rules.disconnectGraceMs;
                loser = i;
            }
        });
        const limit = Math.min(now, cutoff);
        let steps = 0;
        while (this.at + rules.tickMs <= limit && steps < rules.maxCatchupSteps && !this.result) {
            this.at += rules.tickMs;
            this.frame++;
            this.step();
            steps++;
        }
        if (this.at + rules.tickMs <= limit) {
            this.at = limit - (limit - this.started) % rules.tickMs;
            this.frame = (this.at - this.started) / rules.tickMs;
        }
        if (!this.result && now >= cutoff)
            this.finish(cutoff, loser, loser < 0 ? 'deadline' : 'forfeit');
    }
    presence(user: string, kind: string, at: number) {
        const i = this.actor(user);
        if (i < 0)
            return;
        if (kind === 'disconnect') {
            if (!this.disconnected[i])
                this.disconnected[i] = at;
            this.axis[i] = 0;
            return;
        }
        this.tick(at);
        if (this.result)
            return;
        if (kind === 'resume') {
            this.disconnected[i] = 0;
            this.axis[i] = 0;
            this.seq[i] = 0;
        }
        if (kind === 'leave')
            this.finish(at, i, 'forfeit');
    }
    private finish(at: number, loser: number, reason: string) {
        if (this.result)
            return;
        this.result = { startsAt: this.started, finishedAt: at, pingpong: { scores: [...this.scores], rallies: this.rally, reason }, participants: this.actors.map((a, i) => ({ ...a, botAssisted: false, score: this.scores[i], position: loser === i || loser < 0 && this.scores[i] < this.scores[1 - i] ? 2 : 1, outcome: loser >= 0 ? loser === i ? 'forfeit' : 'win' : this.scores[i] > this.scores[1 - i] ? 'win' : this.scores[i] < this.scores[1 - i] ? 'loss' : 'draw' })) };
        this.ball.vx = 0;
        this.ball.vy = 0;
        this.frame++;
    }
    snapshot(): PingpongSnapshot {
        return structuredClone({ matchId: this.id, phase: this.result ? 'results' : this.playing ? 'rally' : 'serve', frame: this.frame, serverTimeMs: this.at, startsAt: this.started, deadlineAt: this.started + rules.deadlineMs, rallyId: this.rally, server: this.server, actors: this.actors, paddles: this.paddles, scores: this.scores, ball: this.ball, ...(this.result ? { result: this.result } : {}) });
    }
}
