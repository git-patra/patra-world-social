import {readFileSync} from 'node:fs';
import {OnlineDuckController} from '../src/features/duck/online';
import {clickPresented} from './duck-online-driver';
const input=JSON.parse(readFileSync(0,'utf8'));let local=input.calibrationClientAt;let payload:unknown,echo:unknown;
const controller=new OnlineDuckController(input.snapshot.worldId,input.snapshot.matchId,async(type,p)=>{
 if(type==='duck.clock')return input.challenge;
 if(type==='duck.clock_reply'){echo=p;return input.clockReply}
 if(type==='duck.shoot'){payload=p;return input.resolution??{targetId:0,points:0}}
 throw Error('Unexpected controller command');
},()=>local);
await controller.calibrate();local=input.receivedClientAt;controller.receive(input.snapshot);local=input.clickClientAt;
const view=await clickPresented(controller,input.targetId,input.mode??'center');
let after:unknown;if(input.afterClickClientAt!==undefined){local=input.afterClickClientAt;after=await clickPresented(controller,input.targetId)}
process.stdout.write(JSON.stringify({payload,echo,...view,after}));
