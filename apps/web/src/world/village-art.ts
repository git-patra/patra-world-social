// A deliberately self-contained canvas painting for the exploration prototype.
// Positions use feet / ground contact as their depth-sorting anchor.
const W = 2560, H = 1920;
const C = {
  grass: '#7f956b', grassLight: '#8b9d73', grassDark: '#748c64', grassGold: '#a5a16d',
  ink: '#354c43', path: '#e0c69b', pathLight: '#edd6aa', pathEdge: '#b39771',
  brick: '#c78768', brickLight: '#d89b79', brickDark: '#ad705b',
  wood: '#af7956', woodLight: '#c99568', woodDark: '#765441',
  cream: '#f4dfb4', green: '#455d49', orange: '#d88a4e', red: '#b7634c', purple: '#71677f',
};

function random(seed = 17429) {
  let n = seed;
  return () => { n = (n * 1664525 + 1013904223) >>> 0; return n / 4294967296; };
}
const rect = (c,x,y,w,h,color) => { c.fillStyle=color; c.fillRect(Math.round(x),Math.round(y),w,h); };
function ellipse(c,x,y,rx,ry,color) { c.fillStyle=color; c.beginPath(); c.ellipse(x,y,rx,ry,0,0,Math.PI*2); c.fill(); }
function line(c,points,color,width=4) {
  c.strokeStyle=color; c.lineWidth=width; c.lineCap='butt'; c.lineJoin='miter';
  c.beginPath(); points.forEach(([x,y],i)=>i ? c.lineTo(x,y) : c.moveTo(x,y)); c.stroke();
}
function rounded(c,x,y,w,h,r,color) {
  c.fillStyle=color; c.beginPath(); c.roundRect(x,y,w,h,r); c.fill();
}
function label(c,text,x,y,size=17,color=C.cream) {
  c.save(); c.fillStyle=color; c.font=`800 ${size}px ui-monospace, SFMono-Regular, Menlo, monospace`;
  c.textAlign='center'; c.textBaseline='middle'; c.fillText(text,x,y); c.restore();
}
function shadow(c,x,y,w=36,h=12) { ellipse(c,x,y,w,h,'#354c4326'); }
function pixelLeaf(c,x,y,color,s=4) { rect(c,x,y,s*2,s,color); rect(c,x+s,y-s,s,s*3,color); }
function paving(c,x,y,w,h,brick=false) {
  c.save(); c.beginPath(); c.rect(x,y,w,h); c.clip();
  rect(c,x,y,w,h,brick ? C.brick : C.path);
  const tileW=brick?32:48, tileH=brick?16:32;
  for(let yy=y;yy<y+h;yy+=tileH) {
    const row=Math.round((yy-y)/tileH);
    for(let xx=x-(row%2)*tileW/2;xx<x+w;xx+=tileW) {
      rect(c,xx+1,yy+1,tileW-2,tileH-2,brick ? (row%3===0?C.brickLight:C.brick) : (row%3===0?C.pathLight:C.path));
      rect(c,xx+1,yy+tileH-2,tileW-2,1,brick?'#ae735d65':'#bca47d60');
    }
  }
  c.restore();
}
function planks(c,x,y,w,h) {
  rect(c,x-8,y-8,w+16,h+20,C.woodDark);
  for(let yy=y;yy<y+h;yy+=20) {
    rect(c,x,yy,w,18,((yy-y)/20)%3===0?C.woodLight:C.wood);
    for(let xx=x+24;xx<x+w;xx+=128) {
      rect(c,xx,yy+4,3,3,'#785742'); rect(c,xx,yy+13,3,3,'#785742');
      rect(c,xx+10,yy+9,38,2,'#a36f4d66');
    }
  }
  rect(c,x,y+h,w,6,'#d4a679'); rect(c,x-8,y+h+6,w+16,6,C.woodDark);
}

function drawTree(c,x,y,size,palette,seed) {
  const r=random(seed), s=size;
  shadow(c,x+5,y-1,s*.46,s*.17);
  rect(c,x-9,y-s*.66,18,s*.67,'#795b44');
  rect(c,x-4,y-s*.66,7,s*.66,'#a5825a');
  rect(c,x-14,y-6,28,7,'#66583f');
  line(c,[[x,y-s*.27],[x-s*.23,y-s*.56]],'#795b44',8);
  line(c,[[x,y-s*.36],[x+s*.22,y-s*.72]],'#795b44',7);
  const crownY=y-s*.88;
  const layers=[[-.26,.06,.46,.36],[.04,-.01,.47,.4],[-.37,-.2,.5,.42],[-.04,-.36,.48,.45],[-.18,-.53,.38,.36]];
  layers.forEach(([dx,dy,w,h],i)=> {
    const px=x+dx*s, py=crownY+dy*s;
    rect(c,px+7,py,w*s-14,h*s,palette[0]); rect(c,px,py+9,w*s,h*s-18,palette[0]);
    rect(c,px+7,py+6,w*s-16,h*s-15,palette[i%2+1]);
  });
  for(let i=0;i<40;i++) {
    const xx=x+(r()-.5)*s*.68, yy=crownY-s*.39+r()*s*.7;
    rect(c,Math.round(xx/4)*4,Math.round(yy/4)*4,r()>.7?12:8,4,palette[2+Math.floor(r()*2)]);
  }
  rect(c,x-7,crownY-s*.46,10,6,palette[3]);
}

