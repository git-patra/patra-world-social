import React,{lazy,Suspense} from 'react';
import {createRoot} from 'react-dom/client';
const PenaltyDuel=lazy(()=>import('../features/penalty/Duel').then(module=>({default:module.PenaltyDuel})));
export function PenaltyGame(){return <Suspense fallback={<p>Memuat penalti…</p>}><PenaltyDuel/></Suspense>}
export function mountPenalty(container){const root=createRoot(container);root.render(<PenaltyGame/>);return()=>root.unmount()}
