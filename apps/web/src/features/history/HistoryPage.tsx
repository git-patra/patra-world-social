import {guestHistory,type LocalResult} from '../../guest/history';
import React,{useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import {api,type User} from '../../net/api';
import {MatchDetail,statusLabels,type MatchRecord} from './MatchDetail';
import './history.css';
type Page={items:MatchRecord[];nextCursor?:string};
async function read<T>(path:string,signal:AbortSignal):Promise<T>{const response=await fetch(path,{credentials:'same-origin',cache:'no-store',signal});if(!response.ok)throw new Error(response.status===401?'Masuk untuk melihat history akun.':'History belum dapat dimuat. Coba lagi.');return response.json()}
const localNames={penalty:'Penalti',race:'Balapan',duck:'Duck Arcade'};
function LocalDetail({record}:{record:LocalResult}){
 const detail=record.detail as {scores?:number[];turns?:unknown[];result?:{participants:{displayName:string;outcome:string}[]};karts?:{displayName:string;actorType:string;position:number;laps:number;dnf:boolean;elapsedMs?:number}[];hits?:number;shots?:number;score?:number};
 const outcomes:Record<string,string>={win:'Menang',loss:'Kalah',draw:'Seri',finished:'Selesai'};
 return <div><p>Durasi {Math.round(record.durationMs/1000)} detik · hanya perangkat ini</p>{record.gameType==='penalty'&&<><p>{detail.result?.participants.map(p=>`${p.displayName}: ${outcomes[p.outcome]??p.outcome}`).join(' · ')}</p><p>Skor {detail.scores?.join(' : ')} · {detail.turns?.length??0} tendangan · lawan BOT</p></>}{record.gameType==='race'&&<ol>{detail.karts?.slice().sort((a,b)=>a.position-b.position).map(k=><li key={k.position}>{k.displayName} {k.actorType==='bot'?'(BOT)':''} · {k.laps}/5 lap · {k.dnf?'Tidak finis (DNF)':`${((k.elapsedMs??0)/1000).toFixed(2)} detik`}</li>)}</ol>}{record.gameType==='duck'&&<p>Skor {detail.score??0} · Hit {detail.hits??0}/{detail.shots??0} tembakan · Akurasi {detail.shots?((detail.hits??0)/detail.shots*100).toFixed(1):'0'}%</p>}</div>
}
export function GuestHistoryPage(){
 const[items,setItems]=useState<LocalResult[]>([]),[error,setError]=useState('');
 useEffect(()=>{let live=true;const load=()=>void guestHistory.list().then(v=>{if(live)setItems(v)}).catch(()=>{if(live)setError('History perangkat tidak dapat dibaca.')});load();window.addEventListener('patra-guest-history',load);window.addEventListener('patra-guest-reset',load);return()=>{live=false;window.removeEventListener('patra-guest-history',load);window.removeEventListener('patra-guest-reset',load)}},[]);
 return <section><h3>History · perangkat ini</h3><p>Maksimal 100 sesi selesai; tidak masuk leaderboard resmi. Muat ulang mengakhiri sesi belum selesai. Data browser yang hilang tidak dapat dipulihkan.</p><p role="alert">{error}</p>{!items.length&&<p>Belum ada hasil lokal tersimpan.</p>}<ul>{items.map(r=><li key={r.id}><details><summary>{localNames[r.gameType]} · {r.summary} · {new Date(r.finishedAt).toLocaleString()} · perangkat ini</summary><LocalDetail record={r}/></details></li>)}</ul></section>
}
export function HistoryPage({user}:{user:User|null;key?:string}){
 const [items,setItems]=useState<MatchRecord[]>([]),[detail,setDetail]=useState<MatchRecord|null>(null),[matchId,setMatchId]=useState(''),[next,setNext]=useState(''),[cursor,setCursor]=useState(''),[error,setError]=useState(''),[busy,setBusy]=useState(false),[reload,setReload]=useState(0),[message,setMessage]=useState('');
 const [filter,setFilter]=useState({game:'',world:'',status:'',from:'',to:''});
 useEffect(()=>{const listener=(e:Event)=>{const d=(e as CustomEvent).detail;if(d?.type==='match.result_status'){setMessage(statusLabels[d.payload.status]??'');setCursor('');setReload(v=>v+1)}};window.addEventListener('patra-world-event',listener);return()=>window.removeEventListener('patra-world-event',listener)},[]);
 useEffect(()=>{const controller=new AbortController();setError('');setBusy(true);if(!user){setItems([]);setBusy(false);return()=>controller.abort()}
 const query=new URLSearchParams();for(const [key,value]of Object.entries(filter) as [string,string][]){if(value)query.set(key,key==='from'?new Date(value+'T00:00:00').toISOString():key==='to'?new Date(value+'T23:59:59').toISOString():value)}if(cursor)query.set('cursor',cursor);
 void read<Page>('/api/v1/history?'+query,controller.signal).then(page=>{if(!controller.signal.aborted){setItems(old=>cursor?[...old,...page.items]:page.items);setNext(page.nextCursor??'')}}).catch(e=>{if(!controller.signal.aborted)setError(e.message)}).finally(()=>{if(!controller.signal.aborted)setBusy(false)});return()=>controller.abort();
 },[user,filter,cursor,reload]);
 useEffect(()=>{setDetail(null);if(!matchId||!user)return;const controller=new AbortController();void read<MatchRecord>('/api/v1/history/'+encodeURIComponent(matchId),controller.signal).then(value=>{if(!controller.signal.aborted)setDetail(value)}).catch(e=>{if(!controller.signal.aborted)setError(e.message)});return()=>controller.abort()},[matchId,user,reload]);
 const change=(key:string,value:string)=>{setCursor('');setItems([]);setMatchId('');setFilter(old=>({...old,[key]:value}))};
 return <section><GuestHistoryPage/><h2 id="history-title">History pribadi</h2><p>Hanya pertandingan akunmu. Tetap tersedia setelah keluar dari dunia.</p>{!user?<p>Masuk untuk melihat history. Permainan tamu tidak menghasilkan skor resmi.</p>:<>
 <div className="history-filters"><label>Permainan<select value={filter.game} onChange={e=>change('game',e.target.value)}><option value="">Semua</option><option value="race">Race</option><option value="penalty">Penalti</option><option value="duck">Duck</option></select></label>
 <label>Status<select value={filter.status} onChange={e=>change('status',e.target.value)}><option value="">Semua</option>{Object.entries(statusLabels).filter(([s])=>!['unconfirmed','unavailable'].includes(s)).map(([s,label])=><option key={s} value={s}>{label}</option>)}</select></label>
 <label>ID dunia<input value={filter.world} onChange={e=>change('world',e.target.value)} placeholder="Semua dunia"/></label><label>Dari<input type="date" value={filter.from} onChange={e=>change('from',e.target.value)}/></label><label>Sampai<input type="date" value={filter.to} onChange={e=>change('to',e.target.value)}/></label></div>
 <p role="status">{message}</p><p role="alert">{error}</p><button type="button" disabled={busy} onClick={()=>{setCursor('');setReload(v=>v+1)}}>Muat ulang</button>
 {busy&&<p>Memuat history…</p>}{!busy&&!error&&!items.length&&<p>Belum ada pertandingan untuk filter ini.</p>}
 <ul>{items.map(m=><li key={m.matchId}><button type="button" onClick={()=>setMatchId(m.matchId)}>{m.gameType} · {m.worldName} · {new Date(m.preparedAt).toLocaleString()} · {statusLabels[m.status]}</button></li>)}</ul>
 {next&&<button type="button" disabled={busy} onClick={()=>setCursor(next)}>Muat berikutnya</button>}{detail&&<MatchDetail match={detail}/>}</>}
 </section>
}
export function mountHistory(){
 const button=document.querySelector<HTMLButtonElement>('#history-button')!,dialog=document.querySelector<HTMLDialogElement>('#history-dialog')!,container=document.querySelector('#history-content')!;let user:User|null=null;let authEpoch=0;let root:ReturnType<typeof createRoot>|null=null;
 const render=()=>{root?.render(<HistoryPage key={user?.id??'guest'} user={user}/>)};
 window.addEventListener('patra-auth-change',e=>{authEpoch++;user=(e as CustomEvent<User|null>).detail;render()});void api.me().then(value=>{if(authEpoch===0){user=value;render()}}).catch(()=>{});
 button.onclick=()=>{root=createRoot(container);render();dialog.showModal()};dialog.querySelector<HTMLButtonElement>('#close-history')!.onclick=()=>dialog.close();dialog.addEventListener('close',()=>{root?.unmount();root=null;button.focus({preventScroll:true})});
}
