import React, { useSyncExternalStore, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { activities, type ActivityStore } from './store';
import { activityInput } from './input-router';
import type { RaceSnapshot } from '../features/race/local';
import type { PublicPenalty } from '../features/penalty/local';
import './activities.css';
export function ActivityHUD({ store = activities }: {
    store?: ActivityStore;
}) {
    const state = useSyncExternalStore(store.subscribe, store.snapshot, store.snapshot), [collapsed, setCollapsed] = useState(false), game = state.active;
    if (!game) {
        const rider = state.vehicles.find(v => v.riderId === state.selfId);
        if (state.inputMode !== 'ride' && !state.nearbyRide)
            return null;
        const button = (label: string, key: 'steer' | 'throttle' | 'brake', value: number) => <button type="button" disabled={!state.connected} aria-label={label} onPointerDown={e => {
            e.preventDefault();
            e.currentTarget.setPointerCapture(e.pointerId);
            activityInput.pointerControl(e.pointerId, key, value);
        }} onPointerUp={e => activityInput.releasePointer(e.pointerId)} onPointerCancel={e => activityInput.releasePointer(e.pointerId)} onLostPointerCapture={e => activityInput.releasePointer(e.pointerId)}>{label}</button>;
        return <section className="activity-hud" aria-label="Free ride"><strong>{rider ? rider.kind === 'bicycle' ? 'Sepeda · Cruise' : 'Mobil · Cruise' : 'Free ride · ' + (state.nearbyRide?.kind === 'bicycle' ? 'Sepeda' : 'Mobil')}</strong><span>{rider ? 'WASD / arrows · E turun ke titik aman' : 'E untuk naik · tanpa history'}</span>{rider && <div className="activity-hud-controls">{button('←', 'steer', -1)}{button('→', 'steer', 1)}{button('↓', 'brake', 1)}{button('↑', 'throttle', 1)}</div>}<button type="button" disabled={!state.connected} onClick={() => store.interactRide()}>{rider ? 'Turun · E' : 'Naik · E'}</button>{state.message && <span role="status">{state.message}</span>}</section>;
    }
    const saved = state.lobbies.some(l => l.matchId === game.matchId && l.state === 'results'), done = ['finished', 'results'].includes(game.snapshot.phase);
    const race = game.kind === 'race' ? game.snapshot as RaceSnapshot : null, penalty = game.kind === 'penalty' ? game.snapshot as PublicPenalty : null;
    const kart = race?.karts.find(k => k.userId === state.selfId), me = penalty?.actors.findIndex(a => a.userId === state.selfId) ?? -1;
    const controlsDisabled = !state.connected || state.inputMode !== game.kind || done || Boolean(kart?.finished || kart?.dnf) || Boolean(typeof document !== 'undefined' && document.querySelector('dialog[open]'));
    const control = (label: string, key: 'steer' | 'throttle' | 'brake', value: number) => <button type="button" disabled={controlsDisabled} aria-label={label} onPointerDown={e => {
        e.preventDefault();
        e.currentTarget.setPointerCapture(e.pointerId);
        activityInput.pointerControl(e.pointerId, key, value);
    }} onPointerUp={e => activityInput.releasePointer(e.pointerId)} onPointerCancel={e => activityInput.releasePointer(e.pointerId)} onLostPointerCapture={e => activityInput.releasePointer(e.pointerId)} onKeyDown={e => {
            if (e.key === ' ' || e.key === 'Enter') {
                e.preventDefault();
                activityInput.control(key, value);
            }
        }} onKeyUp={e => {
            if (e.key === ' ' || e.key === 'Enter')
                activityInput.control(key, 0);
        }} onBlur={() => activityInput.control(key, 0)}>{label}</button>;
    return <section className="activity-hud" aria-label={race ? 'Kontrol race di dunia' : 'Kontrol penalti di dunia'}><div className="activity-hud-status" role="status"><strong>{race ? done ? 'Race selesai' : race.phase === 'countdown' ? `Mulai ${Math.max(0, Math.ceil((race.startsAt - state.now) / 1000))}s` : `LAP ${Math.min(5, (kart?.laps ?? 0) + 1)}/5 · #${kart?.position ?? '—'}` : `Penalti ${penalty!.scores.join(' : ')}`}</strong><span>{!state.connected ? 'Terputus · kontrol dinetralkan' : done ? state.online ? saved ? 'Hasil tersimpan' : 'Menunggu penyimpanan hasil' : 'Practice selesai' : race ? 'WASD / arrows · Gas / rem' : penalty!.phase === 'choosing' ? `${penalty!.actors[me]?.actorId === penalty!.kickerId ? 'Tendang' : 'Jaga'} · ${Math.max(0, Math.ceil((penalty!.choiceDeadlineAt - state.now) / 1000))}s` : 'Reveal · pilihan terkunci'}</span><button type="button" className="activity-hud-toggle" aria-expanded={!collapsed} aria-label={collapsed ? 'Tampilkan kontrol aktivitas' : 'Sembunyikan kontrol aktivitas'} onClick={() => {
        activityInput.release();
        setCollapsed(!collapsed);
    }}>{collapsed ? 'Kontrol' : '⌄'}</button></div>
 {!collapsed && !done && race && <div className="activity-hud-controls">{control('←', 'steer', -1)}{control('→', 'steer', 1)}{control('Rem', 'brake', 1)}{control('Gas', 'throttle', 1)}</div>}
 {!collapsed && !done && penalty && <div className="activity-hud-controls">{['↖ Zona 1', '↗ Zona 2', '↙ Zona 3', '↘ Zona 4'].map((label, zone) => <button key={zone} type="button" disabled={controlsDisabled || penalty.phase !== 'choosing' || me < 0 || penalty.chosen[me] || state.now >= penalty.choiceDeadlineAt} onClick={() => void store.choose(zone)}>{label}</button>)}</div>}
 {done && <div className="activity-hud-controls"><button type="button" disabled={!state.connected || state.online && !saved} onClick={() => void store.action('return')}>Kembali ke arena</button>{penalty && <button type="button" disabled={!state.connected || state.online && !saved} onClick={() => void store.action('rematch')}>Main lagi</button>}</div>}
 <button type="button" className="activity-hud-details" onClick={() => window.dispatchEvent(new CustomEvent('patra-activity-details', { detail: game.kind }))}>Menu / detail</button>{state.message && <span role="status">{state.message}</span>}</section>;
}
export function mountActivityHUD(container: Element) {
    const root = createRoot(container), dock = document.querySelector('.game-dock');
    const measure = () => {
        if (dock)
            (container as HTMLElement).style.setProperty('--activity-hud-bottom', `${Math.max(12, window.innerHeight - dock.getBoundingClientRect().top + 12)}px`);
    };
    const observer = new ResizeObserver(measure);
    if (dock)
        observer.observe(dock);
    window.addEventListener('resize', measure);
    window.visualViewport?.addEventListener('resize', measure);
    measure();
    root.render(<ActivityHUD />);
    return () => {
        observer.disconnect();
        window.removeEventListener('resize', measure);
        window.visualViewport?.removeEventListener('resize', measure);
        root.unmount();
    };
}
