import { rules, type Point } from './map';
/** Authoritative movement units are tiles; wire/race values use integer 1/1024 tiles. Motion is visual only. */
export function advanceMovement(position: Point, target: Point, dt: number): Point { const dx = target.x - position.x, dy = target.y - position.y, d = Math.hypot(dx, dy), step = Math.max(0, dt) * rules.hubSpeedSubunitsPerSecond / rules.subunitsPerTile; return d <= step + 1e-10 ? { ...target } : { x: position.x + dx / d * step, y: position.y + dy / d * step }; }
export const toSubunits = (tiles: number) => Math.round(tiles * rules.subunitsPerTile);
export function acceptsMovement(target: EventTarget | null) { return !(target instanceof HTMLElement) || !target.closest('input,textarea,select,[contenteditable="true"]'); }

/** Pointer-up bubbles from canvas clicks too; stop only a hold we actually own. */
export class PointerHold {
    private timer:ReturnType<typeof setInterval>|null=null;
    private pulseTimer:ReturnType<typeof setTimeout>|null=null;
    constructor(private move:(dx:number,dy:number)=>void,private stop:()=>void){}
    start(dx:number,dy:number){
        this.release();
        this.move(dx,dy);
        this.timer=setInterval(()=>this.move(dx,dy),170);
    }
    /** Semantic click (keyboard/assistive technology) is one bounded movement pulse. */
    pulse(dx:number,dy:number){
        this.start(dx,dy);
        this.pulseTimer=setTimeout(()=>this.release(),180);
    }
    release(){
        if(this.pulseTimer!==null){clearTimeout(this.pulseTimer);this.pulseTimer=null;}
        if(this.timer===null)return;
        clearInterval(this.timer);
        this.timer=null;
        this.stop();
    }
}
