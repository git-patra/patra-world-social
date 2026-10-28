import type { Avatar } from './avatars';
interface AvatarOptions {
    motion: boolean;
    elapsed: number;
    walking: boolean;
}
interface ArtHelpers {
    at: (x: number, y: number, draw: () => void) => void;
    ellipse: (x: number, y: number, rx: number, ry: number, fill: string) => void;
    glow: (x: number, y: number, r: number, rgb?: string, intensity?: number) => void;
    rounded: (x: number, y: number, w: number, h: number, r: number, fill: string, stroke?: string) => void;
    line: (points: number[][], color: string, weight?: number) => void;
    polygon: (points: number[][], fill: string, stroke?: string, width?: number) => void;
}
export function drawAvatar(ctx: CanvasRenderingContext2D, avatar: Avatar, options: AvatarOptions, helpers: ArtHelpers) {
    const { motion, elapsed, walking } = options;
    const facing = avatar.facing ?? 1;
    const { at, ellipse, glow, rounded, line, polygon } = helpers;
    at(avatar.x, avatar.y, () => {
        ctx.scale(1.2, 1.2);
        const bounce = motion ? Math.sin(elapsed * (walking ? 17 : 2.5)) * (walking ? 1.6 : .8) : 0;
        ellipse(0, 1, 10, 4, '#211a2d80');
        glow(0, -11, 23, '255,214,147', .13);
        // Small boots and a swinging satchel make the direction of travel readable.
        const step = motion && walking ? Math.sin(elapsed * 17) * 3 : 0;
        rounded(-7, -3 + step * .3, 5, 4, 2, '#805047');
        rounded(2, -3 - step * .3, 5, 4, 2, '#805047');
        ctx.save();
        ctx.translate(0, -3 + bounce);
        ctx.beginPath();
        ctx.moveTo(-10, 0);
        ctx.lineTo(-10, -11);
        ctx.bezierCurveTo(-10, -27, 10, -27, 10, -11);
        ctx.lineTo(10, 0);
        ctx.lineTo(5, -3);
        ctx.lineTo(0, 1);
        ctx.lineTo(-5, -3);
        ctx.closePath();
        ctx.fillStyle = avatar.avatarId==='moon'?'#cfcaef':avatar.avatarId==='autumn'?'#c4d394':'#f9e7ce';
        ctx.fill();
        ctx.strokeStyle = '#be9d87';
        ctx.lineWidth = .9;
        ctx.stroke();
        ellipse(-3 + facing * 1.5, -13, 1.5, 2, '#4b354a');
        ellipse(3 + facing * 1.5, -13, 1.5, 2, '#4b354a');
        ellipse(facing * 1.5, -8, 1.2, .8, '#c9927e');
        line([[-8, -21], [8, -21]], '#da7a42', 3);
        ellipse(0, -23, 7, 3, '#e79955');
        line([[0, -25], [1, -28]], '#b89461', 2);
        line([[-7, -16], [9, -4]], '#c48a62', 1.5);
        rounded(5, -7, 7, 7, 2, '#e59952', '#ad6b42');
        ctx.restore();
        // A small arrow makes the player easy to find among the buildings.
        const arrowY = -39 + (motion ? Math.sin(elapsed * 2) * 2 : 0);
        polygon([[-4, arrowY], [4, arrowY], [0, arrowY + 5]], '#ffce8b');
    });
}
