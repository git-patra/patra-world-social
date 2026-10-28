import type { Vehicle, Football } from './contracts';
import type { ActivityState, ActivityGame } from './store';
export type SceneActor = {
    id: string;
    userId?: string;
    x: number;
    y: number;
    label: string;
    draw: (ctx: CanvasRenderingContext2D, at: number) => void;
};
type Adapter = (game: ActivityGame, at: number, previous?: ActivityGame, fraction?: number) => SceneActor[];
/** Adapters load on first public match; Canvas shares the world's only animation loop. */
export class ActivityScene {
    private footballAdapter: ((ball: Football) => SceneActor[]) | null = null;
    private loadingFootball = false;
    private vehicleAdapter: ((vehicles: Vehicle[]) => SceneActor[]) | null = null;
    private loadingVehicles = false;
    private adapters = new Map<string, Adapter>();
    private loading = new Set<string>();
    private state: ActivityState | null = null;
    private received = 0;
    private samples = new Map<string, {
        current: ActivityGame;
        previous: ActivityGame;
        received: number;
    }>();
    constructor(private changed: () => void) {
    }
    set(state: ActivityState) {
        this.state = state;
        if (state.football && !this.footballAdapter && !this.loadingFootball) {
            this.loadingFootball = true;
            void import('../world/scenes/football').then(module => {
                this.footballAdapter = module.footballActors;
                this.changed();
            }).catch(() => {
                this.loadingFootball = false;
            });
        }
        if (state.vehicles?.length && !this.vehicleAdapter && !this.loadingVehicles) {
            this.loadingVehicles = true;
            void import('../world/scenes/vehicles').then(module => {
                this.vehicleAdapter = module.vehicleActors;
                this.changed();
            }).catch(() => {
                this.loadingVehicles = false;
            });
        }
        this.received = performance.now();
        for (const id of this.samples.keys())
            if (!state.games.some(g => g.matchId === id))
                this.samples.delete(id);
        for (const game of state.games) {
            const sample = this.samples.get(game.matchId);
            if (sample?.current.snapshot !== game.snapshot)
                this.samples.set(game.matchId, { current: game, previous: sample?.current ?? game, received: this.received });
            if (this.adapters.has(game.kind) || this.loading.has(game.kind))
                continue;
            this.loading.add(game.kind);
            void (game.kind === 'pingpong' ? import('../world/scenes/pingpong') : game.kind === 'race' ? import('../world/scenes/race') : import('../world/scenes/penalty')).then(module => {
                this.adapters.set(game.kind, module.sceneActors);
                this.changed();
            }).catch(() => {
                this.loading.delete(game.kind);
            });
        }
    }
    private visibleVehicles() {
        return !this.state?.online ? this.state?.inputMode !== 'race' : this.state.areas.find(a => a.areaId === 'race')?.payload.phase === 'OPEN';
    }
    private visible(game: ActivityGame) {
        const area = this.state?.areas.find(a => a.areaId === game.kind);
        return !this.state?.online || !area || area.payload.phase !== 'OPEN' && area.payload.matchId === game.matchId;
    }
    actors(_at: number) {
        const at = (this.state?.now ?? 0) + performance.now() - this.received;
        return [...(this.state?.football ? this.footballAdapter?.(this.state.football) ?? [] : []), ...(this.visibleVehicles() ? this.vehicleAdapter?.(this.state?.vehicles ?? []) ?? [] : []), ...(this.state?.games.filter(game => this.visible(game)).flatMap(game => {
                const sample = this.samples.get(game.matchId), current = game.snapshot, previous = sample?.previous.snapshot;
                const duration = 'serverTimeMs' in current && previous && 'serverTimeMs' in previous ? current.serverTimeMs - previous.serverTimeMs : 0;
                const fraction = duration <= 0 || duration > 250 ? 1 : Math.min(1, (performance.now() - (sample?.received ?? 0)) / duration);
                return this.adapters.get(game.kind)?.(game, at, sample?.previous, fraction) ?? [];
            }) ?? [])];
    }
    ownActor(actors: SceneActor[]) {
        return actors.find(a => a.userId === this.state?.selfId);
    }
    replaces(id: string) {
        return Boolean(this.visibleVehicles() && this.state?.vehicles?.some(v => v.riderId === id)) || (this.state?.games.filter(game => this.visible(game)).some(g => g.kind === 'race' ? 'karts' in g.snapshot && g.snapshot.karts.some(k => k.userId === id) : 'actors' in g.snapshot && g.snapshot.actors.some(a => a.userId === id)) ?? false);
    }
}
