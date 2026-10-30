import { registerGuestSession } from '../../guest/lifecycle';
import { Completion } from '../../guest/completion';
import { preferences } from '../../guest/history';
import { LocalBlaster, type BlasterControls } from './local';

/** Advanced by ActivityStore, independently of the lobby/HUD lifetime. */
export class GuestBlasterSession {
    private engine: LocalBlaster | null = null;
    private completion: Completion | null = null;
    private seq = 0;
    private lastFinishAttempt = -Infinity;

    start(at: number, _seed?: number, id: string = crypto.randomUUID()) {
        this.clear();
        this.completion = new Completion(id, 'blaster');
        this.engine = new LocalBlaster(id, [
            { actorId: 'guest', userId: 'guest', actorType: 'human', displayName: preferences().displayName, avatarId: preferences().avatar }
        ], at);
        return this.snapshot(at)!;
    }

    snapshot(at: number) {
        if (!this.engine) return null;
        this.engine.tick(at);
        const snapshot = this.engine.snapshot();
        if (snapshot.result && at - this.lastFinishAttempt >= 1000) {
            this.lastFinishAttempt = at;
            this.completion?.finish(snapshot.result.finishedAt - snapshot.startsAt, `${snapshot.result.blaster.waves}/3 wave · ${snapshot.result.blaster.kills} bot`, snapshot);
        }
        return snapshot;
    }

    blast(controls:BlasterControls,at:number){return this.engine?.input('guest',++this.seq,controls,at)??false}

    clear() {
        this.engine = null;
        this.completion = null;
        this.seq = 0;
        this.lastFinishAttempt = -Infinity;
    }
}
export const guestBlaster = new GuestBlasterSession();
registerGuestSession(guestBlaster);
