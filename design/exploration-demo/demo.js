import {createWorldArt,drawCharacter} from './art.js';

const $=s=>document.querySelector(s), $$=s=>[...document.querySelectorAll(s)];
const canvas=$('#world'),ctx=canvas.getContext('2d'),art=createWorldArt(),TILE=32;
const mapImage=document.createElement('canvas');mapImage.width=art.width;mapImage.height=art.height;
const mapCtx=mapImage.getContext('2d');mapCtx.drawImage(art.background,0,0);[...art.foregroundObjects].sort((a,b)=>a.y-b.y).forEach(o=>o.draw(mapCtx,0));
const clamp=(v,a,b)=>Math.max(a,Math.min(b,v)),dist=(a,b)=>Math.hypot(a.x-b.x,a.y-b.y);
const zones=[
 {id:'plaza',name:'Alun-alun',sub:'Ruang untuk bertemu',x:1280,y:1040,r:325,desc:'Mulai dari sini'},
 {id:'warkop',name:'Warkop Senja',sub:'Kopi, cerita, tanpa terburu-buru',x:640,y:650,r:400,desc:'Teras & meja bersama'},
 {id:'arcade',name:'Halaman Arcade',sub:'Main bareng, santai bareng',x:1824,y:610,r:360,desc:'Duck Arcade di udara terbuka'},
 {id:'garden',name:'Kebun Lentera',sub:'Sudut tenang di bawah pepohonan',x:608,y:1440,r:380,desc:'Api unggun & tempat duduk'},
 {id:'penalty',name:'Lapangan Penalti',sub:'Satu tendangan, banyak sorakan',x:1120,y:1540,r:235,desc:'Lapangan terbuka'},
 {id:'race',name:'Lintasan Mini',sub:'Putaran kecil, semangat besar',x:1792,y:1510,r:370,desc:'Lintasan & area penonton'},
 {id:'river',name:'Tepi Sungai',sub:'Ambil jeda, nikmati senja',x:2144,y:960,r:290,desc:'Jembatan & dermaga'}
];
const seats=[
 {x:448,y:610,sitX:448,sitY:612,name:'Meja Warkop',copy:'Duduk dan nikmati suasana teras.'},
 {x:800,y:610,sitX:800,sitY:612,name:'Meja Warkop',copy:'Ada tempat kosong di sini.'},
 {x:572,y:1455,sitX:576,sitY:1514,name:'Lingkar api unggun',copy:'Sudut hangat untuk cerita panjang.'},
 {x:2075,y:1010,sitX:2108,sitY:1048,name:'Bangku tepi sungai',copy:'Duduk sebentar, dunia tidak ke mana-mana.'},
 {x:1415,y:1050,sitX:1456,sitY:1104,name:'Bangku plaza',copy:'Tempat melihat orang berlalu-lalang.'}
];
const npcs=[
 {id:'nara',name:'Nara',x:535,y:630,color:'#d78b68',zone:'warkop',line:'Kopinya masih hangat. Sini duduk!'},
 {id:'bimo',name:'Bimo',x:710,y:665,color:'#819ab3',zone:'warkop',line:'Habis ngopi, keliling dulu yuk.'},
 {id:'sora',name:'Sora',x:1344,y:1048,color:'#b19ac7',zone:'plaza',line:'Ada lima daun emas tersembunyi di kampung.'},
 {id:'aji',name:'Aji',x:1752,y:675,color:'#caa25b',zone:'arcade',line:'Nunggu teman? Nongkrong di halaman dulu.'},
 {id:'luna',name:'Luna',x:646,y:1450,color:'#bd8099',zone:'garden',line:'Ini tempat favoritku kalau mau tenang.'},
 {id:'rio',name:'Rio',x:1992,y:1005,color:'#8ba57d',zone:'river',line:'Coba lewat jembatan. Pemandangannya enak.'}
];
const leaves=[{x:850,y:840},{x:490,y:780},{x:1880,y:800},{x:800,y:1500},{x:2090,y:1110}].map((p,i)=>({...p,id:i,found:false}));
const player={x:1280,y:1040,name:'Kamu',color:'#efd0a0',isPlayer:true,facing:'down',walking:false,seated:false};
const camera={x:player.x,y:player.y},keys=new Set(),visited=new Set(['plaza']);
let w=0,h=0,dpr=1,zoom=innerWidth<700?1.1:1,route=[],last=performance.now(),time=0,lastZone='plaza',currentSeat=null,standPosition=null,nearSeat=null,nearNpc=null,waveUntil=0,toastTimer,moved=0,motion=!matchMedia('(prefers-reduced-motion:reduce)').matches,dialogTrigger=null;

