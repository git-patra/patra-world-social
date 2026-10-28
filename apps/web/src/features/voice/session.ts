export interface MediaRoom {
 connect:(url:string,token:string)=>Promise<void>;
 disconnect:()=>Promise<void>;
 mic:(enabled:boolean,device?:string)=>Promise<void>;
}
type Command=(type:string,payload:Record<string,unknown>)=>Promise<unknown>;
/** Serial lifecycle: delayed join/token/connect results cannot resurrect cancelled audio. */
export class VoiceSession {
 private generation=0;
 private micGeneration=0;
 private pending:Promise<void>=Promise.resolve();
 private microphone:Promise<boolean>=Promise.resolve(false);
 private room:MediaRoom|null=null;
 private admitted=false;
 constructor(private command:Command,private createRoom:()=>Promise<MediaRoom>){}
 join(table:number):Promise<void>{
  const generation=++this.generation;
  const run=async()=>{
   await this.release();
   if(generation!==this.generation)return;
   let confirmed=false,mediaStarted=false;
   try{
    // A timeout can hide an allocated seat. Release must still contact Go.
    this.admitted=true;
    await this.command('voice.join',{table});confirmed=true;
    if(generation!==this.generation){await this.release();return}
    const token=await this.command('voice.token',{}) as {url:string;token:string};
    if(generation!==this.generation){await this.release();return}
    mediaStarted=true;
    this.room=await this.createRoom();
    if(generation!==this.generation){await this.release();return}
    await this.room.connect(token.url,token.token);
    if(generation!==this.generation)await this.release();
   }catch(error){
    if(!confirmed||generation!==this.generation)await this.release().catch(()=>{});
    else{const room=this.room;this.room=null;await room?.disconnect().catch(()=>{})}
    // A confirmed table admission survives an audio outage so chat still works.
    throw mediaStarted?new Error(mediaError(error)):error;
   }
  };
  this.pending=this.pending.catch(()=>{}).then(run);return this.pending;
 }
 leave():Promise<void>{
  ++this.generation;++this.micGeneration;
  const room=this.room;this.room=null;
  const stop=room?.disconnect()??Promise.resolve();
  this.pending=this.pending.catch(()=>{}).then(async()=>{await stop;await this.release()});return this.pending;
 }
 private async release(){
  ++this.micGeneration;
  const room=this.room;this.room=null;await room?.disconnect();
  if(this.admitted){await this.command('voice.leave',{});this.admitted=false}
 }
 mic(on:boolean,device?:string):Promise<boolean>{
  const version=++this.micGeneration,room=this.room;
  const run=async()=>{
   if(!room||room!==this.room||version!==this.micGeneration)return false;
   if(on)await this.command('voice.unmute',{});
   if(room!==this.room||version!==this.micGeneration)return false;
   await room.mic(on,device);
   if(room!==this.room||version!==this.micGeneration){await room.mic(false).catch(()=>{});return false}
   return on;
  };
  this.microphone=this.microphone.catch(()=>false).then(run);return this.microphone;
 }
}
export function mediaError(error:unknown){
 const name=error&&typeof error==='object'&&'name'in error?String(error.name):'';
 if(name==='NotAllowedError'||name==='SecurityError')return 'Izin microphone ditolak. Chat meja dan mode dengar tetap tersedia.';
 if(name==='NotFoundError'||name==='DevicesNotFoundError')return 'Perangkat microphone tidak ditemukan. Pilih perangkat lain atau tetap mendengarkan.';
 if(name==='NotReadableError'||name==='TrackStartError')return 'Perangkat microphone sedang dipakai atau tidak dapat dibuka.';
 return 'Koneksi audio gagal. Periksa jaringan dan coba gabung lagi.';
}
