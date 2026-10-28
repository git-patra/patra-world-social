import React, {lazy,Suspense,useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import {WorldSocket} from '../../net/socket';
import './lobby.css';
const PenaltyDuel=lazy(()=>import('../penalty/Duel').then(m=>({default:m.PenaltyDuel})));
const RaceGame=lazy(()=>import('../race/Race').then(m=>({default:m.RaceGame})));
const DuckGame=lazy(()=>import('../duck/Duck').then(m=>({default:m.DuckGame})));

interface Player {actorId:string;userId:string;displayName:string;ready:boolean;connected:boolean;botAssisted?:boolean}
interface Lobby {id:string;arenaId:string;worldId:string;hostId:string;state:string;gameType:string;capacity:number;engineAvailable:boolean;botChosen:boolean;players:Player[];spectators:string[];matchId?:string;startsAt?:number}
interface Props {lobbies:Lobby[];arenaId:string;selfId:string;online:boolean;pending:boolean;message:string;act:(type:string,payload:Record<string,unknown>)=>void}
const names:Record<string,string>={race:'Race Park',penalty:'Lapangan Penalti','duck-1':'Kabinet 1','duck-2':'Kabinet 2','duck-3':'Kabinet 3','duck-4':'Kabinet 4'};
export function LobbyView({lobbies,arenaId,selfId,online,pending,message,act}:Props){
 const choices=arenaId==='duck'?['duck-1','duck-2','duck-3','duck-4']:[arenaId];
 return <section className="lobby-panel" aria-label="Lobby permainan online">
  <h3>Lobby dunia privat</h3>
  {!online&&<p>Sambungkan ke dunia privat untuk ikut lobby.</p>}
  <p role="status">{message}</p>
  {choices.map(id=>{const l=lobbies.find(l=>l.arenaId===id);
   // Arena definitions expose their stable registry id separately from lobby UUID.
   const mine=l?.players.find(p=>p.userId===selfId),host=l?.hostId===selfId;
   const active=Boolean(l&&l.state!=='idle'&&l.state!=='lobby');
   const canStart=Boolean(online&&!pending&&host&&l?.engineAvailable&&l.state==='lobby'&&l.players.length&&l.players.every(p=>p.ready&&p.connected)&&(l.gameType!=='penalty'||l.players.length===2||l.botChosen));
   return <article key={id}><h4>{names[id]}</h4>
    <p>{l?`${l.players.length}/${l.capacity} pemain · ${l.state}`:online?'Memuat ketersediaan…':'Offline · masuk dunia privat untuk melihat lobby'}</p>
    {l&&!l.engineAvailable&&<p>Mesin permainan belum tersedia. Lobby dapat dipakai; pertandingan belum dapat dimulai.</p>}
    {l?.players.map(p=><p key={p.actorId}>{p.displayName}{p.userId===l.hostId?' · Host':''} · {p.connected?(p.ready?'Ready':'Belum ready'):'Terputus · kursi 15 detik'}{p.botAssisted?' · Bot assistance':''}</p>)}
    {l?.gameType==='race'&&<p>5 lap · 5 peserta termasuk bot. Slot kosong diisi BOT saat Start.</p>}
    {online&&l&&!mine&&<button type="button" disabled={pending||(!active&&l.players.length>=l.capacity)} onClick={()=>act('lobby.join',{arenaId:id})}>{active?'Tonton':'Ikut '+(id==='race'?'balapan':'permainan')}</button>}
    {online&&mine&&<><button type="button" disabled={pending||active} onClick={()=>act('lobby.ready',{lobbyId:l.id,ready:!mine.ready})}>{mine.ready?'Batalkan ready':'Ready'}</button>
     {host&&l.gameType==='penalty'&&!active&&<label><input type="checkbox" checked={l.botChosen} disabled={pending} onChange={e=>act('lobby.bot',{lobbyId:l.id,enabled:e.target.checked})}/> Main lawan BOT bila sendiri</label>}
     <button type="button" disabled={!canStart} onClick={()=>act('match.start',{lobbyId:l.id})}>Mulai</button></>}
    {online&&l&&(mine||l.spectators.includes(selfId))&&<button type="button" disabled={pending} onClick={()=>l.state==='results'&&l.matchId?act('match.return',{matchId:l.matchId}):act('lobby.leave',{})}>{l.state==='results'?'Kembali ke pintu arena':l.gameType==='penalty'&&active?'Keluar duel · forfeit':'Keluar lobby / kembali ke pintu arena'}</button>}
    {Boolean(l?.spectators.length)&&<p>{l.spectators.length} penonton · tidak otomatis masuk match berikutnya.</p>}
   </article>;
  })}
 </section>;
}
let attached:WorldSocket|null=null;
export function mountLobbies(){window.addEventListener('patra-world-attachment',e=>{attached=(e as CustomEvent).detail;});}
function LobbyPanel({arenaId}:{arenaId:string}){
 const [socket,setSocket]=useState(attached),[online,setOnline]=useState(attached?.status==='online'),[lobbies,setLobbies]=useState([] as Lobby[]),[message,setMessage]=useState(''),[pending,setPending]=useState(false);
 useEffect(()=>{const change=(e:Event)=>{setSocket((e as CustomEvent).detail);setLobbies([]);setMessage('');setPending(false)};window.addEventListener('patra-world-attachment',change);return()=>window.removeEventListener('patra-world-attachment',change)},[]);
 useEffect(()=>{let live=true;setOnline(socket?.status==='online');setLobbies([]);
  function accept(payload:Record<string,unknown>){if(live&&socket&&payload.worldId===socket.worldId&&Array.isArray(payload.lobbies)){setLobbies(payload.lobbies as Lobby[])}}
  const load=()=>{if(socket?.status==='online')void socket.gameCommand('lobby.list',{}).then(v=>accept(v as Record<string,unknown>)).catch(e=>{if(live)setMessage(e.message)})};
  const event=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.socket!==socket)return;if(d.type==='lobby.updated')accept(d.payload);if(d.type==='match.result_status'){const labels:Record<string,string>={finalizing:'Hasil belum tersimpan',unconfirmed:'Hasil belum dapat dikonfirmasi',completed:'Hasil tersimpan. Buka History untuk detail.',aborted_storage:'Tidak ada hasil tersimpan setelah rekonsiliasi.',unavailable:'Hasil belum tersedia dari mesin permainan.'};setMessage(labels[d.payload.status]??'')}};
  let wasOnline=socket?.status==='online';
  const status=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.socket!==socket)return;const nextOnline=d.state==='online';setOnline(nextOnline);if(nextOnline&&!wasOnline)load();wasOnline=nextOnline};
  window.addEventListener('patra-world-event',event);window.addEventListener('patra-world-status',status);load();return()=>{live=false;window.removeEventListener('patra-world-event',event);window.removeEventListener('patra-world-status',status)};
 },[socket]);
 const act=async(type:string,payload:Record<string,unknown>)=>{if(!socket||pending||socket.status!=='online')return;const active=socket;setPending(true);setMessage('');try{await active.gameCommand(type,payload);if(attached===active){const v=await active.gameCommand('lobby.list',{}) as {lobbies:Lobby[]};if(attached===active)setLobbies(v.lobbies)}}catch(e){if(attached===active)setMessage((e as Error).message)}finally{if(attached===active)setPending(false)}};
 return <Suspense fallback={<p>Memuat aktivitas…</p>}><LobbyView lobbies={lobbies} arenaId={arenaId} selfId={socket?.selfId??''} online={online} pending={pending} message={message} act={act}/>{arenaId==='race'&&<RaceGame socket={socket} lobby={lobbies.find(l=>l.arenaId==='race')}/>} {arenaId==='duck'&&(!socket?<DuckGame/>:lobbies.filter(l=>l.gameType==='duck'&&l.matchId&&(l.players.some(p=>p.userId===socket.selfId)||l.spectators.includes(socket.selfId))).map(l=><div key={l.matchId}><DuckGame socket={socket} lobby={l}/></div>))} {arenaId==='penalty'&&<PenaltyDuel socket={socket} lobby={lobbies.find(l=>l.arenaId==='penalty')}/>}</Suspense>;
}
export function mountLobbyPanel(container:Element,arenaId:string){const root=createRoot(container);root.render(<LobbyPanel arenaId={arenaId}/>);return()=>root.unmount()}
