import {activities,mountActivities} from './activities/store';
import {activityInput} from './activities/input-router';
import {mountActivityHUD} from './activities/ActivityHUD';
import {mountAudio} from './audio/events';
import {mountMinimap} from './world/minimap';
import {openCenteredDialog,closeDialogThen,mountDialogViewport,setDialogFocusReturn} from './ui/dialogs';
import {villageLocation} from './world/village';
import {mountSettings} from './features/settings/SettingsPage';
import {mountGuestLifecycle,mountGuestUICleanup} from './guest/lifecycle';
import {activePreferences} from './preferences';
import {mountVoice,mountVoicePanel} from './features/voice/VoicePanel';
import {createWorld} from './world.js';
import {PointerHold,acceptsMovement} from './world/input';
import {stations} from './data.js';
import {mountPenalty} from './components/PenaltyGame.jsx';
import {mountAuth} from './auth-ui.ts';
import {mountChampionBoard} from './features/leaderboard/ChampionBoard.tsx';
import {mountHistory} from './features/history/HistoryPage.tsx';
import {mountOnline} from './online-ui.ts';
import {mountSocial} from './social-ui.ts';
import {mountLobbies,mountLobbyPanel} from './features/lobby/LobbyPanel';
import {mountChat} from './chat-ui.ts';
import {mountWorlds} from './worlds-ui.ts';
const unmountDialogViewport=mountDialogViewport();
const unmountAudio=mountAudio();
const $=(selector)=>document.querySelector(selector);
const dialog=$('#project-dialog'),content=$('#dialog-content');
const visited=new Set(['plaza']);
let nearby=null,world,cleanupDemo=null;
function disposeDemo(){cleanupDemo?.();cleanupDemo=null;}
let motion=!matchMedia('(prefers-reduced-motion: reduce)').matches;
const placeMarkup=(station,i)=>`<button class="place-row" data-project="${station.id}" style="--place-color:${station.color}"><span class="place-number">${String(i+1).padStart(2,'0')}</span><span class="place-icon">${station.icon}</span><span class="place-copy"><strong>${station.label}</strong><small>${station.short}</small></span><span class="place-arrow">↗</span></button>`;
$('#places').innerHTML=stations.map(placeMarkup).join('');
$('#visited-count').textContent='1 / 7 sudut';
function hideWelcome(){$('#welcome-note').hidden=true;}
function updateMotion(){
 $('#motion-button').textContent=`Motion ${motion?'on':'reduced'}`;
 $('#motion-button').setAttribute('aria-pressed',String(motion));
 document.body.classList.toggle('reduced-motion',!motion);
 world?.setMotion(motion);
}
function notify(message){$('#travel-status').textContent=message;}
function updateInteraction(){
 const state=activities.snapshot(),station=stations.find(s=>s.id===nearby);
 const riding=state.inputMode==='ride',vehicle=state.nearbyRide,football=state.nearbyFootball;
 $('#interact-button').disabled=!(football||riding||vehicle||station);
 $('#interact-button span').textContent=football?'Kick':riding?'Turun':vehicle?`Naik ${vehicle.kind==='bicycle'?'Sepeda':'Mobil'}`:station?`Explore ${station.name}`:'Dekati aktivitas';
 if(riding)$('#current-location').textContent=stations.find(s=>s.id==='race').name;
}

