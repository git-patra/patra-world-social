import {villageLocation} from './world/village';
import {mountSettings} from './features/settings/SettingsPage';
import {mountGuestLifecycle,mountGuestUICleanup} from './guest/session';
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
const $=(selector)=>document.querySelector(selector);
const dialog=$('#project-dialog'),content=$('#dialog-content');
const visited=new Set(['plaza']);
let nearby=null,world,lastTrigger=null,cleanupDemo=null;
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
world=createWorld($('#world'),{
 stations,
 onNear(id){
  nearby=id;const station=stations.find(s=>s.id===id);

  $('#interact-button').disabled=!station;
  $('#interact-button span').textContent=station?`Explore ${station.name}`:'Dekati aktivitas';
 },
 onSelect(id){openProject(id);},
 onMove({x,y,walking}){
  const location=villageLocation({x,y});$('#current-location').textContent=location.name;if(location.id!=='paths')visited.add(location.id);$('#visited-count').textContent=`${visited.size} / 7 sudut`;
  $('#coordinates').textContent=`${String(Math.round(x)).padStart(2,'0')} : ${String(Math.round(y)).padStart(2,'0')}`;
  $('#world').dataset.walking=String(walking);
  $('#world').dataset.position=`${x.toFixed(2)},${y.toFixed(2)}`;
  if(walking)hideWelcome();
 }
});
updateMotion();
function showDialog(){
 lastTrigger=document.activeElement;
 world.setPaused(true);
 dialog.showModal();
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
 $('#walk-to-place').onclick=()=>{dialog.close();requestAnimationFrame(()=>{$('#home').scrollIntoView({behavior:'instant'});world.setPaused(false);world.travelTo(id);notify(`Walking to ${station.name}.`);});};
 if(id==='warkop')cleanupDemo=mountVoicePanel($('#voice-root'));
 if(['race','penalty','duck'].includes(id))cleanupDemo=mountLobbyPanel($('#lobby-root'),id);
 if(id==='arcade')cleanupDemo=mountPenalty($('#penalty-root'));
}
document.addEventListener('click',event=>{const button=event.target.closest('[data-project]');if(!button)return;
 if(button.closest('#places')){
  const station=stations.find(s=>s.id===button.dataset.project);
  $('#map-dialog').close();hideWelcome();requestAnimationFrame(()=>{syncWorldPause();notify(`Walking to ${station.name}. Tekan E untuk membuka aktivitas setelah tiba.`);world.travelTo(station.id);});
 }else openProject(button.dataset.project);});
$('#list-button').onclick=()=>{world.stopInput();world.drawMap($('#village-map'));$('#map-dialog').showModal();};
$('#close-map').onclick=()=>$('#map-dialog').close();
$('#menu-button').onclick=()=>$('#menu-dialog').showModal();
$('#close-menu').onclick=()=>$('#menu-dialog').close();
$('#menu-dialog').addEventListener('click',e=>{if(e.target.closest('button'))$('#menu-dialog').close()});
let zoom=1;
$('#zoom-in').onclick=()=>{zoom=Math.min(1.5,zoom+.15);world.setZoom(zoom)};
$('#zoom-out').onclick=()=>{zoom=Math.max(.65,zoom-.15);world.setZoom(zoom)};
$('#close-dialog').onclick=()=>dialog.close();
dialog.addEventListener('click',e=>{if(e.target===dialog){const r=dialog.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)dialog.close();}});
dialog.addEventListener('close',()=>{disposeDemo();syncWorldPause();lastTrigger?.focus?.({preventScroll:true});});
$('#dismiss-note').onclick=hideWelcome;
$('#interact-button').onclick=()=>world.interact();
$('#reset-button').onclick=()=>{zoom=1;world.recenter();notify('Kamera dipusatkan ke karaktermu.');};
$('.brand').onclick=e=>{e.preventDefault();zoom=1;world.recenter();};
$('#motion-button').onclick=()=>{motion=!motion;updateMotion();};
const media=matchMedia('(prefers-reduced-motion: reduce)');
media.addEventListener('change',e=>{motion=!e.matches;updateMotion();});
const directions={w:[0,-1],a:[-1,0],s:[0,1],d:[1,0],ArrowUp:[0,-1],ArrowLeft:[-1,0],ArrowDown:[0,1],ArrowRight:[1,0]};
document.addEventListener('keydown',event=>{
 if(document.body.dataset.controls==='touch'||!worldVisible||document.querySelector('dialog[open]')||event.ctrlKey||event.metaKey||event.altKey||!acceptsMovement(event.target))return;
 const key=event.key.length===1?event.key.toLowerCase():event.key;
 if(directions[key]){event.preventDefault();heldKeys.add(key);applyDirection();hideWelcome();}
 else if(key==='e'){event.preventDefault();world.interact();}
 else if(key==='m'){event.preventDefault();$('#list-button').click();}
});
const heldKeys=new Set();
function applyDirection(){let dx=0,dy=0;for(const key of heldKeys){dx+=directions[key][0];dy+=directions[key][1]}if(dx||dy)world.move(Math.sign(dx),Math.sign(dy));else world.stopInput();}
const keyboardTimer=setInterval(()=>{if(world.isOnline()&&heldKeys.size&&worldVisible&&!document.hidden&&acceptsMovement(document.activeElement)&&!document.querySelector('dialog[open]')){applyDirection();}},100);
window.addEventListener('keyup',event=>{const key=event.key.length===1?event.key.toLowerCase():event.key;if(!directions[key])return;heldKeys.delete(key);applyDirection()});
const pointerHold=new PointerHold((dx,dy)=>world.move(dx,dy),()=>world.stopInput());
function stopHold(){pointerHold.release();}
function releaseInputs(){stopHold();heldKeys.clear();world.stopInput();}
document.querySelectorAll('[data-move]').forEach(button=>{
 const direction=button.dataset.move.split(',').map(Number);
 button.addEventListener('pointerdown',event=>{event.preventDefault();pointerHold.start(...direction);hideWelcome();});
 button.addEventListener('click',event=>{if(event.detail===0){pointerHold.pulse(...direction);hideWelcome();}});
});
window.addEventListener('pointerup',stopHold);window.addEventListener('pointercancel',stopHold);window.addEventListener('blur',releaseInputs);
window.addEventListener('pagehide',event=>{releaseInputs();if(!event.persisted){clearInterval(keyboardTimer);disposeDemo();world.destroy();}});

let worldVisible=true;
function syncWorldPause(){world.setPaused(Boolean(document.querySelector('dialog[open]'))||!worldVisible||document.hidden);}
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
