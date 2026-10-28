import React from 'react';
export type Actor={actorId:string;actorType:string;userId?:string;displayName:string;avatarId:string};
export type Participant=Actor&{botAssisted:boolean;outcome:string;score?:number;position?:number;race?:{laps:number;elapsedMs?:number;lapTimesMs?:number[];bestLapMs?:number;dnf:boolean};duck?:{hits:number;shots:number}};
export type MatchRecord={matchId:string;worldId:string;worldName:string;gameType:string;status:string;preparedAt:string;startsAt:string|null;finishedAt:string|null;version:{mapId:string;mapVersion:number;rulesId:string;rulesVersion:number;protocolVersion:number};participants:Actor[];result?:{pingpong?:{scores:number[];rallies:number;reason:string};startsAt:number;finishedAt:number;participants:Participant[];penaltyTurns?:{turn:number;kickerId:string;keeperId:string;kickZone:number;keepZone:number;goal:boolean;kickTimeout:boolean;keepTimeout:boolean}[]}};
export const statusLabels:Record<string,string>={running:'Berlangsung',finalizing:'Hasil belum tersimpan',completed:'Tersimpan',aborted_server:'Terhenti saat server dimulai ulang',aborted_storage:'Tidak ada hasil durable setelah rekonsiliasi',abandoned:'Ditinggalkan',unconfirmed:'Hasil belum dapat dikonfirmasi',unavailable:'Hasil belum tersedia dari mesin permainan'};
function isParticipant(p:Actor|Participant):p is Participant{return 'outcome' in p}
export function MatchDetail({match}:{match:MatchRecord}){
 const r=match.result;
 return <article aria-label="Detail pertandingan"><h3>{match.gameType} · {match.worldName}</h3><p>{statusLabels[match.status]??match.status}</p>
 <p>Mulai: {match.startsAt?new Date(match.startsAt).toLocaleString():'Waktu mulai tidak diketahui'}</p>
 {r&&<p>Durasi: {((r.finishedAt-r.startsAt)/1000).toLocaleString()} detik</p>}
 <p>Peta {match.version.mapId} v{match.version.mapVersion} · aturan {match.version.rulesId} v{match.version.rulesVersion} · protokol {match.version.protocolVersion}</p>
 {(r?.participants??match.participants).map((p,i)=><div key={p.actorId}><strong>{p.displayName} {p.actorType==='bot'?'· BOT':''}</strong>{isParticipant(p)&&<p>{p.botAssisted?'Bot assistance · ':''}{({incomplete:'Tidak lengkap · tidak eligible leaderboard',dnf:'DNF',win:'Menang',loss:'Kalah',draw:'Seri',forfeit:'Kalah karena keluar',abandoned:'Ditinggalkan'} as Record<string,string>)[p.outcome]??'Selesai'}{p.score!==undefined?` · Skor ${p.score}`:''}{p.position!==undefined?` · Posisi ${p.position}`:''}{p.race?` · Grid ${i+1} · ${p.race.laps} lap · Total ${p.race.elapsedMs!==undefined?`${p.race.elapsedMs} ms`:"DNF"}${p.race.bestLapMs!==undefined?` · Lap terbaik ${p.race.bestLapMs} ms`:''}`:''}{p.duck?` · ${p.duck.hits}/${p.duck.shots} hit · Akurasi ${p.duck.shots?((100*p.duck.hits)/p.duck.shots).toFixed(1):'0'}%`:''}</p>}</div>)}
 {match.gameType==='race'&&r&&<p>Seri waktu simulasi memakai slot start tersimpan. DNF memakai progres terakhir.</p>}
 {r?.pingpong&&<p>Pingpong · {r.pingpong.scores.join(' : ')} · {r.pingpong.rallies} rally · {r.pingpong.reason==='deadline'?'Batas waktu':r.pingpong.reason==='forfeit'?'Forfeit':'Target skor'} · 7 poin / unggul 2 / cap 11</p>}
 {r?.penaltyTurns?.map(t=><p key={t.turn}>Giliran {t.turn}: {t.goal?'Gol':'Tidak gol'} · zona {t.kickZone+1}/{t.keepZone+1}{t.kickTimeout||t.keepTimeout?' · pilihan timeout server':''}</p>)}
 </article>
}
