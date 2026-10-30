import rules from '../../../../../contracts/blaster-rules.v1.json';
import type { Actor, Participant } from '../history/MatchDetail';
export type BlasterControls = {
    dx: number;
    dy: number;
    aim: number;
    fire: boolean;
};
export type BlasterPlayer = {
    actorId: string;
    x: number;
    y: number;
    aim: number;
    hp: number;
    kills: number;
    hitAt: number;
};
export type BlasterBot = {
    id: number;
    x: number;
    y: number;
    nextFire: number;
};
export type Projectile = {
    id: number;
    x: number;
    y: number;
    vx: number;
    vy: number;
    owner: number;
    ttl: number;
};
export type BlasterResult = {
    startsAt: number;
    finishedAt: number;
    participants: Participant[];
    blaster: {
        waves: number;
        kills: number;
        reason: string;
    };
};
export type BlasterSnapshot = {
    matchId: string;
    phase: string;
    frame: number;
    serverTimeMs: number;
    startsAt: number;
    deadlineAt: number;
    wave: number;
    actors: Actor[];
    players: BlasterPlayer[];
    bots: BlasterBot[];
    projectiles: Projectile[];
    result?: BlasterResult;
};
export const neutralBlaster = (): BlasterControls => ({ dx: 0, dy: 0, aim: 0, fire: false });
const clamp = (v: number, lo: number, hi: number) => Math.max(lo, Math.min(hi, v));
const distance = (x: number, y: number, ax: number, ay: number, bx: number, by: number) => { const dx = bx - ax, dy = by - ay, n = dx * dx + dy * dy, t = n > 0 ? clamp(((x - ax) * dx + (y - ay) * dy) / n, 0, 1) : 0; return Math.hypot(x - ax - t * dx, y - ay - t * dy); };
/** Deterministic 20 Hz mirror; input carries intent, never positions or hits. */
export class LocalBlaster {
    private at: number;
    private frame = 0;
    private wave = 1;
    private nextID = 0;
    private kills = 0;
    private players: BlasterPlayer[];
    private bots: BlasterBot[] = [];
    private projectiles: Projectile[] = [];
    private controls = Array.from({ length: 4 }, neutralBlaster);
    private seq = [0, 0, 0, 0];
    private expires = [0, 0, 0, 0];
    private nextFire = [0, 0, 0, 0];
    private disconnected = [0, 0, 0, 0];
    private left = [false, false, false, false];
    private result?: BlasterResult;
    constructor(private id: string, private actors: Actor[], private started: number) {
        if (actors.length < 1 || actors.length > 4 || actors.some(a => a.actorType !== 'human' || !a.userId))
            throw Error('Blaster membutuhkan 1–4 manusia');
        this.actors = structuredClone(actors);
        this.at = started;
        this.players = actors.map((a, i) => ({ actorId: a.actorId, x: .35 + i * .1, y: .5, aim: 0, hp: 3, kills: 0, hitAt: 0 }));
        this.spawn();
    }
    private actor(user: string) { return this.actors.findIndex(a => a.userId === user && a.actorType === 'human'); }
    input(user: string, seq: number, c: BlasterControls, at: number) {
        this.tick(at);
        const i = this.actor(user);
        if (i < 0 || this.result || this.left[i] || this.disconnected[i] || !this.players[i].hp || seq <= this.seq[i] || ![c.dx, c.dy, c.aim].every(Number.isFinite) || Math.abs(c.dx) > 1 || Math.abs(c.dy) > 1 || Math.abs(c.aim) > Math.PI)
            return false;
        this.seq[i] = seq;
        this.controls[i] = { ...c };
        this.expires[i] = at + rules.inputExpiryMs;
        return true;
    }
    private shoot(x: number, y: number, aim: number, owner: number) {
        if (this.projectiles.length >= rules.maxProjectiles)
            return;
        this.projectiles.push({ id: ++this.nextID, x, y, vx: Math.cos(aim) * rules.projectileSpeed, vy: Math.sin(aim) * rules.projectileSpeed, owner, ttl: rules.projectileTTL });
    }
    private spawn() {
        this.bots = [];
        this.projectiles = [];
        this.players.forEach((p, i) => {
            if (!this.left[i] && !this.disconnected[i])
                p.hp = rules.hp;
            this.controls[i] = { ...neutralBlaster(), aim: p.aim };
        });
        for (let i = 0; i < rules.waves[this.wave - 1]; i++) {
            let bestX = .05, bestY = .05, best = -1;
            for (let j = 0; j < 32; j++) {
                const side = Math.floor(j / 8), t = j % 8 / 7 * .9 + .05;
                let x = t, y = .05;
                if (side === 1) {
                    x = .95;
                    y = t;
                }
                else if (side === 2) {
                    x = t;
                    y = .95;
                }
                else if (side === 3) {
                    x = .05;
                    y = t;
                }
                let d = 2;
                for (const p of [...this.players, ...this.bots])
                    d = Math.min(d, Math.hypot(x - p.x, y - p.y));
                if (d > best) {
                    bestX = x;
                    bestY = y;
                    best = d;
                }
            }
            this.bots.push({ id: (this.wave - 1) * 8 + i, x: bestX, y: bestY, nextFire: this.at + rules.botCooldownMs + i * 100 });
        }
    }
    private step() {
        const dt = rules.tickMs / 1000;
        let alive = false;
        this.players.forEach((p, i) => {
            if (this.left[i] || this.disconnected[i] > 0 && this.at >= this.disconnected[i] + rules.disconnectGraceMs)
                p.hp = 0;
            if (!p.hp)
                return;
            alive = true;
            const c = this.expires[i] <= this.at || this.disconnected[i] ? { ...neutralBlaster(), aim: p.aim } : this.controls[i], n = Math.max(1, Math.hypot(c.dx, c.dy));
            p.x = clamp(p.x + c.dx / n * rules.playerSpeed * dt, rules.playerRadius, 1 - rules.playerRadius);
            p.y = clamp(p.y + c.dy / n * rules.playerSpeed * dt, rules.playerRadius, 1 - rules.playerRadius);
            p.aim = c.aim;
            if (c.fire && this.at >= this.nextFire[i]) {
                this.shoot(p.x, p.y, c.aim, i);
                this.nextFire[i] = this.at + rules.fireCooldownMs;
            }
        });
        if (!alive) {
            this.finish(this.at, 'all_down');
            return;
        }
        for (const b of this.bots) {
            let target = -1, distance = 2;
            this.players.forEach((p, i) => {
                const d = Math.hypot(p.x - b.x, p.y - b.y);
                if (p.hp > 0 && d < distance) {
                    target = i;
                    distance = d;
                }
            });
            if (target < 0)
                continue;
            const p = this.players[target], aim = Math.atan2(p.y - b.y, p.x - b.x);
            if (distance > .2) {
                b.x = clamp(b.x + Math.cos(aim) * rules.botSpeed * dt, .025, .975);
                b.y = clamp(b.y + Math.sin(aim) * rules.botSpeed * dt, .025, .975);
            }
            if (this.at >= b.nextFire) {
                this.shoot(b.x, b.y, aim, -1);
                b.nextFire = this.at + rules.botCooldownMs;
            }
        }
        this.projectiles = this.projectiles.filter(p => {
            const oldX = p.x, oldY = p.y;
            p.x += p.vx * dt;
            p.y += p.vy * dt;
            p.ttl -= rules.tickMs;
            if (p.ttl <= 0 || p.x < 0 || p.x > 1 || p.y < 0 || p.y > 1)
                return false;
            if (p.owner >= 0) {
                const j = this.bots.findIndex(b => distance(b.x, b.y, oldX, oldY, p.x, p.y) <= rules.botRadius + rules.projectileRadius);
                if (j >= 0) {
                    this.bots.splice(j, 1);
                    this.kills++;
                    this.players[p.owner].kills++;
                    return false;
                }
            }
            else {
                const v = this.players.find(v => v.hp > 0 && distance(v.x, v.y, oldX, oldY, p.x, p.y) <= rules.playerRadius + rules.projectileRadius);
                if (v) {
                    v.hp--;
                    v.hitAt = this.at;
                    return false;
                }
            }
            return true;
        });
        if (!this.players.some(p => p.hp > 0)) {
            this.finish(this.at, 'all_down');
            return;
        }
        if (!this.bots.length) {
            if (this.wave === rules.waves.length)
                this.finish(this.at, 'cleared');
            else {
                this.wave++;
                this.spawn();
            }
        }
    }
    tick(now: number) {
        if (this.result)
            return;
        const deadline = this.started + rules.deadlineMs, limit = Math.min(now, deadline);
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
        if (!this.result && now >= deadline)
            this.finish(deadline, 'timeout');
    }
    presence(user: string, kind: string, at: number) {
        const i = this.actor(user);
        if (i < 0 || this.result)
            return;
        if (kind === 'disconnect') {
            if (!this.disconnected[i])
                this.disconnected[i] = at;
            this.controls[i] = neutralBlaster();
            return;
        }
        this.tick(at);
        if (this.result)
            return;
        this.controls[i] = neutralBlaster();
        if (kind === 'resume') {
            this.disconnected[i] = 0;
            this.seq[i] = 0;
        }
        if (kind === 'leave' || kind === 'expired') {
            this.left[i] = true;
            this.players[i].hp = 0;
            if (!this.players.some(p => p.hp > 0))
                this.finish(at, 'all_down');
        }
    }
    private finish(at: number, reason: string) {
        if (this.result)
            return;
        this.result = { startsAt: this.started, finishedAt: at, blaster: { waves: reason === 'cleared' ? 3 : this.wave - 1, kills: this.kills, reason }, participants: this.actors.map((a, i) => ({ ...a, botAssisted: false, outcome: this.left[i] ? 'forfeit' : reason === 'cleared' ? 'win' : 'loss', score: this.players[i].kills })) };
        this.projectiles = [];
        this.frame++;
    }
    snapshot(): BlasterSnapshot { return structuredClone({ matchId: this.id, phase: this.result ? 'results' : 'playing', frame: this.frame, serverTimeMs: this.at, startsAt: this.started, deadlineAt: this.started + rules.deadlineMs, wave: this.wave, actors: this.actors, players: this.players, bots: this.bots, projectiles: this.projectiles, ...(this.result ? { result: this.result } : {}) }); }
}
