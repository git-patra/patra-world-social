import layout from '../../../../contracts/kampung-senja-layout.v1.json';
import type {Point} from './map';
export const VILLAGE=layout;
export const VILLAGE_DESTINATIONS=layout.destinations;
export const projectVillage=({x,y}:Point):Point=>({x:(x+.5)*40,y:(y+.5)*40-320});
export const unprojectVillage=({x,y}:Point):Point=>({x:x/40-.5,y:(y+320)/40-.5});
export function villageLocation(position:Point){
 return VILLAGE_DESTINATIONS.filter(d=>d.id!=='arcade').map(d=>({...d,distance:Math.hypot(position.x-d.x,position.y-d.y)})).filter(d=>d.distance<8).sort((a,b)=>a.distance-b.distance)[0]??{id:'paths',name:'Jalan Kampung'};
}