world=createWorld($('#world'),{
 stations,minimapCanvas:$('#minimap'),
 onNear(id){
  nearby=id;updateInteraction();
 },
 onSelect(id){openProject(id);},
 onMove({x,y,walking}){
  const location=activities.snapshot().inputMode==='ride'?stations.find(s=>s.id==='race'):villageLocation({x,y});$('#current-location').textContent=location.name;if(location.id!=='paths')visited.add(location.id);$('#visited-count').textContent=`${visited.size} / 7 sudut`;
  $('#coordinates').textContent=`${String(Math.round(x)).padStart(2,'0')} : ${String(Math.round(y)).padStart(2,'0')}`;
  $('#world').dataset.walking=String(walking);
  $('#world').dataset.position=`${x.toFixed(2)},${y.toFixed(2)}`;
  if(walking)hideWelcome();
 }
});
const unmountActivities=mountActivities(world);
const unmountRideDock=activities.subscribe(updateInteraction);
const unmountActivityInput=activityInput.install();
const unregisterWalk=activityInput.register('walk',input=>{if(input.x||input.y)world.move(input.x,input.y);else world.stopInput()});
const activityHUD=document.createElement('div');document.body.append(activityHUD);
const unmountActivityHUD=mountActivityHUD(activityHUD);
window.addEventListener('patra-activity-details',event=>openProject(event.detail));
window.addEventListener('patra-activity-play',()=>dialog.close());
updateMotion();
function showDialog(){
 world.setPaused(true);
 openCenteredDialog(dialog);
 $('#close-dialog').focus();
}
function openProject(id){
 const station=stations.find(s=>s.id===id);if(!station)return;
 hideWelcome();
 document.querySelector(`[data-project="${id}"]`)?.classList.add('visited');
 disposeDemo();
 dialog.classList.toggle('game-dialog',id==='arcade');
 dialog.classList.toggle('race-dialog',id==='race');
 content.innerHTML=`<div class="case-header" style="--case-color:${station.color}"><div class="case-eyebrow"><span>${station.icon}</span>${station.status}</div><h2 id="dialog-title">${station.title}</h2><p>${station.description}</p></div>${id==='warkop'?'<div id="voice-root"></div>':''}${id==='arcade'?'<div id="penalty-root"></div>':['race','penalty','duck'].includes(id)?'<div id="lobby-root"></div>':''}<div class="case-footer"><button class="walk-button" id="walk-to-place">Jalan ke ${station.name} →</button><span>${visited.size} / 7 sudut dijelajahi</span><button id="back-to-world">Kembali ke dunia ↗</button></div>`;
 notify(`Aktivitas ${station.name}.`);
 if(!dialog.open)showDialog();
 else {dialog.scrollTop=0;$('#close-dialog').focus();}
 $('#back-to-world').onclick=()=>dialog.close();
 $('#walk-to-place').onclick=()=>closeDialogThen(dialog,()=>{syncWorldPause();world.travelTo(id);notify(`Walking to ${station.name}.`);});
 if(id==='warkop')cleanupDemo=mountVoicePanel($('#voice-root'));
 if(['race','penalty','duck'].includes(id))cleanupDemo=mountLobbyPanel($('#lobby-root'),id);
 if(id==='arcade')cleanupDemo=mountPenalty($('#penalty-root'));
}
document.addEventListener('click',event=>{const button=event.target.closest('[data-project]');if(!button)return;
 if(button.closest('#places')){
  const station=stations.find(s=>s.id===button.dataset.project);
  hideWelcome();closeDialogThen($('#map-dialog'),()=>{syncWorldPause();notify(`Walking to ${station.name}. Tekan E untuk membuka aktivitas setelah tiba.`);world.travelTo(station.id);});
 }else openProject(button.dataset.project);});
