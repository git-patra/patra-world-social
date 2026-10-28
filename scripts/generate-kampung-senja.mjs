import {readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
const root=fileURLToPath(new URL('../',import.meta.url));
const read=name=>JSON.parse(readFileSync(root+name,'utf8'));
const write=(name,data)=>writeFileSync(root+name,JSON.stringify(data,null,2)+'\n');
const layout=read('contracts/kampung-senja-layout.v1.json');
const oldChecksum=read('contracts/map.v1.json').checksum;
const obstacles=[];
for(let y=0;y<64;y++)for(let x=0;x<64;x++){
 const px=(x+.5)*40,py=(y+.5)*40-320;
 // A blocked cell covers the complete fixture footprint, with room for avatar feet.
 if(px<40||px>2520||py<40||py>1880||layout.solids.some(s=>px+20>s.x&&px-20<s.x+s.w&&py+20>s.y&&py-20<s.y+s.h))obstacles.push({x,y});
}
const fixtures={warkop:[16,17],duck:[46,18],arcade:[41,18],race:[52,36],penalty:[28,40],garden:[14,42],river:[53,29],plaza:[32,29]};
const map={v:1,id:'kampung-senja-v1',width:64,height:64,spawn:{x:32,y:32},buildings:layout.destinations.map(d=>({id:d.id,x:fixtures[d.id][0],y:fixtures[d.id][1],width:1,height:1,entrance:{x:d.x,y:d.y}})),districts:[{id:'warkop',x:7,y:15,width:17,height:13},{id:'central',x:25,y:26,width:13,height:13},{id:'duck',x:39,y:16,width:13,height:12},{id:'garden',x:7,y:36,width:14,height:13},{id:'penalty',x:24,y:40,width:9,height:12},{id:'race',x:38,y:37,width:15,height:11},{id:'river',x:54,y:27,width:8,height:9}],obstacles};
const blocked=new Set([...obstacles,...map.buildings].map(p=>`${p.x},${p.y}`));
const queue=[map.spawn],seen=new Set([`${map.spawn.x},${map.spawn.y}`]);
for(let i=0;i<queue.length;i++)for(const [dx,dy]of[[1,0],[-1,0],[0,1],[0,-1]]){const n={x:queue[i].x+dx,y:queue[i].y+dy},k=`${n.x},${n.y}`;if(n.x<0||n.x>63||n.y<0||n.y>63||seen.has(k)||blocked.has(k))continue;seen.add(k);queue.push(n)}
for(const d of [...layout.destinations,...layout.voiceTables])if(!seen.has(`${d.x},${d.y}`)||blocked.has(`${d.x},${d.y}`))throw Error(`Unreachable ${d.id} at ${d.x},${d.y}`);
const canonical=v=>Array.isArray(v)?v.map(canonical):v&&typeof v==='object'?Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])])):v;
map.checksum=createHash('sha256').update(JSON.stringify(canonical(map))).digest('hex');
const tables={v:1,id:'kampung-senja-tables-v1',mapId:map.id,mapVersion:map.v,mapChecksum:map.checksum,tables:layout.voiceTables};
for(const name of ['contracts/map.v1.json','apps/server/internal/world/assets/map.v1.json'])write(name,map);
for(const name of ['contracts/voice-tables.v1.json','apps/server/internal/voice/tables.v1.json'])write(name,tables);
const snapshots=read('contracts/snapshots.golden.json');
for(const s of [...snapshots.valid,...snapshots.invalid]){if(s.mapId==='autumn-village'||s.mapId==='kampung-senja-v1')s.mapId=map.id;if([oldChecksum,'cb56899197324a5017021b345d60acc9a38da9b4ad78d14cace30130e29c99b2'].includes(s.mapChecksum))s.mapChecksum=map.checksum}
write('contracts/snapshots.golden.json',snapshots);
console.log(`Generated ${map.id}, ${seen.size} reachable cells; checksum ${map.checksum}`);