function drawBush(c,x,y,w=50,color='#627d53') {
  shadow(c,x,y,w*.5,8);
  rect(c,x-w/2,y-19,w,18,'#526c48'); rect(c,x-w/2+5,y-29,w-10,26,color);
  rect(c,x-w/2+11,y-35,w-22,14,color);
  for(let n=0;n<5;n++) rect(c,x-w/2+8+n*8,y-25+(n%2)*7,6,4,'#91a36a');
}

function drawPumpkin(c,x,y,s=1,face=false) {
  shadow(c,x,y,14*s,5*s);
  rect(c,x-12*s,y-18*s,24*s,16*s,'#bd663b');
  rect(c,x-9*s,y-22*s,18*s,22*s,'#e0974d');
  rect(c,x-4*s,y-23*s,9*s,23*s,'#edaa59');
  rect(c,x-1*s,y-28*s,5*s,7*s,'#5c7048');
  if(face) { rect(c,x-7*s,y-15*s,4*s,4*s,'#654934'); rect(c,x+4*s,y-15*s,4*s,4*s,'#654934'); rect(c,x-4*s,y-7*s,9*s,3*s,'#654934'); }
}

function drawBench(c,x,y,vertical=false) {
  c.save(); c.translate(x,y);
  if(vertical) c.rotate(Math.PI/2);
  shadow(c,0,5,42,12);
  rect(c,-34,-11,6,19,C.ink); rect(c,28,-11,6,19,C.ink);
  rect(c,-43,-25,86,8,C.woodDark); rect(c,-43,-35,86,8,C.woodLight);
  rect(c,-45,-10,90,9,C.woodDark); rect(c,-45,-15,90,9,C.woodLight);
  rect(c,-43,-2,86,5,C.wood); rect(c,-39,-37,5,34,C.ink); rect(c,34,-37,5,34,C.ink);
  c.restore();
}

function drawTable(c,x,y) {
  shadow(c,x,y,42,16);
  rect(c,x-6,y-40,12,44,C.woodDark); rect(c,x-25,y-3,50,6,C.woodDark);
  ellipse(c,x,y-37,42,24,'#745740'); ellipse(c,x,y-42,42,24,'#d2a475');
  line(c,[[x-32,y-51],[x+32,y-51]],'#b88b60',2); line(c,[[x-39,y-39],[x+39,y-39]],'#b88b60',2);
  rect(c,x+9,y-53,11,13,C.cream); rect(c,x+10,y-55,9,3,'#715745'); rect(c,x+19,y-50,4,6,C.cream);
  rect(c,x-13,y-50,8,12,'#f7d6a0'); rect(c,x-13,y-53,8,3,'#cf8154');
}

function drawStool(c,x,y) {
  shadow(c,x,y,14,6); rect(c,x-10,y-15,5,16,C.woodDark); rect(c,x+5,y-15,5,16,C.woodDark);
  rect(c,x-15,y-22,30,8,C.woodDark); rect(c,x-15,y-25,30,6,'#d0a071');
}

function drawLamp(c,x,y,t) {
  shadow(c,x,y,16,6); rect(c,x-7,y-4,14,6,C.ink); rect(c,x-3,y-88,6,85,C.ink);
  rect(c,x-13,y-108,26,22,C.ink); rect(c,x-9,y-104,18,14,'#efc785');
  rect(c,x-14,y-111,28,5,C.ink); rect(c,x-8,y-116,16,5,C.ink); rect(c,x-2,y-121,4,5,C.ink);
  rect(c,x-2,y-105,4,17,C.ink);
  c.globalAlpha=.06+Math.sin(t*.001+x)*.015; ellipse(c,x,y-96,24,23,'#ffdd9a'); c.globalAlpha=1;
}

