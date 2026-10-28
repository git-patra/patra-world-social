import React,{useEffect,useRef,useState} from 'react';
import type {WorldSocket} from '../../net/socket';
import {track,type Controls,type RaceSnapshot} from './local';
import {guestRace} from './session';
import './race.css';
const colors=['#e7a74a','#8caf82','#bc8dcb','#73afc2','#dd7971'];
const secs=(ms?:number)=>ms===undefined?'—':`${(ms/1000).toFixed(2)}s`;
function TrackView({snapshot,selfId}:{snapshot:RaceSnapshot;selfId:string}){
 const svg=useRef<SVGSVGElement>(null),latest=useRef(snapshot),previous=useRef(snapshot),received=useRef(performance.now());
 useEffect(()=>{previous.current=latest.current;latest.current=snapshot;received.current=performance.now()},[snapshot]);
 useEffect(()=>{let id=0;function render(at:number){const t=document.body.classList.contains('reduced-motion')?1:Math.min(1,(at-received.current)/67),s=latest.current;for(const k of s.karts){const old=previous.current.karts.find(p=>p.actorId===k.actorId)??k,node=svg.current?.querySelector(`[data-slot="${k.startSlot}"]`);if(node){const delta=(k.heading-old.heading+540)%360-180;node.setAttribute('transform',`translate(${old.x+(k.x-old.x)*t} ${old.y+(k.y-old.y)*t}) rotate(${old.heading+delta*t})`)}}id=requestAnimationFrame(render)}id=requestAnimationFrame(render);return()=>cancelAnimationFrame(id)},[]);
 return <svg ref={svg} className="race-track" viewBox="0 0 40960 28672" role="img" aria-label="Lintasan Autumn Ring dan minimap posisi lima kart">
 <defs><pattern id="race-grass" width="1500" height="1500" patternUnits="userSpaceOnUse"><rect width="1500" height="1500" fill="#56634c"/><path d="M200 400l150 -170m700 800l100 -130" stroke="#6b7453" strokeWidth="55"/></pattern><pattern id="race-checker" width="900" height="900" patternUnits="userSpaceOnUse"><rect width="900" height="900" fill="#f0e3c7"/><path d="M0 0h450v450H0zM450 450h450v450H450z" fill="#3a3739"/></pattern></defs>
 <rect width="40960" height="28672" rx="1500" fill="url(#race-grass)"/>
 <rect x={track.outer.x} y={track.outer.y} width={track.outer.width} height={track.outer.height} rx="1300" fill="#423f40" stroke="#e6b97b" strokeWidth="200"/>
 <rect x={track.inner.x} y={track.inner.y} width={track.inner.width} height={track.inner.height} rx="1000" fill="#82744f" stroke="#e6b97b" strokeWidth="200"/>
 <path d="M20480 6144H34816V22528H6144V6144Z" fill="none" stroke="#e7d2a5" strokeWidth="100" strokeDasharray="700 900" opacity=".35"/>
 <rect x="20255" y="2250" width="900" height="7800" fill="url(#race-checker)"/>
 {track.gates.slice(0,3).map((g,i)=><g key={i} opacity=".65"><line x1={g.axis==='x'?g.at:g.min} y1={g.axis==='x'?g.min:g.at} x2={g.axis==='x'?g.at:g.max} y2={g.axis==='x'?g.max:g.at} stroke="#83b5aa" strokeWidth="130" strokeDasharray="250 220"/><text x={g.axis==='x'?g.at:g.max-1200} y={g.axis==='x'?g.max-500:g.at-700} fill="#f7e5bb" fontSize="850">{i+1}</text></g>)}
 <text x="20480" y="13700" textAnchor="middle" fill="#f7dfae" fontSize="1550" fontWeight="bold">AUTUMN RING</text><text x="20480" y="15700" textAnchor="middle" fill="#ebd1a2" fontSize="900">5 LAP · RACE PARK</text>
 {[{x:12200,y:12000},{x:28600,y:16500}].map((p,i)=><g key={i} transform={`translate(${p.x} ${p.y})`}><rect x="-130" y="0" width="260" height="1100" fill="#694b3a"/><circle r="1050" fill={i?'#a95a36':'#c68a3e'}/><circle cx="-450" cy="-250" r="650" fill="#df9c42"/></g>)}
 {snapshot.karts.map(k=><g key={k.actorId} data-slot={k.startSlot} transform={`translate(${k.x} ${k.y}) rotate(${k.heading})`}><rect x="-530" y="-340" width="1050" height="680" rx="200" fill={colors[k.startSlot]} stroke={k.userId===selfId?'#fff5d0':'#27272c'} strokeWidth={k.userId===selfId?130:70}/><rect x="-80" y="-240" width="330" height="480" rx="100" fill="#332e34"/><path d="M430 -190l270 190-270 190" fill="#ffe4a9"/><text transform={`rotate(${-k.heading})`} y="-900" textAnchor="middle" fill="#fff3dc" fontSize="700" fontWeight="bold">{k.position}{k.actorType==='bot'?' BOT':''}</text></g>)}
 </svg>
}
export function RaceView({snapshot,selfId,online,now,saved,disabled,controls,message,onControl,onReturn}:{snapshot:RaceSnapshot;selfId:string;online:boolean;now:number;saved:boolean;disabled:boolean;controls:Controls;message:string;onControl:(key:keyof Controls,v:number)=>void;onReturn:()=>void}){
 const me=snapshot.karts.find(k=>k.userId===selfId),done=snapshot.phase==='finished',racing=snapshot.phase==='racing',count=Math.max(0,Math.ceil((snapshot.startsAt-now)/1000));
 function pedal(label:string,key:keyof Controls,value:number){return <button type="button" aria-label={label} aria-pressed={controls[key]===value} disabled={disabled||!me||done} onPointerDown={e=>{e.preventDefault();e.currentTarget.setPointerCapture(e.pointerId);onControl(key,value)}} onPointerUp={()=>onControl(key,0)} onPointerCancel={()=>onControl(key,0)} onLostPointerCapture={()=>onControl(key,0)}>{label}</button>}
 return <section className="race-game" aria-label="Balap lima lap"><header><div><span>{online?'DUNIA PRIVAT':'PERANGKAT INI · GUEST + 4 BOT'}</span><h3>Autumn Ring</h3></div><strong>{done?'FINISH':!racing?count||'GO!':me?`LAP ${Math.min(5,me.laps+1)}/5`:'PENONTON'}</strong></header>
 <div className="race-metrics"><span>{me?`Posisi ${me.position}/5`:'Menonton · tidak mengendalikan kart'}</span><span>{secs(Math.max(0,(done?snapshot.finishedAt??snapshot.serverTimeMs:Math.min(now,snapshot.deadlineAt))-snapshot.startsAt))}</span><span>{Math.max(0,Math.ceil((snapshot.deadlineAt-now)/1000))}s tersisa</span></div>
 <TrackView snapshot={snapshot} selfId={selfId}/>
 {me&&!done&&<><p>↑ / W gas · ↓ / S rem · ← → / A D belok. Ikuti checkpoint 1 → 2 → 3 → finish. Kart tidak saling bertabrakan.</p><div className="race-controls">{pedal('← Kiri','steer',-1)}{pedal('Kanan →','steer',1)}{pedal('Rem','brake',1)}{pedal('Gas ↑','throttle',1)}</div></>}
 <p role="status">{message||(!racing&&!done?'Semua kart mulai bersamaan setelah hitung mundur.':done?online?(saved?'Hasil tersimpan. Arena kembali terbuka setelah 10 detik atau semua manusia kembali.':'Hasil sementara · menunggu penyimpanan server.'):'Hasil practice lokal. Tidak dikirim ke leaderboard resmi.':me?'Lewati semua checkpoint. Menutup panel melepas kontrol; balapan tetap berjalan.':'Penonton tidak menggantikan BOT di tengah balapan.')}</p>
 <ol className="race-standings" aria-label="Posisi balapan">{[...snapshot.karts].sort((a,b)=>a.position-b.position).map(k=><li key={k.actorId}><b style={{color:colors[k.startSlot]}}>{k.position}. {k.displayName}</b><span>{k.actorType==='bot'?'BOT · ':''}{k.botAssisted?'BOT assistance · ':''}{k.dnf?'DNF':k.finished?secs(k.elapsedMs):`${k.laps}/5 lap`} · Best {secs(k.bestLapMs)} · Grid {k.startSlot+1}</span></li>)}</ol>
 {done&&<><p>Seri waktu simulasi diurutkan menurut slot start tersimpan. DNF diurutkan berdasarkan progres terakhir.</p><button type="button" disabled={disabled||online&&(!saved||!me)} onClick={onReturn}>Kembali ke Race Park</button></>}
 </section>
}
export function RaceGame({socket,lobby}:{socket?:WorldSocket|null;lobby?:{matchId?:string;state:string}}){
 const [snapshot,setSnapshot]=useState<RaceSnapshot|null>(null),[now,setNow]=useState(0),[message,setMessage]=useState(''),[controls,setControls]=useState<Controls>({throttle:0,brake:0,steer:0});const held=useRef<Controls>({throttle:0,brake:0,steer:0}),current=useRef<RaceSnapshot|null>(null),match=lobby?.matchId;
 const neutral=()=>{held.current={throttle:0,brake:0,steer:0};setControls({...held.current})};
 useEffect(()=>{setSnapshot(null);setMessage('');current.current=null;held.current={throttle:0,brake:0,steer:0};if(!socket){const s=guestRace.snapshot(performance.now());setSnapshot(s);current.current=s;return}if(!match)return;let live=true;const load=()=>{if(socket.status==='online')void socket.gameCommand('race.snapshot',{matchId:match}).then(s=>{if(live){setSnapshot(s as RaceSnapshot);current.current=s as RaceSnapshot}}).catch(e=>{if(live)setMessage(e.message)})};const event=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.socket!==socket)return;if(d.type==='world.snapshot')load();if((d.type==='race.snapshot'||d.type==='race.finished')&&d.payload.matchId===match){setSnapshot(d.payload);current.current=d.payload}};window.addEventListener('patra-world-event',event);load();return()=>{live=false;window.removeEventListener('patra-world-event',event)}},[socket,match]);
 useEffect(() => {
  let alive = true;
  let inFlight = false;
  const update = () => {
   const at = socket ? socket.serverNow() : performance.now();
   setNow(at);
   const snapshot = current.current;
   const input = document.hidden ? {throttle: 0, brake: 0, steer: 0} : held.current;
   if (!socket) {
    guestRace.input(input, at);
    const next = guestRace.snapshot(at);
    setSnapshot(next);
    current.current = next;
   } else if (
    snapshot?.phase === 'racing' &&
    snapshot.karts.some(k => k.userId === socket.selfId && !k.finished && !k.dnf) &&
    socket.status === 'online' && !inFlight
   ) {
    inFlight = true;
    void socket.gameCommand('race.input', {matchId: snapshot.matchId, ...input})
     .catch(() => {
      if (alive) setMessage('Kontrol belum diterima. Menunggu sinkronisasi server.');
     })
     .finally(() => { inFlight = false; });
   }
  };
  const timer = setInterval(update, 50);
  return () => {
   alive = false;
   clearInterval(timer);
   held.current = {throttle: 0, brake: 0, steer: 0};
  };
 }, [socket]);
 useEffect(()=>{const pressed=new Set<string>(),keys=['ArrowUp','ArrowDown','ArrowLeft','ArrowRight','w','a','s','d'];function key(e:KeyboardEvent){if(document.body.dataset.controls==='touch')return;if(!current.current||current.current.phase==='finished'||!keys.includes(e.key)||e.target instanceof HTMLInputElement||e.target instanceof HTMLTextAreaElement)return;e.preventDefault();e.stopImmediatePropagation();if(e.type==='keydown')pressed.add(e.key);else pressed.delete(e.key);held.current={throttle:Number(pressed.has('w')||pressed.has('ArrowUp')),brake:Number(pressed.has('s')||pressed.has('ArrowDown')),steer:Number(pressed.has('d')||pressed.has('ArrowRight'))-Number(pressed.has('a')||pressed.has('ArrowLeft'))};setControls({...held.current})}const release=()=>{pressed.clear();neutral()};window.addEventListener('keydown',key,true);window.addEventListener('keyup',key,true);window.addEventListener('blur',release);document.addEventListener('visibilitychange',release);return()=>{window.removeEventListener('keydown',key,true);window.removeEventListener('keyup',key,true);window.removeEventListener('blur',release);document.removeEventListener('visibilitychange',release);held.current={throttle:0,brake:0,steer:0}}},[]);
 function start(){neutral();const s=guestRace.start(performance.now());current.current=s;setSnapshot(s);setMessage('')}
 async function back(){neutral();if(!socket){guestRace.clear();current.current=null;setSnapshot(null);return}if(snapshot)try{const r=await socket.gameCommand('match.return',{matchId:snapshot.matchId}) as {waiting?:boolean};setMessage(r.waiting?'Menunggu manusia lain kembali, atau batas hasil 10 detik.':'Kembali ke pintu Race Park.')}catch(e){setMessage((e as Error).message)}}
 if(!snapshot)return <section className="race-game" aria-label="Balap lima lap"><h3>Autumn Ring · 5 lap</h3>{socket?<p>{match?'Menyinkronkan balapan…':'Gabung Race Park, Ready, lalu Mulai. Slot kosong diisi BOT.'}</p>:<><p>Balapan lokal melawan empat BOT. Kontrol keyboard dan touch tersedia. Hasil lokal tidak masuk leaderboard resmi.</p><button type="button" onClick={start}>Mulai race solo · 5 lap</button></>}</section>;
 return <RaceView snapshot={snapshot} selfId={socket?.selfId??'guest'} online={Boolean(socket)} now={now} saved={lobby?.state==='results'} disabled={Boolean(socket&&socket.status!=='online')} controls={controls} message={message} onControl={(key,value)=>{held.current={...held.current,[key]:value};setControls({...held.current})}} onReturn={()=>void back()}/>;
}
