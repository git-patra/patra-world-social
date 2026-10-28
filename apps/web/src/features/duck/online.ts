import type {Aim,DuckSnapshot,PublicTarget} from './local';

export type Calibration={offsetMs:number;rttMs:number;rewindMs:number;measuredAt:number;expiresAt:number};
export type ClockReply={measured:boolean;calibration:Calibration};
type Resolution={targetId:number;points:number};
type Command=(type:string,payload:Record<string,unknown>)=>Promise<unknown>;
export const predictionHorizonMs=500;

// Predict only targets already disclosed by a snapshot. No client trajectory,
// predicted hit, or display timestamp is sent as authoritative game state.
export function presentedTargets(snapshot:DuckSnapshot,at:number):PublicTarget[]{
 const elapsed=at-snapshot.serverTimeMs;
 if(snapshot.phase!=='playing'||elapsed<0||elapsed>predictionHorizonMs)return[];
 return snapshot.targets.filter(t=>Number.isFinite(t.vx)&&Number.isFinite(t.expiresAt)&&at<t.expiresAt!).map(t=>({...t,x:t.x+t.vx!*elapsed/1000}));
}
export class OnlineDuckController{
 private snapshot:DuckSnapshot|null=null;
 private receivedAt=0;
 private calibration:Calibration|null=null;
 private generation=0;
 private dead=new Set<number>();
 constructor(readonly worldId:string,readonly matchId:string,private command:Command,private localNow:()=>number=()=>performance.now()){}
 reset(){this.generation++;this.snapshot=null;this.calibration=null;this.dead.clear();this.receivedAt=0}
 receive(snapshot:DuckSnapshot){
  if(snapshot.worldId!==this.worldId||snapshot.matchId!==this.matchId||this.snapshot&&snapshot.serverTimeMs<this.snapshot.serverTimeMs||this.snapshot?.phase==='finished'&&snapshot.phase!=='finished')return;
  this.snapshot={...snapshot,targets:snapshot.targets.filter(t=>!this.dead.has(t.id))};this.receivedAt=this.localNow();
  if(snapshot.phase==='finished')this.calibration=null;
 }
 async calibrate(){
  const generation=this.generation;
  const challenge=await this.command('duck.clock',{matchId:this.matchId}) as {nonce:string};
  if(generation!==this.generation)return;
  const reply=await this.command('duck.clock_reply',{matchId:this.matchId,nonce:challenge.nonce,clientTimeMs:this.localNow()}) as ClockReply;
  if(generation!==this.generation)return;
  const c=reply.calibration;
  if(!reply.measured||!c||![c.offsetMs,c.rttMs,c.rewindMs,c.measuredAt,c.expiresAt].every(Number.isFinite)||c.rttMs<0||c.rttMs>1000||c.rewindMs!==Math.min(150,Math.floor(c.rttMs/2))||c.expiresAt!==c.measuredAt+15000)throw Error('Invalid measured Duck clock');
  this.calibration={...c};
 }
 frame(localAt=this.localNow()){
  const s=this.snapshot;
  const c=this.calibration;
  const estimated=c?localAt+c.offsetMs:0;
  const calibrated=Boolean(c&&estimated>=c.measuredAt&&estimated<=c.expiresAt);
  // For RTT/2>150 the selected server shot time is slightly ahead of the
  // estimated click time. This presentation lead never changes server rewind.
  const at=calibrated?estimated+Math.max(0,c!.rttMs/2-c!.rewindMs):s?s.serverTimeMs+Math.max(0,localAt-this.receivedAt):0;
  const stale=Boolean(s&&(at-s.serverTimeMs>predictionHorizonMs||at<s.serverTimeMs));
  return{snapshot:s,at,calibrated,stale,message:stale?'Snapshot terlambat · target disembunyikan sampai sinkron.':!calibrated?'Kalibrasi koneksi terbatas · tampilan waktu perkiraan.':c!.rttMs>300?'Koneksi lambat · kompensasi maksimal 150 ms.':'Koneksi terukur · kompensasi maksimal 150 ms.'};
 }
 async shoot(aim:Aim){
  const generation=this.generation;
  const result=await this.command('duck.shoot',{matchId:this.matchId,...aim,clientTimeMs:this.localNow()}) as Resolution;
  if(generation===this.generation&&result.targetId>0){this.dead.add(result.targetId);if(this.snapshot)this.snapshot={...this.snapshot,targets:this.snapshot.targets.filter(t=>t.id!==result.targetId)}};
  return result;
 }
}