export function createWorldArt() {
  const background=document.createElement('canvas'); background.width=W; background.height=H;
  const c=background.getContext('2d'); c.imageSmoothingEnabled=false;
  const foregroundObjects=[],solids=[];
  const r=random();
  const solid=(x,y,w,h)=>solids.push({x,y,w,h});
  const object=(x,y,draw)=>foregroundObjects.push({x,y,draw});
  const tree=(x,y,size=110,kind=0)=>{
    const palettes=[['#936448','#ba7d4f','#d39b60','#e6b579'],['#9a614d','#ba7657','#d79465','#e5ab78'],['#61734e','#83935c','#a4a66b','#c0b878'],['#6c634d','#8a8057','#b09a65','#cab47f']];
    const seed=Math.round(x*4+y); object(x,y,(ctx)=>drawTree(ctx,x,y,size,palettes[kind%4],seed)); solid(x-12,y-18,24,25);
  };
  const bush=(x,y,w=50)=>object(x,y,(ctx)=>drawBush(ctx,x,y,w));
  const pumpkin=(x,y,s=1,face=false)=>object(x,y,(ctx)=>drawPumpkin(ctx,x,y,s,face));
  const bench=(x,y,vertical=false,backClearance=0)=>{ object(x,y,(ctx)=>drawBench(ctx,x,y,vertical)); if(!vertical) solid(x-45,y-30+backClearance,90,32-backClearance); else solid(x-15,y-44,30,88); };
  const lamp=(x,y)=>{ object(x,y,(ctx,t=0)=>drawLamp(ctx,x,y,t)); solid(x-8,y-8,16,16); };

  // Ground has tiny, irregular tufts rather than a repeating checkerboard.
  rect(c,0,0,W,H,C.grass);
  for(let i=0;i<4400;i++) {
    const x=Math.floor(r()*W/4)*4,y=Math.floor(r()*H/4)*4;
    rect(c,x,y,4+Math.floor(r()*3)*4,4,[C.grassLight,C.grassDark,'#84996d','#8d9a6b'][Math.floor(r()*4)]);
    if(r()>.8) rect(c,x+4,y-4,4,4,C.grassDark);
  }
  for(let i=0;i<90;i++) {
    const x=r()*2180,y=r()*H;
    ellipse(c,x,y,30+r()*80,12+r()*36,'#c0b16a0d');
  }

  // A complete, connected lane network: every district has multiple approaches.
  const roads=[
    [[640,688],[640,960],[1120,960]],
    [[1472,960],[1824,960],[1824,672]],
    [[1992,608],[2160,608],[2160,1392],[2048,1392]],
    [[1504,960],[2376,960]],
    [[1824,1504],[1824,1696],[1104,1696]],
    [[1104,1696],[896,1696],[896,1376],[736,1376]],
    [[576,1200],[576,960],[640,960]],
    [[1280,1120],[1280,1232],[1120,1232],[1120,1360]],
  ];
  roads.forEach(p=>line(c,p,C.pathEdge,116));
  roads.forEach(p=>line(c,p,C.path,104));
  c.save(); c.beginPath(); roads.forEach(p=>{p.forEach(([x,y],i)=>i?c.lineTo(x,y):c.moveTo(x,y));}); c.lineWidth=104; c.strokeStyle=C.path; c.stroke();
  c.restore();
  // Fine path joints and scattered light paving make the routes read at any zoom.
  roads.forEach(p=>{
    for(let i=1;i<p.length;i++) {
      const [ax,ay]=p[i-1],[bx,by]=p[i],d=Math.hypot(bx-ax,by-ay);
      for(let n=18;n<d;n+=48) {
        const x=ax+(bx-ax)*n/d,y=ay+(by-ay)*n/d;
        if(ax===bx) line(c,[[x-49,y],[x+49,y]],'#cbb18b',1); else line(c,[[x,y-49],[x,y+49]],'#cbb18b',1);
      }
    }
  });

  // A scalloped riverbank, with a deliberate gap in collision at the footbridge.
  const banks=[2236,2252,2244,2228,2240,2252,2244,2228,2236,2252,2244,2236];
  for(let i=0;i<12;i++) {
    const y=i*160,left=banks[i];
    rect(c,left-12,y,216,160,'#c3b984'); rect(c,left-4,y,208,160,'#d3c393');
    rect(c,left+4,y,188,160,'#538c87'); rect(c,left+12,y,172,160,'#67a19a');
    const segStart=y,segEnd=y+160;
    if(segStart<864) solid(left+4,segStart,188,Math.min(segEnd,864)-segStart);
    if(segEnd>1056) solid(left+4,Math.max(segStart,1056),188,segEnd-Math.max(segStart,1056));
    for(let j=0;j<16;j++) {
      const xx=left+24+r()*144,yy=y+r()*160;
      rect(c,xx,yy,12+r()*30,3,j%3===0?'#97c0ab':'#79b1a5');
    }
  }
  // Reeds and tiny stones sit on both banks; the bridge stays wide and open.
  for(let i=0;i<28;i++) {
    const y=80+i*64;if(y>830&&y<1090)continue;
    const left=banks[Math.min(11,Math.floor(y/160))];
    [left-15,left+210].forEach(x=>{
      rect(c,x,y-16,3,18,'#687f55');rect(c,x+6,y-23,3,25,'#687f55');rect(c,x+5,y-27,5,9,'#a49360');
      rect(c,x-8,y+7,11,5,'#a6ad87');
    });
  }
  planks(c,2176,872,296,176);
  rect(c,2170,870,308,9,'#e0b886'); rect(c,2170,1042,308,9,'#d4a775');
  for(let x=2176;x<2472;x+=56) {
    rect(c,x,852,8,26,C.woodDark); rect(c,x,1034,8,26,C.woodDark);
  }
  solid(2170,852,308,12); solid(2170,1056,308,12);

  // The plaza is the visual center; several paths join its broad open edges.
  rounded(c,1012,724,536,472,36,'#a7795d');
  c.save(); c.beginPath(); c.roundRect(1020,732,520,456,30); c.clip();
  paving(c,1016,728,528,464,true);
  rounded(c,1040,752,480,416,24,'#e5c997'); rounded(c,1048,760,464,400,18,C.brick);
  // Inlaid center compass / medallion.
  ellipse(c,1280,997,102,63,'#d9b180'); ellipse(c,1280,997,88,51,'#c48867');
  line(c,[[1214,997],[1346,997]],'#dfb889',3); line(c,[[1280,952],[1280,1042]],'#dfb889',3);
  c.restore();
  label(c,'ALUN-ALUN',1280,1129,16,'#775846');
  // Fountain appears above avatars standing behind it and below those in front.
  object(1280,916,(ctx,t=0)=>{
    shadow(ctx,1280,915,76,25);
    ellipse(ctx,1280,892,76,41,'#8f9a84'); ellipse(ctx,1280,885,76,39,'#e2d0a7');
    ellipse(ctx,1280,882,64,29,'#527f7a'); ellipse(ctx,1280,879,59,25,'#78a99b');
    ellipse(ctx,1280,878,45,17,'#91b6a4');
    rect(ctx,1271,819,18,58,'#c2bea0'); rect(ctx,1266,852,28,20,'#dad0ae');
    ellipse(ctx,1280,820,39,16,'#9b9d87'); ellipse(ctx,1280,814,39,15,'#e1d1a8');
    ellipse(ctx,1280,811,30,9,'#76a697');
    rect(ctx,1276,791,8,23,'#b6d1b6'); ellipse(ctx,1280,790,8,5,'#d7e0be');
    for(let i=0;i<5;i++) { const xx=1256+i*12,yy=830+((t*.035+i*13)%42); rect(ctx,xx,yy,3,7,'#b7d3bb'); }
    rect(ctx,1221,889,8,5,'#c0dbbd'); rect(ctx,1300,873,13,3,'#c0dbbd');
  });
  solid(1216,816,128,100);
  bench(1100,852,true); bench(1460,852,true); bench(1104,1104); bench(1456,1104);
  lamp(1044,770); lamp(1516,770); lamp(1044,1160); lamp(1516,1160);
  pumpkin(1079,1172,.9,true); pumpkin(1497,1172,1.1); pumpkin(1523,1177,.7);

  // Warkop: an inviting, entirely open deck, with all service fixtures at the back.
  planks(c,320,320,640,448);
  rect(c,326,744,628,18,'#d6aa79'); rect(c,352,769,576,12,'#c39167');
  // Back wall is only one counter deep; the large side and south entries are free.
  object(624,416,(ctx)=>{
    rect(ctx,328,292,624,76,'#efd2a3'); rect(ctx,328,344,624,12,'#c29b6b');
    rect(ctx,312,284,656,21,'#665a48'); rect(ctx,306,276,668,13,'#805e47');
    for(let i=0;i<16;i++) { rect(ctx,312+i*41,306,41,38,i%2?'#f0d7ad':'#be7957');rect(ctx,312+i*41,344,41,12,i%2?'#e8cca2':'#a96a4e'); }
    rect(ctx,352,365,544,52,'#855c45'); rect(ctx,348,356,552,13,'#e0ad76');
    for(let x=364;x<884;x+=64) {rect(ctx,x,378,48,26,'#986c4c');rect(ctx,x+3,381,42,3,'#ac7c55');}
    // Coffee machine, glass bottles, pastry case and hanging menu.
    rect(ctx,390,326,48,31,'#3e5750');rect(ctx,395,322,38,7,'#60756b');rect(ctx,397,336,14,7,'#c5c5a3');rect(ctx,420,334,9,18,'#b0b1a0');
    rect(ctx,449,344,14,12,C.cream);rect(ctx,467,344,14,12,C.cream);rect(ctx,482,337,4,20,'#e6ca98');
    rect(ctx,530,331,11,23,'#718f69');rect(ctx,533,326,5,6,'#496348');rect(ctx,548,335,11,19,'#b98050');
    rect(ctx,704,326,88,29,'#8a8f78');rect(ctx,708,329,80,20,'#bed0b0');
    for(let x=715;x<784;x+=18) { rect(ctx,x,340,12,6,'#b9824d');rect(ctx,x+2,336,8,4,'#e3b670'); }
    rect(ctx,818,308,56,39,'#41584a'); label(ctx,'KOPI',846,319,9,'#f1d8a8');label(ctx,'SENJA',846,332,9,'#f1d8a8');
    rect(ctx,492,251,304,36,'#3f584b');rect(ctx,500,257,288,23,'#4c6452');label(ctx,'WARKOP SENJA',644,269,20);
  });
  solid(352,352,544,64);
  [448,800].forEach(x=>{
    object(x,556,ctx=>drawTable(ctx,x,556));solid(x-38,528,76,34);
    [[x-60,555],[x+60,555],[x,612]].forEach(([xx,yy])=>object(xx,yy,ctx=>drawStool(ctx,xx,yy)));
  });
  // Low lounge set in the back corners, leaving a wide center aisle.
  bench(416,704); bench(848,704);
  pumpkin(348,742,1.1,true); pumpkin(375,752,.75); pumpkin(930,743,1.05);
  lamp(294,750); lamp(984,750);
  line(c,[[320,452],[640,478],[960,452]],'#5b6348',2);
  for(let i=0;i<17;i++) {
    const x=320+i*40,y=452+Math.sin(i/16*Math.PI)*26;
    rect(c,x-3,y,6,9,i%3===0?'#f1c574':'#f4dbaa');rect(c,x-1,y-3,2,3,'#4e634a');
  }
  // Rug and flower planters make the deck feel used rather than empty.
  rect(c,562,530,156,126,'#7f775b');rect(c,570,538,140,110,'#a1a17a');
  for(let i=0;i<7;i++) rect(c,572+i*20,545,8,96,'#b6b68b');
  object(338,438,ctx=>{rect(ctx,325,415,26,23,'#9d6b50');drawBush(ctx,338,418,38);});
  object(940,438,ctx=>{rect(ctx,927,415,26,23,'#9d6b50');drawBush(ctx,940,418,38);});

  // Arcade is another roofless social room: a striped canopy above the machines.
  planks(c,1568,352,512,384);
  rect(c,1580,364,488,356,'#7c7373');
  for(let yy=380;yy<710;yy+=48) for(let xx=1596;xx<2052;xx+=48) {
    rect(c,xx,yy,46,46,((xx+yy)/48)%2?'#847b7c':'#8a807c');
    rect(c,xx+18,yy+18,9,9,'#a69b87');
  }
  object(1824,448,(ctx,t=0)=>{
    rect(ctx,1558,307,532,24,'#474f55');
    for(let i=0;i<14;i++) {rect(ctx,1558+i*38,330,38,28,i%2?'#dfbc88':'#826d82');rect(ctx,1558+i*38,356,38,10,i%2?'#cba578':'#685c6d');}
    rect(ctx,1697,279,254,35,'#3d5551');label(ctx,'PLAY CLUB',1824,297,21);
    const colors=['#b77557','#657d81','#91815d','#8a6c82','#b27d52'];
    for(let i=0;i<5;i++) {
      const x=1600+i*92;
      rect(ctx,x,378,64,70,'#3a4748');rect(ctx,x+5,374,54,67,colors[i]);
      rect(ctx,x+4,365,56,15,'#424f4d');rect(ctx,x+10,369,44,6,'#d9be84');
      rect(ctx,x+10,386,44,29,'#273e3d');rect(ctx,x+14,390,36,21,i%2?'#85aa9a':'#d0b584');
      for(let j=0;j<3;j++) {rect(ctx,x+19+j*9,397,5,5,'#426d64');rect(ctx,x+18+j*9,404,5,3,'#efcf94');}
      rect(ctx,x+5,418,54,10,'#424b49');rect(ctx,x+15,416,4,9,'#d8a26c');rect(ctx,x+36,420,7,4,'#ce7d59');rect(ctx,x+47,420,6,4,'#d6bc82');
      rect(ctx,x+28,434,15,3,'#344441');
    }
  });
  solid(1600,380,436,68);
  bench(1624,654,true); bench(2024,654,true);
  // Ping-pong centerpiece: legs, net, paddles, and one ball.
  object(1824,622,ctx=>{
    shadow(ctx,1824,621,89,22);rect(ctx,1750,593,7,34,'#394f45');rect(ctx,1891,593,7,34,'#394f45');
    rect(ctx,1736,541,176,65,'#415d57');rect(ctx,1740,539,168,60,'#6e9682');
    line(ctx,[[1745,543],[1903,543],[1903,594],[1745,594],[1745,543]],'#d7d9b2',2);
    line(ctx,[[1824,542],[1824,596]],'#d7d9b2',2);rect(ctx,1818,529,5,70,'#eee2bd');rect(ctx,1823,532,3,62,'#40584b');
    rect(ctx,1764,555,10,13,'#c37d5a');rect(ctx,1767,568,4,8,'#ddc190');rect(ctx,1866,578,5,5,'#f6e6c5');
  });solid(1736,552,176,60);
  pumpkin(1585,716,1,true); pumpkin(2065,717,.9); lamp(1540,728);lamp(2108,728);

  // Race circuit: rounded asphalt loop with a grass infield; the whole track is walkable.
  rounded(c,1494,1174,628,404,160,'#c5b087');
  rounded(c,1504,1184,608,384,152,'#59615c');
  rounded(c,1568,1248,480,256,104,'#d8c39a');
  rounded(c,1577,1257,462,238,96,'#879a6d');
  c.strokeStyle='#e1cb9e';c.lineWidth=3;c.setLineDash([20,18]);c.beginPath();c.roundRect(1536,1216,544,320,130);c.stroke();c.setLineDash([]);
  for(let y=1290;y<1462;y+=16) for(let x=1506;x<1567;x+=16) rect(c,x,y,15,15,((x-1506+y-1290)/16)%2?'#ddd1ac':'#3f504a');
  label(c,'SENJA CIRCUIT',1808,1334,19,'#52684f');label(c,'READY, SET, WANDER.',1808,1363,11,'#5c714f');
  // Parked tiny cars sit in the infield, away from racing and walking lanes.
  ([[1726,1424,'#c67f51'],[1806,1424,'#e2bd71'],[1886,1424,'#737e8b']] as [number,number,string][]).forEach(([x,y,col])=>object(x,y,ctx=>{
    shadow(ctx,x,y,26,9);rect(ctx,x-26,y-21,52,22,'#3d5148');rect(ctx,x-26,y-25,52,20,col);
    rect(ctx,x-14,y-35,28,18,col);rect(ctx,x-10,y-32,20,10,'#b4c6b0');rect(ctx,x-21,y-5,10,9,'#344b42');rect(ctx,x+12,y-5,10,9,'#344b42');
    rect(ctx,x-27,y-20,5,7,'#f4d6a1');rect(ctx,x+23,y-20,5,7,'#d7aa74');
  }));
  lamp(1514,1160);lamp(2100,1160);lamp(1514,1604);lamp(2100,1604);

  // Mini football pitch: soft striped turf, a tiny goal, room to run in from either end.
  rect(c,952,1272,336,464,'#d6c29a');
  rect(c,960,1280,320,448,'#62886a');
  for(let i=0;i<7;i++)rect(c,960,1280+i*64,320,64,i%2?'#729473':'#658969');
  c.strokeStyle='#dce0b7';c.lineWidth=3;c.strokeRect(976,1296,288,416);
  line(c,[[976,1504],[1264,1504]],'#dce0b7',3);
  c.beginPath();c.arc(1120,1504,48,0,Math.PI*2);c.stroke();
  c.strokeRect(1040,1296,160,80);c.strokeRect(1040,1632,160,80);
  ellipse(c,1120,1504,4,4,'#e4e3bc');
  object(1120,1311,ctx=>{
    rect(ctx,1064,1257,5,55,'#e4d9b2');rect(ctx,1171,1257,5,55,'#e4d9b2');rect(ctx,1064,1257,112,5,'#e4d9b2');
    for(let x=1076;x<1172;x+=12)rect(ctx,x,1262,1,47,'#c9d3af');for(let y=1272;y<1309;y+=12)rect(ctx,1068,y,104,1,'#c9d3af');
  });
  object(1120,1560,ctx=>{shadow(ctx,1120,1562,9,4);ellipse(ctx,1120,1554,9,9,'#ece4c4');rect(ctx,1116,1550,7,7,'#4c6151');rect(ctx,1125,1555,4,4,'#4c6151');});
  // Spectator bench is outside the boundary and does not interrupt the outer road.
  bench(1334,1460,true); pumpkin(1307,1718,.8);

  // The garden circle is quieter and earthy, with a flickering campfire.
  rounded(c,320,1120,512,512,108,'#859668');
  ellipse(c,576,1382,190,174,'#b1ab78');ellipse(c,576,1382,176,159,'#c5b78a');
  for(let i=0;i<30;i++) {
    const a=i/30*Math.PI*2;rect(c,576+Math.cos(a)*181-7,1382+Math.sin(a)*165-4,14,8,i%2?'#b9ac83':'#a59f79');
  }
  label(c,'KEBUN API',576,1556,16,'#776c4f');
  bench(576,1284);bench(436,1410,true);bench(716,1410,true);bench(576,1514);
  object(576,1395,(ctx,t=0)=>{
    shadow(ctx,576,1390,47,19);ellipse(ctx,576,1385,43,23,'#7d7b62');ellipse(ctx,576,1382,35,17,'#665f4c');
    for(let i=0;i<10;i++){const a=i/10*Math.PI*2;rect(ctx,576+Math.cos(a)*38-7,1381+Math.sin(a)*19-5,14,10,'#b1af8d');}
    line(ctx,[[552,1388],[601,1370]],'#7c513b',10);line(ctx,[[552,1373],[599,1390]],'#996342',10);
    const flick=Math.sin(t*.01)*3;
    rect(ctx,560,1350+flick,31,31-flick,'#d78548');rect(ctx,567,1334-flick,17,44+flick,'#e8a54e');
    rect(ctx,572,1350,13,28,'#f4c66c');rect(ctx,575,1363,8,17,'#ffe0a0');rect(ctx,568,1326-flick,8,16,'#e7a450');
    for(let i=0;i<3;i++){const yy=1320-((t*.013+i*17)%45);rect(ctx,570+i*6,yy,3,3,'#dcc19a');}
  });solid(541,1350,70,46);
  pumpkin(387,1280,1.2,true);pumpkin(410,1290,.75);pumpkin(754,1504,1.1);
  lamp(352,1150);lamp(800,1150);lamp(352,1600);lamp(800,1600);
  // Flower beds and a small garden shed / potting surface along the edges.
  [[368,1172],[740,1172],[364,1550],[750,1550]].forEach(([x,y])=>{
    rect(c,x-31,y-23,62,46,'#716e4e');rect(c,x-27,y-19,54,38,'#9a8560');
    for(let i=0;i<6;i++){const xx=x-20+(i%3)*20,yy=y-12+Math.floor(i/3)*18;rect(c,xx,yy,3,14,'#526a45');pixelLeaf(c,xx-3,yy,'#d3a65e',3);}
  });

  // Riverside resting dock, attached to the route and facing the bridge.
  planks(c,2056,880,120,160);bench(2108,1048,false,12);pumpkin(2066,1030,.7);lamp(2130,865);
  object(2106,896,ctx=>{rect(ctx,2097,852,8,42,'#705b44');rect(ctx,2055,833,94,28,'#e3c79a');rect(ctx,2059,837,86,20,'#c7aa79');label(ctx,'TEPI SUNGAI',2102,848,10,'#5b674d');});

  // A few small scenes give the open meadow a lived-in scale.
  const blanket=(x,y)=>{
    rect(c,x-52,y-30,104,60,'#bb8062');
    for(let yy=-24;yy<28;yy+=12)for(let xx=-46;xx<50;xx+=12)rect(c,x+xx,y+yy,10,10,((xx+yy)/12)%2?'#d1a17c':'#b8755b');
    object(x+24,y-4,ctx=>{rect(ctx,x+4,y-25,35,22,'#c79d60');rect(ctx,x+8,y-29,27,8,'#dfb978');rect(ctx,x+15,y-34,12,7,'#a78352');rect(ctx,x-24,y-10,12,6,C.cream);});
  };
  blanket(1216,464); blanket(1640,1010);
  // Deliberately keep tree trunks and dense planting away from lane centers.
  const trees=[
    [208,270,150,2],[410,232,136,0],[578,230,137,1],[777,235,138,2],[968,262,154,0],
    [1115,330,132,2],[1370,280,147,0],[1490,360,125,1],[1744,250,144,0],[1940,254,150,2],[2118,324,134,0],
    [210,526,141,0],[200,748,147,1],[362,856,130,2],[944,870,125,0],
    [1083,558,132,0],[1427,536,147,1],[1555,868,110,2],[1991,851,115,0],
    [292,1004,142,1],[432,1039,122,0],[773,1024,137,2],[923,1131,119,1],
    [250,1203,141,0],[244,1452,147,2],[310,1700,143,1],[485,1702,137,0],[705,1736,135,2],
    [885,1845,141,0],[1141,1850,148,1],[1376,1789,143,0],[1454,1465,119,2],
    [1431,1297,116,1],[1685,1813,141,2],[1910,1818,149,0],[2131,1753,132,1],
    [2040,1107,104,0],[2192,295,110,2],[2189,523,108,1],[2192,778,109,0],
    [2190,1152,112,2],[2191,1573,110,0],
  ];
  trees.forEach(t=>tree(t[0],t[1],t[2],t[3]));
  // Dense tree silhouettes frame the map without becoming a hard rectangular wall.
  for(let x=48;x<2210;x+=112) { tree(x,100+r()*58,125+r()*30,Math.floor(r()*4)); tree(x+20,1900+r()*18,125+r()*32,Math.floor(r()*4)); }
  for(let y=230;y<1820;y+=128) { tree(64+r()*45,y,136+r()*28,Math.floor(r()*4));tree(2504+r()*18,y,140+r()*22,Math.floor(r()*4)); }
  for(let x=2448;x<2560;x+=88){tree(x,150,134,0);tree(x,1880,144,2);}
  [[296,832,50],[419,817,62],[861,818,66],[1032,672,54],[1456,672,64],[1560,1040,60],[2024,794,64],
   [298,1100,68],[792,1100,62],[836,1517,56],[978,1802,64],[1307,1808,56],[1967,1665,64],[2147,1668,56]].forEach(b=>bush(b[0],b[1],b[2]));
  // Fallen leaves, mushrooms and stepping stones cluster in quiet grass only.
  const clusters=[[260,900],[940,1000],[1400,450],[1460,1620],[1940,1060],[380,1760],[1000,260],[2010,290],[800,1170],[2020,1760]];
  clusters.forEach(([x,y])=>{
    for(let i=0;i<22;i++)pixelLeaf(c,x+(r()-.5)*118,y+(r()-.5)*80,['#c1a26b','#b98352','#d6ad70'][Math.floor(r()*3)],2);
    for(let i=0;i<3;i++){const xx=x+(r()-.5)*70,yy=y+(r()-.5)*42;rect(c,xx,yy,4,8,'#ded0a4');rect(c,xx-4,yy-4,12,5,'#b77251');rect(c,xx-1,yy-4,3,2,'#eed1a1');}
  });
  // Small signposts are useful landmarks without covering the walkable lanes.
  ([[982,1010,'← WARKOP'],[1548,1010,'PLAY CLUB →'],[886,1280,'KEBUN ←'],[1355,1227,'LAPANGAN ↓'],[1960,1627,'← SENJA']] as [number,number,string][]).forEach(([x,y,text])=>{
    object(x,y,ctx=>{rect(ctx,x-3,y-47,6,49,C.woodDark);rect(ctx,x-45,y-61,90,22,'#dfbf89');rect(ctx,x-43,y-59,86,18,'#e6c99a');label(ctx,text,x,y-49,10,'#58634b');});solid(x-4,y-5,8,9);
  });
  return { background, foregroundObjects, solids, width:W, height:H };
}

