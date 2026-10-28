import React,{useEffect,useRef,useState} from 'react';
import {createRoot} from 'react-dom/client';
import type {Room} from 'livekit-client';
import type {WorldSocket} from '../../net/socket';
import {VoiceSession,mediaError} from './session';
import {updateMicrophone} from './microphone';
import {TABLES} from './zones';
import './voice.css';
interface Participant{id:string;identity:string;userId:string;name:string;table:number;muted:boolean}
interface TableState{nextMessageSeq?:number;admission?:Participant;participants?:Participant[];messages?:{id:string;name:string;text:string}[];hostId?:string;canModerate?:boolean;status?:string}
interface Submission {id:string;text:string;seq:number}
interface ChatContext {socket:WorldSocket|null;admission:string;nextSeq:number;submission:Submission|null;sending:boolean;synced:boolean}
let attached:WorldSocket|null=null;
export function mountVoice(){window.addEventListener('patra-world-attachment',e=>{attached=(e as CustomEvent).detail})}
export function VoicePanel(){
 const [socket,setSocket]=useState(attached),[state,setState]=useState<TableState>({}),[status,setStatus]=useState('Memeriksa layanan voice…'),[ready,setReady]=useState(false),[pending,setPending]=useState(false),[mic,setMic]=useState(false),[deaf,setDeaf]=useState(false),[volume,setVolume]=useState(80),[ptt,setPTT]=useState(false),[device,setDevice]=useState(''),[devices,setDevices]=useState<MediaDeviceInfo[]>([]),[speakers,setSpeakers]=useState<string[]>([]),[draft,setDraft]=useState(''),[chatPending,setChatPending]=useState(false),[retry,setRetry]=useState(false),[chatSynced,setChatSynced]=useState(false);
 const audioRoot=useRef<HTMLDivElement>(null),session=useRef<VoiceSession|null>(null),room=useRef<Room|null>(null),draftRef=useRef(''),chat=useRef<ChatContext>({socket:null,admission:'',nextSeq:1,submission:null,sending:false,synced:false}),generation=useRef(0),deafRef=useRef(false),volumeRef=useRef(.8),busyRef=useRef(false);
 function resetChat(source:WorldSocket|null=null,admission=''){chat.current={socket:source,admission,nextSeq:1,submission:null,sending:false,synced:false};draftRef.current='';setDraft('');setChatPending(false);setRetry(false);setChatSynced(false)}
 function syncChat(source:WorldSocket,next:TableState){const id=next.admission?.id||'';if(chat.current.socket!==source||chat.current.admission!==id)resetChat(source,id);const seq=next.nextMessageSeq;if(id&&typeof seq==='number'&&Number.isSafeInteger(seq)&&seq>0){chat.current.nextSeq=Math.max(chat.current.nextSeq,seq);chat.current.synced=true;setChatSynced(true)}}
 async function sendChat(e:React.FormEvent){
  e.preventDefault();const context=chat.current;if(!context.socket||!context.admission||context.sending||!context.synced)return;
  if(!context.submission){const text=draftRef.current;if(!text.trim()||Array.from(text).length>500){setStatus('Pesan harus 1–500 karakter.');return}context.submission={id:context.admission+':'+context.nextSeq,text,seq:context.nextSeq}}
  const submitted=context.submission;context.sending=true;setChatPending(true);
  const clearMatchingDraft=()=>{if(draftRef.current===submitted.text){draftRef.current='';setDraft('')}};
  try{await context.socket.gameCommand('voice.chat',{clientMessageId:submitted.id,text:submitted.text});if(chat.current!==context)return;context.nextSeq=Math.max(context.nextSeq,submitted.seq+1);context.submission=null;setRetry(false);clearMatchingDraft()}
  catch(error){if(chat.current!==context)return;if((error as {code?:string})?.code==='voice_retry_expired'){
   context.submission=null;context.synced=false;setRetry(false);setChatSynced(false);clearMatchingDraft();setStatus('Pesan sebelumnya sudah diproses. Menyinkronkan meja sebelum pesan baru.');
   try{const next=await context.socket.gameCommand('voice.state',{}) as TableState;if(chat.current!==context)return;syncChat(context.socket,next);setState(next)}catch{if(chat.current===context)setStatus('Pesan sebelumnya sudah diproses. Tunggu sinkronisasi meja sebelum pesan baru.')}
  }else{setRetry(true);setStatus(error instanceof Error?error.message:'Pesan belum terkirim.')}}
  finally{if(chat.current===context){context.sending=false;setChatPending(false)}}
 }
 useEffect(()=>{const change=(e:Event)=>{resetChat();setSocket((e as CustomEvent).detail)};window.addEventListener('patra-world-attachment',change);return()=>window.removeEventListener('patra-world-attachment',change)},[]);
 useEffect(()=>{let live=true;const controller=new AbortController();fetch('/api/v1/voice/status',{signal:controller.signal,cache:'no-store'}).then(async r=>{if(!r.ok)throw Error();return r.json()}).then(v=>{if(!live)return;setReady(v.status==='ready');setStatus(v.status==='ready'?'Berjalan ke area meja, lalu gabung untuk mendengarkan. Mic tetap mati.':'Voice belum tersedia: LiveKit Cloud belum siap. Chat dunia tetap tersedia.')}).catch(()=>{if(live)setStatus('Layanan voice belum tersedia. Chat dunia tetap tersedia.')});return()=>{live=false;controller.abort()}},[]);
 useEffect(()=>{if(!socket)return;let live=true;const version=++generation.current;
 const command=(type:string,payload:Record<string,unknown>)=>socket.gameCommand(type,payload);
 const current=new VoiceSession(command,async()=>{
  const {Room,RoomEvent,Track}=await import('livekit-client');const media=new Room({adaptiveStream:true,dynacast:true});room.current=media;
  media.on(RoomEvent.TrackSubscribed,track=>{if(track.kind!==Track.Kind.Audio)return;const audio=track.attach();audio.volume=deafRef.current?0:volumeRef.current;audioRoot.current?.append(audio)});
  media.on(RoomEvent.TrackUnsubscribed,track=>track.detach().forEach(element=>element.remove()));
  media.on(RoomEvent.ActiveSpeakersChanged,participants=>{if(live)setSpeakers(participants.map(p=>p.identity))});
  media.on(RoomEvent.LocalTrackUnpublished,()=>{if(live)setMic(false)});
  media.on(RoomEvent.Disconnected,()=>{if(live){setMic(false);setStatus('Audio terputus. Chat meja tetap tersedia. Keluar meja lalu gabung lagi untuk mencoba.')}});
  return {connect:async(url,token)=>{await media.connect(url,token,{autoSubscribe:true});await media.startAudio().catch(()=>{if(live)setStatus('Browser menahan audio. Tekan Aktifkan audio untuk mendengarkan.')})},disconnect:async()=>{await media.disconnect(true);media.removeAllListeners();if(room.current===media)room.current=null;audioRoot.current?.replaceChildren()},mic:async(on,deviceId)=>{await media.localParticipant.setMicrophoneEnabled(on,deviceId?{deviceId}:undefined)}};
 });session.current=current;
 const stop=()=>{if(!live)return;setMic(false);resetChat();setState({});void current.leave().catch(()=>{if(live)setStatus('Keluar meja belum dikonfirmasi. Tunggu koneksi dunia pulih.')})};
 const change=(e:Event)=>{const d=(e as CustomEvent).detail;if(d.socket===socket&&d.state!=='online')stop()};
 const event=(e:Event)=>{const d=(e as CustomEvent).detail;if(d.socket===socket&&d.type==='match.started'&&(d.payload?.players?.some((p:{userId:string})=>p.userId===socket.selfId)||d.payload?.spectators?.includes(socket.selfId)))stop()};
 const visibility=()=>{if(document.hidden)stop()};window.addEventListener('patra-world-status',change);window.addEventListener('patra-world-event',event);document.addEventListener('visibilitychange',visibility);
 let polling=false;const timer=setInterval(async()=>{if(polling||busyRef.current||socket.status!=='online'||!live)return;polling=true;try{const next=await command('voice.state',{}) as TableState;if(!live||busyRef.current||generation.current!==version)return;syncChat(socket,next);setState(next);if(next.status){setReady(next.status==='ready');if(next.status==='ready')setStatus(text=>text.startsWith('Voice belum tersedia')||text.startsWith('Layanan voice belum tersedia')?'Berjalan ke area meja, lalu gabung untuk mendengarkan. Mic tetap mati.':text);else setStatus(next.status==='degraded'?'Voice sedang memulihkan isolasi. Chat dunia tetap tersedia.':'Voice belum tersedia: LiveKit Cloud belum siap. Chat dunia tetap tersedia.');if(!next.admission)await current.leave()}if(next.admission?.muted){await current.mic(false);if(live)setMic(false)}}catch{if(live){resetChat();setState({});setMic(false);await current.leave().catch(()=>{})}}finally{polling=false}},1000);
 return()=>{live=false;resetChat();++generation.current;clearInterval(timer);window.removeEventListener('patra-world-status',change);window.removeEventListener('patra-world-event',event);document.removeEventListener('visibilitychange',visibility);if(session.current===current)session.current=null;void current.leave().catch(()=>{});setState({});setMic(false)};
 },[socket]);
 async function microphone(on:boolean){if(deaf&&on)return;await updateMicrophone(session.current,on,device,()=>navigator.mediaDevices.enumerateDevices(),{mic:setMic,devices:setDevices,status:setStatus})}
 useEffect(()=>{if(!ptt)return;const typing=(e:KeyboardEvent)=>{const t=e.target as HTMLElement|null;return Boolean(t?.isContentEditable||t?.closest('input,textarea,select,button'))};const down=(e:KeyboardEvent)=>{if(e.code==='Space'&&!e.repeat&&!typing(e)){e.preventDefault();void microphone(true)}};const up=(e:KeyboardEvent)=>{if(e.code==='Space'){void microphone(false)}};const blur=()=>void microphone(false);window.addEventListener('keydown',down);window.addEventListener('keyup',up);window.addEventListener('blur',blur);return()=>{window.removeEventListener('keydown',down);window.removeEventListener('keyup',up);window.removeEventListener('blur',blur)}},[ptt,device,deaf]);
 async function join(table:number){busyRef.current=true;setPending(true);try{await session.current?.join(table);setStatus('Tersambung sebagai pendengar. Tekan Nyalakan mic jika ingin bicara.');setMic(false)}catch(e){setStatus(e instanceof Error&&e.message?e.message:mediaError(e))}finally{busyRef.current=false;setPending(false)}}
 async function leave(){busyRef.current=true;setPending(true);try{await session.current?.leave();resetChat();setState({});setMic(false);setStatus('Sudah keluar meja.')}catch{setStatus('Keluar belum dikonfirmasi. Coba lagi setelah koneksi pulih.')}finally{busyRef.current=false;setPending(false)}}
 const online=socket?.status==='online';
 return <section className="voice-panel" aria-label="Warkop voice dan chat meja">
  <h3>Dua meja, satu Warkop</h3><p role="status" aria-live="polite">{status}</p>
  {!online&&<p>Masuk dunia privat melalui undangan untuk bergabung. Guest hanya menjelajah solo.</p>}
  <div className="voice-tables">{TABLES.tables.map(t=><article key={t.id}><h4>{t.name}</h4><p>8 kursi · obrolan satu meja</p><button type="button" onClick={()=>socket?.destination(t.x,t.y)} disabled={!online}>Jalan ke meja</button><button type="button" disabled={!online||!ready||pending||Boolean(state.admission)} onClick={()=>void join(t.id)}>Gabung obrolan</button></article>)}</div>
  {state.admission&&<><h4>{TABLES.tables[state.admission.table]?.name} · {state.participants?.length}/8</h4><p>Chat meja sementara · dihapus 5 menit setelah meja kosong.</p>
   <ul>{state.participants?.map(p=><li key={p.id}>{p.name}{p.userId===state.hostId?' · Host':''}{speakers.includes(p.identity)?' · Berbicara':''}{p.muted?' · Dimute moderator':''}{state.canModerate&&p.id!==state.admission?.id&&<><button onClick={()=>void socket?.gameCommand('voice.moderate',{target:p.id,action:'mute'}).catch(e=>setStatus(String(e.message)))}>Mute</button><button onClick={()=>void socket?.gameCommand('voice.moderate',{target:p.id,action:'kick'}).catch(e=>setStatus(String(e.message)))}>Keluarkan</button></>}</li>)}</ul>
   <div className="voice-controls"><button disabled={deaf} onClick={()=>void microphone(!mic)}>{mic?'Matikan mic':'Nyalakan mic'}</button><button onClick={()=>void room.current?.startAudio().catch(e=>setStatus(mediaError(e)))}>Aktifkan audio</button><button onClick={()=>{const value=!deaf;deafRef.current=value;setDeaf(value);audioRoot.current?.querySelectorAll('audio').forEach(a=>a.volume=value?0:volume/100);if(value)void microphone(false)}}>{deaf?'Dengarkan lagi':'Deafen · suara & mic mati'}</button>
   <label>Volume<input type="range" min="0" max="100" value={volume} onChange={e=>{const value=Number(e.target.value);setVolume(value);volumeRef.current=value/100;audioRoot.current?.querySelectorAll('audio').forEach(a=>a.volume=deaf?0:value/100)}}/></label>
   <label>Input microphone<select value={device} onChange={e=>{setDevice(e.target.value);if(mic)void room.current?.switchActiveDevice('audioinput',e.target.value).catch(error=>setStatus(mediaError(error)))}}><option value="">Default perangkat</option>{devices.map(d=><option key={d.deviceId} value={d.deviceId}>{d.label||'Microphone'}</option>)}</select></label>
   <label><input type="checkbox" checked={ptt} onChange={e=>{setPTT(e.target.checked);void microphone(false)}}/> Push-to-talk · tahan spasi ketika tidak mengetik</label>
   {ptt&&<button disabled={deaf} onPointerDown={e=>{e.currentTarget.setPointerCapture(e.pointerId);void microphone(true)}} onPointerUp={()=>void microphone(false)} onPointerCancel={()=>void microphone(false)}>Tahan untuk bicara</button>}
   <button disabled={pending} onClick={()=>void leave()}>Keluar meja</button></div>
   <div className="table-messages" role="log" aria-label="Chat meja">{state.messages?.map(m=><p key={m.id}><strong>{m.name}: </strong>{m.text}</p>)}</div>
   <form onSubmit={sendChat}><label>Pesan meja<textarea value={draft} onChange={e=>{draftRef.current=e.target.value;setDraft(e.target.value)}} rows={2}/></label><button disabled={chatPending||!chatSynced}>{retry?'Coba lagi pesan sebelumnya':'Kirim ke meja'}</button>{retry&&<p>Pesan sebelumnya belum dikonfirmasi. Coba lagi memakai pesan yang sama; draf baru tetap disimpan.</p>}</form>
  </>}
  <p className="voice-note">Microphone hanya diminta setelah tombol mic ditekan. Menutup panel, tab tersembunyi, masuk game, atau keluar area meja akan melepas voice. Tanpa video, rekaman, atau transkripsi.</p><div ref={audioRoot}/>
 </section>
}
export function mountVoicePanel(element:HTMLElement){const root=createRoot(element);root.render(<VoicePanel/>);return()=>root.unmount()}