function openMap(trigger){world.stopInput();world.drawMap($('#village-map'));openCenteredDialog($('#map-dialog'),trigger);}
$('#list-button').onclick=()=>openMap($('#list-button'));
const unmountMinimap=mountMinimap($('#minimap-button'),{openMap:()=>openMap($('#minimap-button'))});
$('#village-map').addEventListener('click',event=>{
 const point=world.mapDestination({x:event.clientX,y:event.clientY},$('#village-map').getBoundingClientRect());
 if(!point)return;
 hideWelcome();closeDialogThen($('#map-dialog'),()=>{syncWorldPause();world.walkToPoint(point);notify('Walking there. Tekan E untuk berinteraksi setelah tiba.');});
});
$('#close-map').onclick=()=>$('#map-dialog').close();
$('#menu-button').onclick=()=>openCenteredDialog($('#menu-dialog'),$('#menu-button'));
$('#close-menu').onclick=()=>$('#menu-dialog').close();
$('#menu-dialog').addEventListener('click',e=>{
 if(!e.target.closest('button'))return;
 $('#menu-dialog').close();
 // Feature handlers and delegated quick play finish opening their child first.
 queueMicrotask(()=>{const child=[...document.querySelectorAll('dialog')].find(d=>d!==$('#menu-dialog')&&d.open);if(child)setDialogFocusReturn(child,$('#menu-button'));});
});
let zoom=1;
$('#zoom-in').onclick=()=>{zoom=Math.min(1.5,zoom+.15);world.setZoom(zoom)};
$('#zoom-out').onclick=()=>{zoom=Math.max(.65,zoom-.15);world.setZoom(zoom)};
$('#close-dialog').onclick=()=>dialog.close();
dialog.addEventListener('click',e=>{if(e.target===dialog){const r=dialog.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)dialog.close();}});
dialog.addEventListener('close',()=>{disposeDemo();syncWorldPause();});
$('#dismiss-note').onclick=hideWelcome;
$('#interact-button').onclick=()=>world.interact();
$('#reset-button').onclick=()=>{zoom=1;world.recenter();notify('Kamera dipusatkan ke karaktermu.');};
$('.brand').onclick=e=>{e.preventDefault();zoom=1;world.recenter();};
$('#motion-button').onclick=()=>{motion=!motion;updateMotion();};
const media=matchMedia('(prefers-reduced-motion: reduce)');
media.addEventListener('change',e=>{motion=!e.matches;updateMotion();});
document.addEventListener('keydown',event=>{
 if(document.body.dataset.controls==='touch'||!worldVisible||document.querySelector('dialog[open]')||event.ctrlKey||event.metaKey||event.altKey||!acceptsMovement(event.target))return;
 const key=event.key.toLowerCase();
 if(key==='e'&&['walk','ride'].includes(activityInput.mode())){event.preventDefault();if(!event.repeat)world.interact();}
 else if(key==='m'){event.preventDefault();$('#list-button').click();}
});
const pointerHold=new PointerHold((dx,dy)=>activityInput.direction(dx,dy),()=>activityInput.direction(0,0));
let movementPointer=null;
function stopHold(event){if(event&&event.pointerId!==movementPointer)return;movementPointer=null;pointerHold.release();}
function releaseInputs(){stopHold();activityInput.release();world.stopInput();}
document.querySelectorAll('[data-move]').forEach(button=>{
 const direction=button.dataset.move.split(',').map(Number);
 button.addEventListener('pointerdown',event=>{event.preventDefault();movementPointer=event.pointerId;pointerHold.start(...direction);event.currentTarget.setPointerCapture?.(event.pointerId);hideWelcome();});
 button.addEventListener('click',event=>{if(event.detail===0){movementPointer=null;pointerHold.pulse(...direction);hideWelcome();}});
});
window.addEventListener('pointerup',stopHold);window.addEventListener('pointercancel',stopHold);window.addEventListener('lostpointercapture',stopHold);window.addEventListener('blur',releaseInputs);
window.addEventListener('pagehide',event=>{releaseInputs();if(!event.persisted){unmountRideDock();unmountActivityHUD();unmountActivityInput();unregisterWalk();unmountActivities();activityHUD.remove();unmountMinimap();unmountDialogViewport();unmountAudio();disposeDemo();world.destroy();}});

let worldVisible=true;
function syncWorldPause(){activityInput.block('viewport',!worldVisible);world.setPaused(Boolean(document.querySelector('dialog[open]'))||!worldVisible||document.hidden);}
const visibilityObserver=new IntersectionObserver(([entry])=>{worldVisible=entry.isIntersecting;syncWorldPause();},{threshold:0.1});
visibilityObserver.observe($('#world'));
document.addEventListener('focusin',event=>{if(!acceptsMovement(event.target))releaseInputs()});
document.addEventListener('visibilitychange',()=>{releaseInputs();syncWorldPause();});

$('#locations-button').onclick=()=>$('#list-button').click();
$('#play-button').onclick=()=>openProject('arcade');
mountLobbies();
mountOnline(world);
mountSettings();
mountAuth();
mountGuestLifecycle();
mountHistory();
mountChampionBoard();
mountWorlds();
mountSocial();
mountChat();mountVoice();

const modalObserver=new MutationObserver(()=>{releaseInputs();syncWorldPause()});
document.querySelectorAll('dialog').forEach(d=>modalObserver.observe(d,{attributes:true,attributeFilter:['open']}));

function applyPreferences(p){motion=!p.reducedMotion;document.body.dataset.controls=p.controls;document.body.dataset.audio=String(p.audio);world.setLocalIdentity(p.displayName,p.avatar);updateMotion();}
activePreferences.subscribe(applyPreferences);
activePreferences.start();
window.addEventListener('patra-guest-storage-error',e=>notify(e.detail));
mountGuestUICleanup(releaseInputs,dialog,disposeDemo);
$('#solo-button').onclick=()=>{window.dispatchEvent(new Event('patra-solo-request'));$('#settings-button').click()};
$('#friends-play-button').onclick=()=>window.dispatchEvent(new Event('patra-worlds-request'));
$('#motion-button').onclick=()=>{void activePreferences.toggleMotion().catch(()=>notify('Pengaturan belum tersimpan.'))};
