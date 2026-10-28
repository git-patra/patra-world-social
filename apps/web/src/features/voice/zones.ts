import tables from '../../../../../contracts/voice-tables.v1.json';
export const TABLES=tables;
export function InTable(id:number,x:number,y:number){const t=TABLES.tables.find(t=>t.id===id);return Boolean(t&&Math.hypot(x-t.x,y-t.y)<=t.radius)}
