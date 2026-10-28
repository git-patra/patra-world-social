import React, { useSyncExternalStore } from 'react';
import { activities } from '../../activities/store';
import type { PingpongSnapshot } from './local';
export function PingpongMenu({ online = false }: {
    online?: boolean;
}) {
    const state = useSyncExternalStore(activities.subscribe, activities.snapshot, activities.snapshot), game = state.games.find(g => g.kind === 'pingpong');
    return <section aria-label="Pingpong"><h3>Pingpong · 7 poin</h3><p>Unggul dua · batas 11 · 120 detik. A/D atau tombol panah menggerakkan paddle. Serve lewat tombol; meja tetap berjalan saat menu dibuka.</p>{game ? <><p>Skor {(game.snapshot as PingpongSnapshot).scores.join(' : ')}</p><button type="button" onClick={() => window.dispatchEvent(new Event('patra-activity-play'))}>Lihat meja / kontrol</button>{state.active?.matchId === game.matchId && <button type="button" onClick={() => {
        void activities.action('leave');
        window.dispatchEvent(new Event('patra-activity-play'));
    }}>{online ? 'Keluar · forfeit' : 'Akhiri practice / kembali berjalan'}</button>}</> : online ? <p>Dua pemain Ready, atau pilih BOT secara eksplisit.</p> : <button type="button" onClick={() => void activities.startLocal('pingpong').then(() => window.dispatchEvent(new Event('patra-activity-play')))}>Mulai practice vs BOT</button>}</section>;
}
