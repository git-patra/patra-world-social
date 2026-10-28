import { activityContract, rideDriveable, type Vehicle } from './contracts';
export const vehiclePool = (): Vehicle[] => activityContract.vehicleSlots.map(s => ({ id: s.id, kind: s.kind as Vehicle['kind'], generation: 1, riderId: '', x: s.position.x, y: s.position.y, heading: 0 }));
export const nearbyVehicle = (vehicles: Vehicle[], p: {
    x: number;
    y: number;
}) => vehicles.filter(v => !v.riderId && Math.hypot(v.x - p.x, v.y - p.y) <= 2).sort((a, b) => Math.hypot(a.x - p.x, a.y - p.y) - Math.hypot(b.x - p.x, b.y - p.y))[0];
/** Guest simulation uses the application activity clock, with the owner's fixed step and expiry. */
export class LocalVehicles {
    vehicles = vehiclePool();
    private at = 0;
    private dx = 0;
    private dy = 0;
    private expires = 0;
    mount(id: string, p: {
        x: number;
        y: number;
    }, at: number) {
        const v = this.vehicles.find(v => v.id === id);
        if (!v || v.riderId || this.rider() || Math.hypot(v.x - p.x, v.y - p.y) > 2)
            return false;
        v.riderId = 'guest';
        v.generation++;
        this.at = at;
        this.dx = this.dy = 0;
        return true;
    }
    rider() {
        return this.vehicles.find(v => v.riderId === 'guest');
    }
    dismount() {
        const v = this.rider();
        if (v) {
            v.riderId = '';
            v.generation++;
        }
        this.dx = this.dy = 0;
    }
    reset() {
        this.dismount();
        this.vehicles = vehiclePool().map((v, i) => ({ ...v, generation: this.vehicles[i].generation + 1 }));
    }
    controls(x: number, y: number, at: number) {
        this.dx = Math.max(-1, Math.min(1, x));
        this.dy = Math.max(-1, Math.min(1, y));
        this.expires = at + 250;
    }
    tick(at: number) {
        let stepsThisTick = 0;
        while (this.at + 50 <= at) {
            this.at += 50;
            const v = this.rider();
            if (v && this.at <= this.expires && stepsThisTick++ < 4) {
                const length = Math.hypot(this.dx, this.dy), speed = v.kind === 'car' ? 8 : 6, dx = this.dx / Math.max(1, length) * speed * .05, dy = this.dy / Math.max(1, length) * speed * .05;
                if (length) {
                    const steps = Math.ceil(Math.hypot(dx, dy) / .05), x = v.x, y = v.y;
                    for (let j = 1; j <= steps; j++) {
                        const nx = x + dx * j / steps, ny = y + dy * j / steps;
                        if (!rideDriveable(nx, ny))
                            break;
                        v.x = nx;
                        v.y = ny;
                    }
                    v.heading = Math.atan2(dy, dx) * 180 / Math.PI;
                }
            }
            if (at - this.at > 1000)
                this.at = at - 250;
        }
    }
}
