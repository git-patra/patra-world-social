import type { Vehicle, Football, Claw, FishingView } from './contracts';
import type { ActivityState, ActivityGame } from './store';
export type SceneActor = {
    id: string;
    userId?: string;
    x: number;
    y: number;
    label: string;
    draw: (ctx: CanvasRenderingContext2D, at: number) => void;
};
// Spectators normally arrive every300ms; shared visual budgets may delay a
// full capped snapshot beyond one second. Bound smoothing separately from the
// discontinuity window so delayed updates do not add unbounded visual latency.
const MAX_CONTINUOUS_GAP_MS = 2000;
const MAX_BLEND_MS = 600;
function sceneIdentity(game: ActivityGame, state: ActivityState) {
    const snapshot = game.snapshot, area = state.areas.find(a => a.areaId === game.kind);
    const actors = 'karts' in snapshot ? snapshot.karts : snapshot.actors;
    return JSON.stringify([game.kind, game.matchId, state.online, snapshot.phase,
        area?.worldId, area?.incarnation, area?.generation, area?.payload.phase, area?.payload.matchId,
        'wave' in snapshot ? snapshot.wave : null, 'rallyId' in snapshot ? snapshot.rallyId : null,
        actors.map(actor => [actor.actorId, actor.userId, actor.actorType])]);
}
type Adapter = (game: ActivityGame, at: number, previous?: ActivityGame, fraction?: number) => SceneActor[];
/** Adapters load on first public match; Canvas shares the world's only animation loop. */
export class ActivityScene {
    private fishingAdapter:((spots:FishingView[],at:number)=>SceneActor[])|null=null;
    private loadingFishing=false;
    private clawAdapter:((claw:Claw,at:number)=>SceneActor[])|null=null;
    private loadingClaw=false;
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
        identity: string;
        duration: number;
    }>();
    constructor(private changed: () => void) {
    }
    set(state: ActivityState) {
        this.state = state;
        if(state.fishing?.length&&!this.fishingAdapter&&!this.loadingFishing){this.loadingFishing=true;void import('../world/scenes/fishing').then(module=>{this.fishingAdapter=module.fishingActors;this.changed()}).catch(()=>{this.loadingFishing=false})}
        if(state.claw&&!this.clawAdapter&&!this.loadingClaw){
            this.loadingClaw=true;
            void import('../world/scenes/claw').then(module=>{this.clawAdapter=module.clawActors;this.changed()}).catch(()=>{this.loadingClaw=false});
        }
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
            const sample = this.samples.get(game.matchId), identity = sceneIdentity(game, state);
            if (sample?.current.snapshot !== game.snapshot || sample.identity !== identity) {
                const current = game.snapshot, previous = sample?.current.snapshot;
                const interval = 'serverTimeMs' in current && previous && 'serverTimeMs' in previous
                    ? current.serverTimeMs - previous.serverTimeMs : 0;
                const receiptGap = this.received - (sample?.received ?? this.received);
                const continuous = sample?.identity === identity && interval > 0 && interval <= MAX_CONTINUOUS_GAP_MS
                    && receiptGap >= 0 && receiptGap <= MAX_CONTINUOUS_GAP_MS;
                this.samples.set(game.matchId, { current: game, previous: continuous ? sample.current : game,
                    received: this.received, identity, duration: continuous ? Math.min(interval, MAX_BLEND_MS) : 0 });
            }
            if (this.adapters.has(game.kind) || this.loading.has(game.kind))
                continue;
            this.loading.add(game.kind);
            void (game.kind === 'blaster' ? import('../world/scenes/blaster') : game.kind === 'pingpong' ? import('../world/scenes/pingpong') : game.kind === 'race' ? import('../world/scenes/race') : import('../world/scenes/penalty')).then(module => {
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
        return [...(this.fishingAdapter?.(this.state?.fishing??[],at)??[]),...(this.state?.claw?this.clawAdapter?.(this.state.claw,at)??[]:[]),...(this.state?.football ? this.footballAdapter?.(this.state.football) ?? [] : []), ...(this.visibleVehicles() ? this.vehicleAdapter?.(this.state?.vehicles ?? []) ?? [] : []), ...(this.state?.games.filter(game => this.visible(game)).flatMap(game => {
                const sample = this.samples.get(game.matchId), duration = sample?.duration ?? 0;
                const fraction = duration <= 0 ? 1 : Math.max(0, Math.min(1, (performance.now() - sample!.received) / duration));
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
