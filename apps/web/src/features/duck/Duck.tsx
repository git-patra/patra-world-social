import React,{useEffect,useRef,useState} from 'react';
import type {WorldSocket} from '../../net/socket';
import {type Aim,type DuckSnapshot} from './local';
import {guestDuck} from './session';
import './duck.css';
import {OnlineDuckController,presentedTargets} from './online';
export function DuckView({s,aim,now,online,canShoot,saved,message,onAim,onShoot,onReturn}:{s:DuckSnapshot;aim:Aim;now:number;online:boolean;canShoot:boolean;saved:boolean;message:string;onAim:(a:Aim)=>void;onShoot:(a:Aim)=>void;onReturn:()=>void}){
 const done=s.phase==='finished',count=Math.max(0,Math.ceil((s.startsAt-now)/1000));
 const pointer=(e:React.PointerEvent<SVGSVGElement>)=>{const r=e.currentTarget.getBoundingClientRect();return{x:Math.max(0,Math.min(1,(e.clientX-r.left)/r.width)),y:Math.max(0,Math.min(1,(e.clientY-r.top)/r.height))}};
 return <section className="duck-game" aria-label="Duck Arcade"><header><div><span>{online?'DUNIA PRIVAT':'SOLO · PERANGKAT INI'}</span><h3>Pumpkin Pond</h3></div><b>{done?'SELESAI':count?`MULAI ${count}`:`${Math.max(0,Math.ceil((s.deadlineAt-now)/1000))}s`}</b></header><div className="duck-metrics"><span>Skor <b>{s.score}</b></span><span>Hit {s.hits}/{s.shots}</span><span>Akurasi {s.shots?(100*s.hits/s.shots).toFixed(1):'0'}%</span></div>
 <svg className="duck-pond" viewBox="0 0 1000 1000" role="img" aria-label="Kolam target bebek" onPointerMove={e=>onAim(pointer(e))} onPointerDown={e=>{e.preventDefault();const a=pointer(e);onAim(a);if(canShoot)onShoot(a)}}><defs><linearGradient id="duck-water" x2="0" y2="1"><stop stopColor="#4d777a"/><stop offset="1" stopColor="#273e4a"/></linearGradient></defs><rect width="1000" height="1000" rx="35" fill="url(#duck-water)"/><path d="M0 780Q170 740 320 795T660 790T1000 780V1000H0" fill="#6d6840"/><path d="M0 90Q200 140 400 90T800 80T1000 90" fill="none" stroke="#f6b858" strokeWidth="12"/>{[120,400,720,910].map((x,i)=><g key={x} transform={`translate(${x} ${850+i%2*30})`}><ellipse rx="45" ry="35" fill="#c77737"/><path d="M0 -35v-15" stroke="#45452e" strokeWidth="12"/><path d="M-20 -5l10 -10 10 10m10 0 10-10 10 10M-20 15q20 15 40 0" fill="none" stroke="#3d3430" strokeWidth="7"/></g>)}
 {(online?presentedTargets(s,now):s.targets).map(t=><g data-target={t.id} data-kind={t.kind} key={t.id} transform={`translate(${t.x*1000} ${t.y*1000})`}><circle r={t.radius*1000} fill={t.kind==='gold'?'#eab94e':t.kind==='fast'?'#d9bdde':'#eee0b9'} stroke="#322c2d" strokeWidth="5"/><path d="M-29 8q13 27 39 9q20-10 5-26q-1-25-22-20q-19 3-9 25" fill={t.kind==='gold'?'#fff0a4':t.kind==='fast'?'#886e9e':'#677b52'}/><path d="M10 -17l22 6-20 7" fill="#ed983f"/><circle cx="3" cy="-18" r="4" fill="#29272b"/></g>)}
 {canShoot&&<g transform={`translate(${aim.x*1000} ${aim.y*1000})`} stroke="#fff4ce" strokeWidth="3" fill="none" pointerEvents="none"><circle r="22"/><path d="M-35 0h22m26 0h22M0-35v22m0 26v22"/></g>}</svg>
 <p>Normal 100 · Cepat 200 · Emas 500. Pointer/touch membidik dan menembak. Panah menggerakkan bidikan, Space menembak.</p><p role="status">{message||(!done&&s.phase==='countdown'?'Bersiap · sesi dimulai setelah countdown 3 detik.':!canShoot&&!done?'Menonton sesi kabinet ini.':online?'Tembakan dan hasil ditentukan server.':'Practice lokal. Skor tidak dikirim ke leaderboard akun.')}</p>
 {done&&<><p>{s.result?.participants[0]?.outcome==='incomplete'?'Sesi tidak lengkap · tidak eligible leaderboard.':online?saved?'Hasil tersimpan.':'Hasil sementara · menunggu penyimpanan server.':'Hasil practice lokal.'}</p><button type="button" disabled={online&&!saved} onClick={onReturn}>Kembali ke Duck Arcade</button></>}
 </section>
}
export function DuckGame({socket,lobby}:{socket?:WorldSocket|null;lobby?:{matchId?:string;state:string}}){
 const[snapshot,setSnapshot]=useState<DuckSnapshot|null>(null),[aim,setAim]=useState<Aim>({x:.5,y:.5}),[now,setNow]=useState(0),[message,setMessage]=useState(''),[timing,setTiming]=useState('Kalibrasi koneksi · tanpa kompensasi latensi.');const current=useRef<DuckSnapshot|null>(null),aimRef=useRef(aim),shot=useRef<(a:Aim)=>void>(()=>{}),match=lobby?.matchId;
 const controller=useRef<OnlineDuckController|null>(null);
 const update=(s:DuckSnapshot|null)=>{current.current=s;setSnapshot(s)};
 useEffect(()=>{controller.current?.reset();controller.current=socket&&match?new OnlineDuckController(socket.worldId,match,(type,payload)=>socket.gameCommand(type,payload)):null;update(null);setMessage('');setTiming('Kalibrasi koneksi · tanpa kompensasi latensi.');if(!socket){update(guestDuck.snapshot(performance.now()));return}if(!match)return;let live=true;const load=()=>{if(socket.status==='online')void socket.gameCommand('duck.snapshot',{matchId:match}).then(v=>{if(live){controller.current?.receive(v as DuckSnapshot);update(v as DuckSnapshot)}}).catch(e=>{if(live)setMessage(e.message)})};const event=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.socket!==socket)return;if(d.type==='world.snapshot'){controller.current?.reset();setTiming('Kalibrasi koneksi · tanpa kompensasi latensi.');load()}if((d.type==='duck.snapshot'||d.type==='duck.finished')&&d.payload.matchId===match){controller.current?.receive(d.payload);update(d.payload)}};window.addEventListener('patra-world-event',event);load();return()=>{live=false;window.removeEventListener('patra-world-event',event)}},[socket,match]);
 useEffect(()=>{
  let alive=true,busy=false,animation=0;
  const calibrate=async()=>{
   const online=controller.current;
   if(!online||!socket||socket.status!=='online'||current.current?.actor.userId!==socket.selfId||current.current?.phase==='finished'||busy)return;
   busy=true;try{await online.calibrate()}catch{if(alive)setTiming('Kalibrasi koneksi terbatas · tembakan tetap memakai waktu server.')}finally{busy=false}
  };
  const render=()=>{
   if(!alive)return;
   const online=controller.current;
   if(socket&&online){const frame=online.frame();setNow(frame.at);if(frame.snapshot)update(frame.snapshot);setTiming(frame.message)}
   animation=requestAnimationFrame(render);
  };
  if(socket)animation=requestAnimationFrame(render);
  const timer=setInterval(()=>{if(!socket){const at=performance.now();setNow(at);update(guestDuck.snapshot(at))}},50);
  const clock=setInterval(()=>void calibrate(),3000);void calibrate();
  return()=>{alive=false;cancelAnimationFrame(animation);clearInterval(timer);clearInterval(clock)};
 },[socket,match]);
 const canShoot=Boolean(snapshot&&snapshot.phase==='playing'&&(!socket||socket.status==='online'&&snapshot.actor.userId===socket.selfId));
 shot.current=(a:Aim)=>{if(!canShoot||!snapshot)return;if(!socket){guestDuck.shoot(a,performance.now());update(guestDuck.snapshot(performance.now()));return}void controller.current?.shoot(a).then(v=>{const r=v as {points:number};setMessage(r.points?`Hit +${r.points}`:'Meleset')}).catch(()=>setMessage('Tembakan ditolak: tunggu 250 ms atau sinkronkan koneksi.'))};
 useEffect(()=>{const key=(e:KeyboardEvent)=>{if(document.body.dataset.controls==='touch')return;if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown',' '].includes(e.key)||e.target instanceof HTMLInputElement||e.target instanceof HTMLTextAreaElement||!current.current||current.current.phase!=='playing')return;e.preventDefault();e.stopImmediatePropagation();if(e.key===' '){if(!e.repeat)shot.current(aimRef.current);return}const a={x:Math.max(0,Math.min(1,aimRef.current.x+(e.key==='ArrowRight'?.025:e.key==='ArrowLeft'?-.025:0))),y:Math.max(0,Math.min(1,aimRef.current.y+(e.key==='ArrowDown'?.025:e.key==='ArrowUp'?-.025:0)))};aimRef.current=a;setAim(a)};window.addEventListener('keydown',key,true);return()=>window.removeEventListener('keydown',key,true)},[]);
 function back(){if(!socket){guestDuck.clear();update(null)}else if(snapshot)void socket.gameCommand('match.return',{matchId:snapshot.matchId}).catch(e=>setMessage(e.message))}
 if(!snapshot)return <section className="duck-game" aria-label="Duck Arcade"><h3>Pumpkin Pond · Duck Arcade</h3>{socket?<p>Gabung salah satu dari empat kabinet, Ready, lalu Mulai. Sesi 60 detik setelah countdown 3 detik.</p>:<><p>Latihan solo lokal · 60 detik. Hasil tidak masuk leaderboard akun.</p><button type="button" onClick={()=>update(guestDuck.start(performance.now()))}>Mulai Duck solo · 60 detik</button></>}</section>;
 return <DuckView s={snapshot} aim={aim} now={now} online={Boolean(socket)} canShoot={canShoot} saved={lobby?.state==='results'} message={[message,socket?timing:''].filter(Boolean).join(' · ')} onAim={a=>{aimRef.current=a;setAim(a)}} onShoot={a=>shot.current(a)} onReturn={back}/>;
}
