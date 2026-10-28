import {mediaError,type VoiceSession} from './session';
interface MicrophoneView {mic:(enabled:boolean)=>void;devices:(devices:MediaDeviceInfo[])=>void;status:(message:string)=>void}
/** Keep the rendered mic state tied to publication, not optional device discovery. */
export async function updateMicrophone(session:Pick<VoiceSession,'mic'>|null,on:boolean,device:string,enumerate:()=>Promise<MediaDeviceInfo[]>,view:MicrophoneView){
 try{
  const enabled=Boolean(await session?.mic(on,device));view.mic(enabled);
  if(enabled){view.status('Mic aktif. Matikan kapan saja.');view.devices((await enumerate().catch(()=>[])).filter(d=>d.kind==='audioinput'))}
 }catch(error){view.mic(false);view.status(mediaError(error))}
}
