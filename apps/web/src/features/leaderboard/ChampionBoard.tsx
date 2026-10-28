import React,{useEffect,useRef,useState} from 'react';
import {createRoot} from 'react-dom/client';
import {api,type User} from '../../net/api';
import type {WorldSocket} from '../../net/socket';
import {BoardController,read,accessDenied,type Board,type Entry,type ResultSummary} from './controller';
import './leaderboard.css';
const current='duck-v1@1';
const accuracy=(e:Entry)=>`${(e.shots?e.hits/e.shots*100:0).toFixed(1)}%`;
function resultLink(world:string,rules:string,e:Pick<Entry,'resultId'>){return `#podium/${encodeURIComponent(world)}/${encodeURIComponent(rules)}/${encodeURIComponent(e.resultId)}`}
export function ChampionBoardView({board,busy,error,reload,select}:{board:Board|null;busy:boolean;error:string;reload:()=>void;select:(s:string)=>void}){
 return <section className="champion-board"><h2>Podium Duck dunia privat</h2><p>Top 3 akun unik · satu personal best per akun. Urutan skor, akurasi tepat, waktu pencapaian, lalu ID hasil.</p><p role="alert">{error}</p>{busy&&<p role="status">Memuat podium…</p>}<button type="button" onClick={reload} disabled={busy}>Muat ulang podium</button>
 {board&&<><label>Aturan / arsip <select value={board.ruleset} onChange={e=>select(e.target.value)}>{Array.from(new Set([current,board.ruleset,...board.rulesets])).map(r=><option key={r} value={r}>{r}{r===current?' · saat ini':' · arsip'}</option>)}</select></label><p>Hasil dari aturan berbeda tidak digabung.</p>
 <ol className="podium">{[0,1,2].map(i=>{const e=board.top[i];return <li key={i}><strong>#{i+1}</strong>{e?<><p>{e.displayName}</p><p>{e.score} poin · {accuracy(e)}</p><a href={resultLink(board.worldId,board.ruleset,e)}>Ringkasan hasil</a></>:<p>Belum ada skor sah</p>}</li>})}</ol>
 <h3>Personal best</h3>{board.personalBest?<p>{board.personalBest.score} poin · {accuracy(board.personalBest)} · <a href={resultLink(board.worldId,board.ruleset,board.personalBest)}>Ringkasan hasil</a></p>:<p>Belum ada personal best pada aturan ini.</p>}
 <h3>Hasil sesi terakhir</h3>{board.lastResult?<p>{board.lastResult.status==='completed'?`${board.lastResult.score} poin · ${accuracy(board.lastResult)} · ${board.lastResult.complete?'Selesai':'Tidak lengkap · tidak masuk podium'}`:'Dibatalkan · tidak masuk podium'}{board.lastResult.complete&&<> · <a href={resultLink(board.worldId,board.ruleset,board.lastResult)}>Ringkasan hasil</a></>}</p>:<p>Belum ada sesi tersimpan pada aturan ini.</p>}
 <h3>Juara Terakhir · pergantian pemilik #1</h3><p>Pemilik yang sama menaikkan personal best tidak dihitung sebagai pergantian.</p>{board.changes.length?<ol>{board.changes.map(c=><li key={c.id}>{c.displayName} · {new Date(c.changedAt).toLocaleString()}{c.userId&&<> · <a href={resultLink(board.worldId,board.ruleset,c)}>Hasil saat menjadi juara</a></>}</li>)}</ol>:<p>Belum ada pergantian juara.</p>}
 {board.administrativeChanges.length>0&&<><h3>Penyesuaian administratif</h3><ul>{board.administrativeChanges.map(c=><li key={c.id}>Akun dihapus · podium dihitung ulang, bukan kemenangan baru · {new Date(c.changedAt).toLocaleString()}</li>)}</ul></>}
 </>}
 </section>
}
export function ChampionBoard({world,rules,match,socket,onRules}:{key?:string;world:string;rules:string;match:string;socket:WorldSocket|null;onRules:(r:string)=>void}){
 const [board,setBoard]=useState<Board|null>(null),[error,setError]=useState(''),[busy,setBusy]=useState(true),[detail,setDetail]=useState<ResultSummary|null>(null);
 const retry=useRef<()=>void>(()=>{});
 useEffect(()=>{
  let active=true,denied=false,generation=0,controller:BoardController|null=null,detailRequest:AbortController|null=null;
  const cancel=()=>{generation++;controller?.dispose();detailRequest?.abort()};
  const deny=(message:string)=>{denied=true;cancel();setBoard(null);setDetail(null);setError(message);setBusy(false)};
  const begin=()=>{
   cancel();denied=false;setBoard(null);setDetail(null);setError('');setBusy(true);
   const attempt=generation,live=()=>active&&!denied&&attempt===generation;
   const failed=(message:string,cause:unknown)=>{if(!live())return;if(accessDenied(cause)){deny(message);return}setError(message);setBusy(false)};
   controller=new BoardController(world,rules,read,b=>{if(live()){setBoard(b);setError('');setBusy(false)}},(message,cause)=>{if(live())setBoard(null);failed(message,cause)});
   void controller.load();
   if(match){const request=detailRequest=new AbortController();void read<ResultSummary>(`/api/v1/leaderboards/${encodeURIComponent(world)}/results/${encodeURIComponent(match)}`,request.signal).then(value=>{if(live()&&!request.signal.aborted)setDetail(value)}).catch(cause=>{if(!request.signal.aborted)failed(cause.message,cause)})}
  };
  retry.current=begin;
  const event=(event:Event)=>{const d=(event as CustomEvent).detail;if(!denied&&d?.socket===socket&&socket?.worldId===world)controller?.event(d.type,d.payload)};
  let wasOnline=socket?.status==='online';
  const status=(event:Event)=>{
   const d=(event as CustomEvent).detail;if(d?.socket!==socket||socket?.worldId!==world)return;
   const online=d.state==='online',reconnected=online&&!wasOnline;wasOnline=online;
   if(['access_denied','solo'].includes(d.state)){deny('Keanggotaan aktif atau sesi tidak tersedia. Coba lagi setelah akses dipulihkan.');return}
   // Repeated online snapshots are not a fresh authorization decision. Only a
   // reconnect transition, replacement attachment, or explicit retry rechecks.
   if(reconnected)begin();
  };
  const focus=()=>{if(!denied)void controller?.load()};
  window.addEventListener('patra-world-event',event);window.addEventListener('patra-world-status',status);window.addEventListener('focus',focus);
  if(socket?.worldId===world&&socket.status==='access_denied')deny('Keanggotaan aktif atau sesi tidak tersedia. Coba lagi setelah akses dipulihkan.');else begin();
  return()=>{active=false;cancel();retry.current=()=>{};window.removeEventListener('patra-world-event',event);window.removeEventListener('patra-world-status',status);window.removeEventListener('focus',focus)};
 },[world,rules,match,socket]);
 return <><ChampionBoardView board={board} busy={busy} error={error} reload={()=>retry.current()} select={onRules}/>{detail&&<section><h3>Ringkasan hasil dunia</h3><p>{detail.displayName} · {detail.score} poin · {detail.hits}/{detail.shots} hit · {accuracy(detail)}</p><p>{new Date(detail.achievedAt).toLocaleString()} · Map {detail.version.mapId} v{detail.version.mapVersion} · Aturan {detail.version.rulesId} v{detail.version.rulesVersion} · Protokol v{detail.version.protocolVersion}</p><p>Ringkasan ini tersedia untuk anggota aktif dunia. History akun tetap pribadi.</p></section>}</>
}
export function mountChampionBoard(){
 const button=document.querySelector<HTMLButtonElement>('#podium-button')!,dialog=document.querySelector<HTMLDialogElement>('#podium-dialog')!,container=document.querySelector('#podium-content')!;let socket:WorldSocket|null=null,user:User|null=null,authEpoch=0,root:ReturnType<typeof createRoot>|null=null;
 function route(){const parts=location.hash.split('/');if(parts[0]!=='#podium')return null;try{return {world:decodeURIComponent(parts[1]??''),rules:decodeURIComponent(parts[2]??current),match:decodeURIComponent(parts[3]??'')}}catch{return null}}
 const render=()=>{if(!root)return;const r=route(),world=r?.world||socket?.worldId;root.render(!user?<p>Masuk untuk melihat podium dunia privat. Tamu hanya bermain solo lokal.</p>:world?<ChampionBoard key={`${user.id}:${world}:${r?.rules??current}`} world={world} rules={r?.rules??current} match={r?.match??''} socket={socket} onRules={rules=>{location.hash=`podium/${encodeURIComponent(world)}/${encodeURIComponent(rules)}`}}/>:<p>Sambungkan ke dunia privat untuk membuka podium.</p>)};
 const open=()=>{if(!root)root=createRoot(container);render();if(!dialog.open)dialog.showModal()};button.onclick=()=>{if(route())history.replaceState(null,'',location.pathname+location.search);open()};dialog.querySelector<HTMLButtonElement>('#close-podium')!.onclick=()=>dialog.close();dialog.addEventListener('close',()=>{root?.unmount();root=null;button.focus({preventScroll:true})});
 window.addEventListener('patra-world-attachment',e=>{socket=(e as CustomEvent).detail;if(!socket&&root){root.unmount();root=createRoot(container)};render()});window.addEventListener('patra-auth-change',e=>{authEpoch++;user=(e as CustomEvent).detail;render()});window.addEventListener('hashchange',()=>{if(route())open()});void api.me().then(v=>{if(authEpoch===0){user=v;if(route())open();else render()}}).catch(()=>{});
}
