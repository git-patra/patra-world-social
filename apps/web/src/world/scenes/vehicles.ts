import type { Vehicle } from '../../activities/contracts';
import type { SceneActor } from '../../activities/scene';
import { projectVillage } from '../village';
export function vehicleActors(vehicles: Vehicle[]): SceneActor[] {
    return vehicles.map(v => {
        const p = projectVillage(v), bike = v.kind === 'bicycle';
        return { id: `vehicle:${v.id}`, userId: v.riderId || undefined, ...p, label: v.riderId ? '' : bike ? 'E · Sepeda' : 'E · Mobil', draw(ctx) {
                ctx.save();
                ctx.translate(p.x, p.y);
                ctx.rotate(v.heading * Math.PI / 180);
                ctx.fillStyle = '#253f38';
                if (bike) {
                    ctx.fillRect(-13, -7, 8, 3);
                    ctx.fillRect(5, -7, 8, 3);
                    ctx.strokeStyle = '#e1b567';
                    ctx.lineWidth = 3;
                    ctx.beginPath();
                    ctx.moveTo(-10, -5);
                    ctx.lineTo(1, 4);
                    ctx.lineTo(10, -5);
                    ctx.lineTo(-10, -5);
                    ctx.stroke();
                }
                else {
                    ctx.fillRect(-13, -10, 24, 5);
                    ctx.fillRect(-13, 5, 24, 5);
                    ctx.fillStyle = v.id === 'car-2' ? '#d18f61' : '#b6b880';
                    ctx.beginPath();
                    ctx.roundRect(-16, -7, 32, 14, 4);
                    ctx.fill();
                    ctx.fillStyle = '#46716a';
                    ctx.fillRect(-4, -5, 12, 10);
                }
                if (v.riderId) {
                    ctx.fillStyle = '#f5cf9f';
                    ctx.beginPath();
                    ctx.arc(0, 0, 5, 0, Math.PI * 2);
                    ctx.fill();
                }
                ctx.restore();
            } };
    });
}
