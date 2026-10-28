import React, { useSyncExternalStore } from 'react';
import { activities, type ActivityStore } from '../../activities/store';
import { activityInput } from '../../activities/input-router';
import type { PingpongSnapshot } from './local';
export function PingpongHUD({ store = activities, snapshot: s, spectator = false }: {
    store?: ActivityStore;
    snapshot: PingpongSnapshot;
    spectator?: boolean;
}) {
    const state = useSyncExternalStore(store.subscribe, store.snapshot, store.snapshot), me = s.actors.findIndex(a => a.userId === state.selfId), done = s.phase === 'results', saved = state.lobbies.some(l => l.matchId === s.matchId && l.state === 'results'), blocked = !state.connected || state.inputMode !== 'pingpong' || done || Boolean(typeof document !== 'undefined' && document.querySelector('dialog[open]'));
    const control = (label: string, value: number) => <button type="button" disabled={blocked} aria-label={label} onPointerDown={e => {
        e.preventDefault();
        e.currentTarget.setPointerCapture(e.pointerId);
        activityInput.pointerControl(e.pointerId, 'steer', value);
    }} onPointerUp={e => activityInput.releasePointer(e.pointerId)} onPointerCancel={e => activityInput.releasePointer(e.pointerId)} onLostPointerCapture={e => activityInput.releasePointer(e.pointerId)} onKeyDown={e => {
            if (e.key === ' ' || e.key === 'Enter') {
                e.preventDefault();
                activityInput.control('steer', value);
            }
        }} onKeyUp={() => activityInput.control('steer', 0)} onBlur={() => activityInput.control('steer', 0)}>{label}</button>;
    return <section className="activity-hud" aria-label={spectator ? 'Penonton pingpong' : 'Kontrol pingpong di dunia'}><strong>Pingpong · {s.scores.join(' : ')} · {Math.max(0, Math.ceil((s.deadlineAt - state.now) / 1000))}s</strong><span>{spectator ? 'Menonton dari luar · jalan untuk pergi' : done ? state.online ? saved ? 'Hasil tersimpan' : 'Menunggu penyimpanan hasil' : 'Practice selesai' : !state.connected ? 'Terputus · paddle netral' : s.phase === 'serve' ? `Serve: ${s.actors[s.server].displayName}` : 'Rally · A/D atau ←/→'}</span>{!spectator && me >= 0 && <><div className="activity-hud-controls">{!done && <>{control('←', -1)}{control('→', 1)}<button type="button" disabled={blocked || s.phase !== 'serve' || me !== s.server} onClick={() => void store.serve()}>Serve</button></>}{done && <button type="button" disabled={state.online && !saved || !state.connected} onClick={() => void store.action('return')}>Kembali ke meja</button>}</div><button type="button" onClick={() => window.dispatchEvent(new CustomEvent('patra-activity-details', { detail: 'pingpong' }))}>Menu / detail</button></>}{state.message && <span role="status">{state.message}</span>}</section>;
}
