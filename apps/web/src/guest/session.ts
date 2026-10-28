import {guestRace} from '../features/race/session';
import {guestSession} from '../features/penalty/session';
import {guestDuck} from '../features/duck/session';
import type {GameType} from './history';
export type LocalMatch=ReturnType<typeof guestRace.start>|ReturnType<typeof guestSession.start>|ReturnType<typeof guestDuck.start>;
export const GuestSession={
 start(gameType:GameType,rules:{at?:number}={}):LocalMatch{const at=rules.at??performance.now();return gameType==='race'?guestRace.start(at):gameType==='penalty'?guestSession.start(at):guestDuck.start(at)},
 clear(){guestRace.clear();guestSession.clear();guestDuck.clear()}
};
/** Finished records/settings survive reload; unfinished clocks never do. */
export function mountGuestLifecycle(){const clear=()=>GuestSession.clear();window.addEventListener('patra-auth-change',clear);window.addEventListener('patra-world-attachment',clear);window.addEventListener('patra-guest-reset',clear);window.addEventListener('pagehide',clear);return()=>{clear();window.removeEventListener('patra-auth-change',clear);window.removeEventListener('patra-world-attachment',clear);window.removeEventListener('patra-guest-reset',clear);window.removeEventListener('pagehide',clear)}}

/** Input/modal teardown shares the session boundary used by the local engines. */
export function mountGuestUICleanup(releaseInputs:()=>void,dialog:Pick<HTMLDialogElement,'open'|'close'>,disposeDemo:()=>void){
 const events=['patra-auth-change','patra-world-attachment','patra-guest-reset'];
 const clear=()=>{releaseInputs();if(dialog.open)dialog.close();else disposeDemo()};
 for(const name of events)window.addEventListener(name,clear);
 return()=>{for(const name of events)window.removeEventListener(name,clear)};
}
