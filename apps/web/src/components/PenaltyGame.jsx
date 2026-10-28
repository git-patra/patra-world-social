import React from 'react';
import {createRoot} from 'react-dom/client';
import {PenaltyDuel} from '../features/penalty/Duel';
export function PenaltyGame(){return <PenaltyDuel/>}
export function mountPenalty(container){const root=createRoot(container);root.render(<PenaltyGame/>);return()=>root.unmount()}
