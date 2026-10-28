import {guestRace} from '../features/race/session';
import {guestSession} from '../features/penalty/session';
import {guestDuck} from '../features/duck/session';
import type {GameType} from './history';
export type LocalMatch=ReturnType<typeof guestRace.start>|ReturnType<typeof guestSession.start>|ReturnType<typeof guestDuck.start>;
export const GuestSession={
 start(gameType:GameType,rules:{at?:number}={}):LocalMatch{const at=rules.at??performance.now();return gameType==='race'?guestRace.start(at):gameType==='penalty'?guestSession.start(at):guestDuck.start(at)},
 clear(){guestRace.clear();guestSession.clear();guestDuck.clear()}
};
export {mountGuestLifecycle,mountGuestUICleanup} from './lifecycle';
