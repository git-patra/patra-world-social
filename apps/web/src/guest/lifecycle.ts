const sessions=new Set<{clear:()=>void}>();
export function registerGuestSession(session:{clear:()=>void}){sessions.add(session)}
export function clearGuestSessions(){for(const session of sessions)session.clear()}
/** Finished records/settings survive reload; unfinished clocks never do. */
export function mountGuestLifecycle(){const clear=()=>clearGuestSessions();window.addEventListener('patra-auth-change',clear);window.addEventListener('patra-world-attachment',clear);window.addEventListener('patra-guest-reset',clear);window.addEventListener('pagehide',clear);return()=>{clear();window.removeEventListener('patra-auth-change',clear);window.removeEventListener('patra-world-attachment',clear);window.removeEventListener('patra-guest-reset',clear);window.removeEventListener('pagehide',clear)}}

/** Input/modal teardown shares the session boundary used by the local engines. */
export function mountGuestUICleanup(releaseInputs:()=>void,dialog:Pick<HTMLDialogElement,'open'|'close'>,disposeDemo:()=>void){
 const events=['patra-auth-change','patra-world-attachment','patra-guest-reset'];
 const clear=()=>{releaseInputs();if(dialog.open)dialog.close();else disposeDemo()};
 for(const name of events)window.addEventListener(name,clear);
 return()=>{for(const name of events)window.removeEventListener(name,clear)};
}