function notify(text){$('#toast').textContent=text;$('#toast').classList.add('visible');clearTimeout(toastTimer);toastTimer=setTimeout(()=>$('#toast').classList.remove('visible'),3500)}
function dismissWelcome(){$('#welcome').classList.add('dismissed');$('#welcome').hidden=true}
function resize(){w=canvas.clientWidth;h=canvas.clientHeight;dpr=Math.min(devicePixelRatio||1,2);canvas.width=Math.round(w*dpr);canvas.height=Math.round(h*dpr);ctx.imageSmoothingEnabled=false}
new ResizeObserver(resize).observe(canvas);resize();
function blocked(x,y,r=11){return x<32+r||y<32+r||x>art.width-32-r||y>art.height-32-r||art.solids.some(o=>x+r>o.x&&x-r<o.x+o.w&&y+r>o.y&&y-r<o.y+o.h)}
function nearestOpen(x,y){const tx=clamp(Math.floor(x/TILE),1,78),ty=clamp(Math.floor(y/TILE),1,58);for(let r=0;r<12;r++)for(let dy=-r;dy<=r;dy++)for(let dx=-r;dx<=r;dx++){if(Math.max(Math.abs(dx),Math.abs(dy))!==r)continue;const p={x:(tx+dx)*TILE+16,y:(ty+dy)*TILE+16};if(!blocked(p.x,p.y))return p}return null}
function pathTo(x,y){
 const start=nearestOpen(player.x,player.y),target=nearestOpen(x,y);if(!start||!target)return [];
 const k=p=>`${Math.floor(p.x/TILE)},${Math.floor(p.y/TILE)}`,sk=k(start),tk=k(target),queue=[start],parents=new Map([[sk,null]]),points=new Map([[sk,start]]);
 for(let i=0;i<queue.length;i++){const p=queue[i],pk=k(p);if(pk===tk){const path=[];let key=tk;while(key&&key!==sk){path.push(points.get(key));key=parents.get(key)}path.push(start);return path.reverse()}
  for(const [dx,dy]of[[0,-1],[1,0],[0,1],[-1,0]]){const n={x:p.x+dx*TILE,y:p.y+dy*TILE},nk=k(n);if(parents.has(nk)||blocked(n.x,n.y))continue;parents.set(nk,pk);points.set(nk,n);queue.push(n)}
 }return [];
}
function stand(){if(player.seated&&standPosition){player.x=standPosition.x;player.y=standPosition.y}player.seated=false;currentSeat=null;standPosition=null;$('#player-status').textContent='Berkeliling'}
function go(x,y,name){stand();route=pathTo(x,y);if(!route.length)notify('Jalur terhalang. Coba pilih tanah terbuka di dekatnya.');else if(name)notify(`Jalan ke ${name}. Tekan arah untuk mengambil alih.`);dismissWelcome()}
function screenToWorld(x,y){return{x:(x-w/2)/zoom+camera.x,y:(y-h/2)/zoom+camera.y}}
canvas.addEventListener('pointerdown',e=>{if(e.button!==0)return;const r=canvas.getBoundingClientRect(),p=screenToWorld(e.clientX-r.left,e.clientY-r.top);go(p.x,p.y);canvas.focus({preventScroll:true})});
function showDialog(id){keys.clear();route=[];dialogTrigger=document.activeElement;$(id).showModal()}
$$('.close-dialog').forEach(b=>b.onclick=()=>b.closest('dialog').close());
$$('dialog').forEach(d=>{d.addEventListener('close',()=>{keys.clear();dialogTrigger?.focus({preventScroll:true})});d.addEventListener('click',e=>{if(e.target===d){const r=d.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)d.close()}})});
function detail(title,copy,extra=''){ $('#detail-title').textContent=title;$('#detail-copy').textContent=copy;$('#detail-extra').innerHTML=extra;showDialog('#detail-dialog') }
$('#detail-action').onclick=()=>$('#detail-dialog').close();
$('#help-button').onclick=()=>detail('Selamat datang di Kampung Senja.','Ini prototipe desain untuk merasakan world yang menyambung: berjalan melewati ruang bersama, berpapasan, dan berhenti di mana pun terasa nyaman.','<div class="feature-list"><span><b>Jalan:</b> WASD / panah, atau klik tanah. Shift untuk berlari. Di ponsel, gunakan tombol arah.</span><span><b>Coba:</b> E untuk duduk atau melihat aktivitas, Q untuk menyapa, M untuk peta. Temukan 5 daun emas.</span><span><b>Batas demo:</b> Semua karakter lain adalah NPC. Proximity hanya simulasi visual. Tidak ada mikrofon, jaringan multiplayer, akun, atau pertandingan yang dijalankan.</span></div>');
$('#dismiss-welcome').onclick=()=>dismissWelcome();
function showMap(){drawMap($('#full-minimap'),true);showDialog('#map-dialog')}
$('#map-button').onclick=showMap;$('#mini-button').onclick=showMap;
$('#destinations').innerHTML=zones.map(z=>`<button data-zone="${z.id}"><strong>${z.name} <span aria-hidden="true">↗</span></strong><small>${z.desc}</small></button>`).join('');
$$('[data-zone]').forEach(b=>b.onclick=()=>{const z=zones.find(z=>z.id===b.dataset.zone);$('#map-dialog').close();go(z.x,z.y,z.name)});
function wave(){waveUntil=time+3;notify(nearNpc?`${nearNpc.name} menyapa balik! · NPC demo`:'Hai, kampung! 👋');dismissWelcome()}
$('#wave-button').onclick=wave;
function interact(){
 if(player.seated){stand();notify('Yuk, lanjut jalan.');return}
 if(nearSeat){standPosition={x:player.x,y:player.y};player.seated=true;route=[];keys.clear();currentSeat=nearSeat;player.x=nearSeat.sitX;player.y=nearSeat.sitY;player.facing='down';$('#player-status').textContent='Lagi duduk';notify(`${nearSeat.name} · ${nearSeat.copy}`);return}
 const z=zones.find(z=>z.id===lastZone);
 if(z&&['arcade','race','penalty'].includes(z.id)){detail(z.name,`Kamu sudah berada di area ${z.name.toLowerCase()}. Dalam rancangan ini, pemain dapat menonton dan berkumpul di sekitar area sambil menunggu giliran.`, '<div class="feature-list"><span><b>Konsep interaksi:</b> pilih ikut permainan saat siap. Lewat atau menonton tidak membuka panel otomatis.</span><span><b>Demo desain:</b> pertandingan belum disambungkan di prototipe ini. Implementasi game yang ada tetap berada di aplikasi utama.</span></div>');return}
 if(nearNpc){wave();return}
 notify('Dekati bangku atau meja untuk duduk. Cari penanda lingkaran kecil.');
}
$('#interact-button').onclick=interact;
function setMotion(value){motion=value;document.body.classList.toggle('motion-off',!value);$('#motion-button').setAttribute('aria-pressed',String(value));$('#motion-button').setAttribute('aria-label',value?'Kurangi animasi':'Aktifkan animasi');$('#motion-button').classList.toggle('active',!value)}
setMotion(motion);$('#motion-button').onclick=()=>{setMotion(!motion);notify(motion?'Animasi lingkungan aktif.':'Animasi lingkungan dikurangi.')};
matchMedia('(prefers-reduced-motion:reduce)').addEventListener('change',e=>setMotion(!e.matches));
function setZoom(z){zoom=clamp(z,.65,1.5);$('#zoom-value').textContent=`${Math.round(zoom*100)}%`}
setZoom(zoom);$('#zoom-in').onclick=()=>setZoom(zoom+.15);$('#zoom-out').onclick=()=>setZoom(zoom-.15);
const directions={w:[0,-1],ArrowUp:[0,-1],s:[0,1],ArrowDown:[0,1],a:[-1,0],ArrowLeft:[-1,0],d:[1,0],ArrowRight:[1,0],up:[0,-1],down:[0,1],left:[-1,0],right:[1,0]};
function canMove(){return !document.querySelector('dialog[open]')&&!document.hidden}
window.addEventListener('keydown',e=>{if(!canMove()||e.ctrlKey||e.metaKey||e.altKey||['INPUT','TEXTAREA','SELECT'].includes(e.target.tagName))return;const key=e.key.length===1?e.key.toLowerCase():e.key;if(directions[key]||key==='Shift'){e.preventDefault();keys.add(key);route=[];stand()}else if(!e.repeat){if(key==='e'){e.preventDefault();interact()}if(key==='q'){e.preventDefault();wave()}if(key==='m'){e.preventDefault();showMap()}}});
window.addEventListener('keyup',e=>keys.delete(e.key.length===1?e.key.toLowerCase():e.key));
window.addEventListener('blur',()=>keys.clear());document.addEventListener('visibilitychange',()=>{keys.clear();last=performance.now()});
$$('[data-dir]').forEach(b=>{b.addEventListener('pointerdown',e=>{e.preventDefault();b.setPointerCapture(e.pointerId);keys.clear();keys.add(b.dataset.dir);route=[];stand()});const stop=()=>keys.delete(b.dataset.dir);b.addEventListener('pointerup',stop);b.addEventListener('pointercancel',stop);b.addEventListener('lostpointercapture',stop);b.addEventListener('click',e=>{if(e.detail===0){const[dx,dy]=directions[b.dataset.dir];go(player.x+dx*64,player.y+dy*64)}})});

