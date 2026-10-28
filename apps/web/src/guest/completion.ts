import {guestHistory,type GameType,type LocalResult} from './history';
import map from '../../../../contracts/map.v1.json';
import rules from '../../../../contracts/rules.v1.json';
/** No transport dependency: this completion receipt can only write device storage. */
export class Completion{
 private pending=false;private saved=false;private createdAt=Date.now();
 constructor(private id:string,private game:GameType,private write=(r:LocalResult)=>guestHistory.append(r)){}
 finish(durationMs:number,summary:string,detail:unknown){if(this.pending||this.saved)return;this.pending=true;void this.write({id:this.id,gameType:this.game,finishedAt:this.createdAt+Math.max(0,durationMs),durationMs,scope:'device',version:{mapId:map.id,mapVersion:1,rulesId:this.game==='duck'?'duck-v1':rules.id,rulesVersion:1,protocolVersion:1},summary,detail:structuredClone(detail)}).then(()=>{this.saved=true;if(typeof window!=='undefined')window.dispatchEvent(new Event('patra-guest-history'))}).catch(()=>{if(typeof window!=='undefined')window.dispatchEvent(new CustomEvent('patra-guest-storage-error',{detail:'Hasil belum tersimpan di perangkat. Buka game lagi untuk mencoba ulang.'}))}).finally(()=>{this.pending=false})}
}
