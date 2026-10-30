import {activityContract,fishingRules,type Fishing,type FishingView} from './contracts';
export const fishingSpawn=(castId=0):Fishing=>({ownerId:'',ownerName:'',phase:'idle',castId,biteAt:0,endsAt:0,catchId:''});
export const nearbyFishing=(spots:FishingView[],p:{x:number;y:number})=>spots.map(s=>({s,d:Math.hypot(p.x-activityContract.fishingSpots.find(v=>v.id===s.id)!.position.x,p.y-activityContract.fishingSpots.find(v=>v.id===s.id)!.position.y)})).filter(v=>v.d<=fishingRules.proximity).sort((a,b)=>Number(Boolean(a.s.fishing.ownerId))-Number(Boolean(b.s.fishing.ownerId))||a.d-b.d)[0]?.s;
/** A four-state cosmetic pool driven by the existing store timer, with no completion. */
export class LocalFishing {
 spots:FishingView[]=activityContract.fishingSpots.map(s=>({id:s.id,generation:1,fishing:fishingSpawn()}));
 constructor(private random=()=>Math.random()){}
 reset(){this.spots=this.spots.map(s=>({...s,generation:s.generation+1,fishing:fishingSpawn(s.fishing.castId)}))}
 cast(id:string,p:{x:number;y:number},at:number){const s=this.spots.find(s=>s.id===id);if(!s||this.spots.some(s=>s.fishing.ownerId)||nearbyFishing([s],p)!==s)return false;const biteAt=at+fishingRules.waitMinMs+Math.floor(this.random()*(fishingRules.waitMaxMs-fishingRules.waitMinMs+1));s.fishing={...fishingSpawn(s.fishing.castId+1),ownerId:'guest',ownerName:'Guest',phase:'waiting',biteAt,endsAt:biteAt+fishingRules.biteMs};s.generation++;return true}
 reel(at:number){this.tick(at);const s=this.spots.find(s=>s.fishing.ownerId==='guest');if(!s||!['waiting','bite'].includes(s.fishing.phase))return false;s.fishing={...s.fishing,phase:'result',endsAt:at+fishingRules.resultMs,catchId:s.fishing.phase==='bite'?fishingRules.catches[Math.floor(this.random()*fishingRules.catches.length)]:''};s.generation++;return true}
 tick(at:number){for(const s of this.spots){const f=s.fishing;if(!f.ownerId)continue;if(at>=f.endsAt){s.fishing=fishingSpawn(f.castId);s.generation++}else if(f.phase==='waiting'&&at>=f.biteAt){s.fishing={...f,phase:'bite'};s.generation++}}}
}
export const fishingCatchLabels:Record<string,string>={fish:'Ikan senja!',boot:'Sepatu nyasar!',can:'Kaleng antik!',leaf:'Daun autumn!'};