function update(dt){
 let dx=0,dy=0;for(const key of keys){if(directions[key]){dx+=directions[key][0];dy+=directions[key][1]}}
 if(!dx&&!dy&&route.length){const next=route[0],d=dist(next,player);if(d<4){route.shift()}else{dx=(next.x-player.x)/d;dy=(next.y-player.y)/d}}
 const mag=Math.hypot(dx,dy),speed=(keys.has('Shift')?285:190)*dt;
 player.walking=mag>0&&!player.seated;
 if(player.walking){const step=Math.min(speed,route.length?dist(route[0],player):speed);dx=dx/mag*step;dy=dy/mag*step;let movedStep=false;if(!blocked(player.x+dx,player.y)){player.x+=dx;movedStep=true}if(!blocked(player.x,player.y+dy)){player.y+=dy;movedStep=true}if(!movedStep&&route.length)route=[];player.facing=Math.abs(dx)>Math.abs(dy)?(dx>0?'right':'left'):(dy>0?'down':'up');moved+=dt;if(moved>.8)dismissWelcome()}
 const follow=motion?Math.min(1,dt*9):1;camera.x+=(player.x-camera.x)*follow;camera.y+=(player.y-camera.y)*follow;
 const marginX=Math.min(w/zoom/2,art.width/2),marginY=Math.min(h/zoom/2,art.height/2);camera.x=clamp(camera.x,marginX,art.width-marginX);camera.y=clamp(camera.y,marginY,art.height-marginY);
 let z=zones.filter(z=>dist(z,player)<z.r).sort((a,b)=>dist(a,player)-dist(b,player))[0];if(!z)z={id:'paths',name:'Jalan Kampung',sub:'Tidak perlu buru-buru sampai'};
 if(z.id!==lastZone){lastZone=z.id;$('#zone-name').textContent=z.name;$('#zone-sub').textContent=z.sub;$('#sr-location').textContent=`Memasuki ${z.name}.`;if(z.id!=='paths')visited.add(z.id);$('#zone-count').textContent=`${visited.size} / 7`}
 nearSeat=seats.find(s=>dist(s,player)<92);nearNpc=npcs.filter(n=>dist(n,player)<145).sort((a,b)=>dist(a,player)-dist(b,player))[0];
 $('#nearby').hidden=!nearNpc||!$('#welcome').classList.contains('dismissed');if(nearNpc){$('#nearby-name').textContent=`Dekat ${nearNpc.name} · NPC demo`;$('#nearby-copy').textContent=nearNpc.line}
 const isGame=['arcade','race','penalty'].includes(lastZone);
 $('#interaction-text').textContent=player.seated?`${currentSeat.name} · E untuk berdiri`:nearSeat?`E · Duduk di ${nearSeat.name.toLowerCase()}`:isGame?'E · Lihat konsep aktivitas':nearNpc?'Q · Sapa teman di dekatmu':'Klik tanah atau gunakan arah untuk menjelajah';
 $('#interact-label').textContent=player.seated?'Berdiri':nearSeat?'Duduk':isGame?'Aktivitas':'Duduk';
 $('#interact-button').disabled=!player.seated&&!nearSeat&&!isGame;
 for(const l of leaves)if(!l.found&&dist(l,player)<27){l.found=true;const n=leaves.filter(l=>l.found).length;$('#leaves-count').textContent=`${n} / 5 daun ditemukan`;$('#leaf-slots').textContent=Array.from({length:5},(_,i)=>i<n?'●':'○').join(' ');notify(n===5?'Lima daun ditemukan! Seluruh kampung terasa seperti rumah.':`Daun emas ditemukan! ${n} / 5 · Teruskan jalanmu.`)}
 canvas.dataset.position=`${player.x.toFixed(1)},${player.y.toFixed(1)}`;canvas.dataset.zone=lastZone;canvas.dataset.seated=String(player.seated);canvas.dataset.route=String(route.length);
}
function circle(c,x,y,r,fill,stroke){c.beginPath();c.arc(x,y,r,0,Math.PI*2);if(fill){c.fillStyle=fill;c.fill()}if(stroke){c.strokeStyle=stroke;c.lineWidth=2;c.stroke()}}
function label(x,y,text,bg='#243b32ed',fg='#fcf1d4'){ctx.font='600 11px DM, sans-serif';const tw=ctx.measureText(text).width;ctx.fillStyle=bg;ctx.beginPath();ctx.roundRect(Math.round(x-tw/2-8),Math.round(y-16),tw+16,22,6);ctx.fill();ctx.fillStyle=fg;ctx.textAlign='center';ctx.fillText(text,Math.round(x),Math.round(y-1))}
function render(){
 ctx.setTransform(dpr,0,0,dpr,0,0);ctx.fillStyle='#64744a';ctx.fillRect(0,0,w,h);ctx.save();ctx.translate(Math.round(w/2-camera.x*zoom),Math.round(h/2-camera.y*zoom));ctx.scale(zoom,zoom);ctx.imageSmoothingEnabled=false;ctx.drawImage(art.background,0,0);
 if(route.length){ctx.strokeStyle='#fff4c470';ctx.lineWidth=3;ctx.setLineDash([3,9]);ctx.beginPath();ctx.moveTo(player.x,player.y);for(const p of route)ctx.lineTo(p.x,p.y);ctx.stroke();ctx.setLineDash([]);const p=route.at(-1);circle(ctx,p.x,p.y,9,'#fae8ab55','#fff2ba')}
 if(nearNpc){circle(ctx,player.x,player.y,148,'#f4de9a08','#ffe4a030')}
 for(const s of seats){circle(ctx,s.x,s.y,15,'#fff0c512',dist(s,player)<92?'#ffe0a0aa':'#ffedbd35');ctx.fillStyle='#ebd59b99';ctx.fillRect(s.x-3,s.y-3,6,6)}
 for(const l of leaves)if(!l.found){const b=motion?Math.sin(time*2+l.id)*3:0;circle(ctx,l.x,l.y,16,'#f9db7830');ctx.save();ctx.translate(l.x,l.y-6+b);ctx.rotate(.5);ctx.fillStyle='#91613c';ctx.fillRect(-1,2,3,11);ctx.fillStyle='#f8d67d';ctx.fillRect(-6,-6,12,14);ctx.fillRect(-3,-9,7,19);ctx.fillStyle='#e4a45b';ctx.fillRect(0,-5,2,14);ctx.restore()}
 const objects=art.foregroundObjects.map(o=>({y:o.y,draw:()=>o.draw(ctx,motion?time*1000:0)}));
 for(const n of npcs)objects.push({y:n.y,draw:()=>drawCharacter(ctx,{...n,name:'',facing:n.x>player.x?'left':'right',seated:false,walking:false,isPlayer:false},motion?time*1000:0)});
 objects.push({y:player.y,draw:()=>{circle(ctx,player.x,player.y+1,21,'#ffe5a012','#ffe8ac99');drawCharacter(ctx,{...player,name:''},motion?time*1000:0)}});objects.sort((a,b)=>a.y-b.y).forEach(o=>o.draw());
 for(const n of npcs){label(n.x,n.y-54,`${n.name} · NPC`,'#253a31dd','#ebe9d2');if(waveUntil>time&&dist(n,player)<200)label(n.x,n.y-90,'Hai! 👋','#f5e7c9','#304332')}
 label(player.x,player.y-58,player.seated?'Kamu · duduk':'Kamu','#f5e5bd','#304732');if(waveUntil>time)label(player.x,player.y-94,'Halo! 👋','#f6e7c7','#324633');
 ctx.restore();drawMap($('#minimap'));
}
function drawMap(c,large=false){const mc=c.getContext('2d');mc.imageSmoothingEnabled=false;mc.clearRect(0,0,c.width,c.height);mc.drawImage(mapImage,0,0,c.width,c.height);const sx=c.width/art.width,sy=c.height/art.height;for(const z of zones){circle(mc,z.x*sx,z.y*sy,large?4:2,'#f2d09c');if(large){mc.font='600 13px DM,sans-serif';mc.textAlign='center';const tw=mc.measureText(z.name).width;mc.fillStyle='#243b32e8';mc.fillRect(z.x*sx-tw/2-6,z.y*sy-28,tw+12,20);mc.fillStyle='#fff0d0';mc.fillText(z.name,z.x*sx,z.y*sy-14)}}mc.strokeStyle='#ffedb9a0';mc.lineWidth=1;mc.strokeRect((camera.x-w/zoom/2)*sx,(camera.y-h/zoom/2)*sy,w/zoom*sx,h/zoom*sy);circle(mc,player.x*sx,player.y*sy,large?6:4,'#fff0bf','#3c5141');}
function frame(now){const dt=Math.min((now-last)/1000,.04);last=now;if(canMove()){time+=dt;update(dt)}else player.walking=false;render();requestAnimationFrame(frame)}requestAnimationFrame(frame);
