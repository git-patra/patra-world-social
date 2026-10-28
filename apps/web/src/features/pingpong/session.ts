import { registerGuestSession } from '../../guest/lifecycle';
import { Completion } from '../../guest/completion';
import { preferences } from '../../guest/history';
import { LocalPingpong } from './local';

/** Advanced by ActivityStore, independently of the lobby/HUD lifetime. */
export class GuestPingpongSession {
    private engine: LocalPingpong | null = null;
    private completion: Completion | null = null;
    private seq = 0;
    private lastFinishAttempt = -Infinity;

    start(at: number, _seed?: number, id: string = crypto.randomUUID()) {
        this.clear();
        this.completion = new Completion(id, 'pingpong');
        this.engine = new LocalPingpong(id, [
            { actorId: 'guest', userId: 'guest', actorType: 'human', displayName: preferences().displayName, avatarId: preferences().avatar },
            { actorId: 'bot', actorType: 'bot', displayName: 'BOT', avatarId: 'pumpkin' }
        ], at);
        return this.snapshot(at)!;
    }

    snapshot(at: number) {
        if (!this.engine) return null;
        this.engine.tick(at);
        const snapshot = this.engine.snapshot();
        if (snapshot.result && at - this.lastFinishAttempt >= 1000) {
            this.lastFinishAttempt = at;
            this.completion?.finish(snapshot.result.finishedAt - snapshot.startsAt, snapshot.scores.join(' : '), snapshot);
        }
        return snapshot;
    }

    paddle(axis: number, at: number) {
        return this.engine?.input('guest', ++this.seq, axis, at) ?? false;
    }

    serve(rally: number, at: number) {
        return this.engine?.serve('guest', at, rally) ?? false;
    }

    clear() {
        this.engine = null;
        this.completion = null;
        this.seq = 0;
        this.lastFinishAttempt = -Infinity;
    }
}
export const guestPingpong = new GuestPingpongSession();
registerGuestSession(guestPingpong);
