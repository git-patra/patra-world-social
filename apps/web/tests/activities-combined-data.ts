import {LocalRace} from '../src/features/race/local';
import {LocalPenalty} from '../src/features/penalty/local';
import {LocalPingpong} from '../src/features/pingpong/local';
import {LocalBlaster} from '../src/features/blaster/local';
import {vehiclePool} from '../src/activities/vehicles';
import {footballSpawn} from '../src/activities/football';
import {clawSpawn} from '../src/activities/claw';
import {activityContract} from '../src/activities/contracts';
import type {ActivityState,ActivityGame} from '../src/activities/store';
import type {Avatar} from '../src/world/avatars';
const actor=(i:number)=>({actorId:`human-${i}`,actorType:'human',userId:`fixture-${i}`,displayName:`Player ${i+1}`,avatarId:'pumpkin'});
/** Public constructor snapshots, animated within bounded presentation coordinates. No auth or engine internals. */
export function combinedFixture(scenario:'A'|'B',at:number,cycle:number,spectatorQuick=false):{state:ActivityState;avatars:Avatar[];entities:Record<string,number>}{
 const phase=at/1000, id=(kind:string)=>`fixture-${cycle}-${kind}`;
 const race=new LocalRace(id('race'),Array.from({length:5},(_,i)=>actor(i)),0).snapshot();
 race.phase='racing';race.serverTimeMs=at;race.karts=race.karts.map((k,i)=>({...k,x:k.x+Math.round(Math.sin(phase+i)*50),speed:100}));
 const penalty=new LocalPenalty(id('penalty'),[actor(5),actor(6)],at,()=>0).snapshot();
 const blaster=new LocalBlaster(id('blaster'),[7,8,9,10].map(actor),at).snapshot();
 blaster.players=blaster.players.map((p,i)=>({...p,x:.35+i*.1+Math.sin(phase+i)*.04,aim:phase%Math.PI}));
 // Bounded public projectile and bot workload, independent of network authority.
 blaster.bots=Array.from({length:8},(_,i)=>({id:i,x:.1+i*.1,y:.25,nextFire:at+100}));
 blaster.projectiles=Array.from({length:48},(_,i)=>({id:i,x:.15+(i%4)*.2,y:.2+(Math.sin(phase*2+i)+1)*.3,vx:1,vy:0,owner:i%4,ttl:1}));
 const pingpong=new LocalPingpong(id('pingpong'),[actor(11),actor(12)],at).snapshot();
 pingpong.ball={x:.5+Math.sin(phase*2)*.4,y:.5+Math.cos(phase)*.35,vx:.5,vy:.2};pingpong.paddles=[.5+Math.sin(phase)*.3,.5+Math.cos(phase)*.3];
 const games:ActivityGame[]=[{kind:'blaster',matchId:blaster.matchId,snapshot:blaster},{kind:'pingpong',matchId:pingpong.matchId,snapshot:pingpong}];
 if(scenario==='A')games.push({kind:'race',matchId:race.matchId,snapshot:race},{kind:'penalty',matchId:penalty.matchId,snapshot:penalty});
 const vehicles=scenario==='B'?vehiclePool().map((v,i)=>({...v,riderId:`fixture-${i}`,heading:Math.sin(phase)*90})):[];
 const state:ActivityState={games,areas:[],fishing:activityContract.fishingSpots.map((s,i)=>({id:s.id,generation:cycle+1,fishing:{ownerId:`fixture-${14+i}`,ownerName:`Player ${15+i}`,phase:'bite',castId:cycle+1,biteAt:at-100,endsAt:at+1000,catchId:''}})),claw:{...clawSpawn(),operatorId:'fixture-13',operatorName:'Player 14',phase:'aiming',leaseUntil:at+1000,x:.5+Math.sin(phase)*.3},football:footballSpawn(scenario==='B'),vehicles,nearbyFishing:null,nearbyClaw:false,nearbyFootball:false,nearbyRide:null,active:null,spectating:null,inputMode:'walk',lobbies:[],selfId:'observer',online:true,connected:true,now:at,message:'Synthetic public snapshot fixture'};
 // Keep the historical100ms fixture unchanged. Only the new spectator quick
 // workload supplies the OPEN public lease needed to show casual free riders.
 if(spectatorQuick&&scenario==='B')state.areas=[{worldId:'fixture',areaId:'race',incarnation:`fixture-${cycle}`,generation:0,serverTimeMs:Math.floor(at),tick:Math.floor(at/50),kind:'area',payload:{phase:'OPEN',matchId:'',startsAt:0,clearAt:0,resetAt:0}}];
 const avatars:Avatar[]=Array.from({length:20},(_,i)=>({id:`fixture-${i}`,name:`Player ${i+1}`,avatarId:'pumpkin',x:33+i%5+Math.sin(phase+i)*.2,y:29+Math.floor(i/5)+Math.cos(phase+i)*.2,status:'available',seq:0}));
 return{state,avatars,entities:{avatars:20,raceKarts:scenario==='A'?5:0,penaltyPlayers:scenario==='A'?2:0,freeRiders:vehicles.length,freeBall:scenario==='B'?1:0,blasterPlayers:4,blasterBots:blaster.bots.length,projectiles:48,pingpongPlayers:2,clawOperators:1,fishingOwners:4,footballUsers:scenario==='B'?3:0,spectators:2}};
}