// Chunky 3-pixel shapes, natural feet anchor, and leg motion remain readable at 1×.
export function drawCharacter(ctx,{x,y,color='#d49057',name='',isPlayer=false,facing='down',walking=false,seated=false},time=0) {
  const step=walking?Math.sin(time*.016):0, bob=walking?Math.abs(step)*1.5:0;
  const px=Math.round(x),py=Math.round(y-bob)-(seated?5:0);
  const dir=facing;
  shadow(ctx,px,y,12,4);
  if(isPlayer) {ctx.strokeStyle='#f5dfad';ctx.lineWidth=2;ctx.beginPath();ctx.ellipse(px,y+1,16,7,0,0,Math.PI*2);ctx.stroke();}
  // Shoes and trousers.
  const left=walking?Math.round(step*3):0,right=-left;
  rect(ctx,px-8,py-10,6,8+left,'#435351');rect(ctx,px+2,py-10,6,8+right,'#435351');
  rect(ctx,px-10,py-3+left,8,4,'#ede1bd');rect(ctx,px+2,py-3+right,8,4,'#ede1bd');
  // Oversized jumper / jacket, with two light cuffs.
  rect(ctx,px-10,py-24,20,16,color);rect(ctx,px-13,py-22+right,4,12,color);rect(ctx,px+9,py-22+left,4,12,color);
  rect(ctx,px-13,py-12+right,4,4,'#dfae83');rect(ctx,px+9,py-12+left,4,4,'#dfae83');
  rect(ctx,px-5,py-23,10,3,'#ffffff27');rect(ctx,px-9,py-10,18,3,'#263b3a22');
  // Head and hair. Side-facing characters get a nose and a single visible eye.
  rect(ctx,px-8,py-37,16,14,'#dfb18b');rect(ctx,px-6,py-24,12,3,'#cd9d78');
  rect(ctx,px-9,py-39,18,7,'#48504b');rect(ctx,px-11,py-36,4,10,'#48504b');rect(ctx,px+7,py-36,4,7,'#48504b');
  rect(ctx,px-5,py-41,11,4,'#48504b');rect(ctx,px-6,py-37,10,3,'#68705a');
  if(dir==='up') {rect(ctx,px-8,py-33,16,10,'#48504b');rect(ctx,px-5,py-25,10,2,'#57614f');}
  else if(dir==='left') {rect(ctx,px-10,py-31,4,6,'#dfb18b');rect(ctx,px-6,py-31,2,3,'#384c44');rect(ctx,px+4,py-33,5,9,'#48504b');}
  else if(dir==='right') {rect(ctx,px+6,py-31,4,6,'#dfb18b');rect(ctx,px+4,py-31,2,3,'#384c44');rect(ctx,px-9,py-33,5,9,'#48504b');}
  else {rect(ctx,px-5,py-31,2,3,'#384c44');rect(ctx,px+4,py-31,2,3,'#384c44');rect(ctx,px-1,py-25,4,2,'#a66f58');}
  if(isPlayer) {rect(ctx,px-10,py-36,20,4,'#d6a860');rect(ctx,px-7,py-42,14,7,'#d6a860');rect(ctx,px-6,py-43,12,3,'#ebc47c');}
  if(name) {
    ctx.save();ctx.font='600 11px ui-monospace, SFMono-Regular, Menlo, monospace';
    const textWidth=ctx.measureText(name).width;
    rounded(ctx,px-textWidth/2-7,py-63,textWidth+14,18,5,isPlayer?'#405b4ae6':'#3b5146cf');
    ctx.textAlign='center';ctx.textBaseline='middle';ctx.fillStyle='#f2e2bf';ctx.fillText(name,px,py-54);ctx.restore();
  }
}
