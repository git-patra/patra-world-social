import type { Point } from './map';
export const clampZoom = (zoom: number) => Number.isFinite(zoom) ? Math.max(.35, Math.min(1.6, zoom)) : .85;
export function followCamera(position: Point, width: number, height: number, scale: number) { return { scale: clampZoom(scale), x: width / 2 - (position.x - position.y) * 30 * clampZoom(scale), y: height / 2 - (position.x + position.y) * 15 * clampZoom(scale) }; }
