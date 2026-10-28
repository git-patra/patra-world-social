import data from '../../../contracts/map.v1.json';
import test from 'node:test';
import assert from 'node:assert/strict';
import {VoiceSession,mediaError,type MediaRoom} from '../src/features/voice/session';
import {InTable,TABLES} from '../src/features/voice/zones';
test('voice joins listen-only and releases old connection before next admission',async()=>{
 const calls:string[]=[];let rooms=0;
 const fake=():MediaRoom=>({connect:async()=>{calls.push('connect')},disconnect:async()=>{calls.push('disconnect')},mic:async(on)=>{calls.push(`mic:${on}`)}});
 const s=new VoiceSession(async(type)=>{calls.push(type);return type==='voice.token'?{url:'wss://example',token:'opaque'}:{id:'seat'}},async()=>{rooms++;return fake()});
 await s.join(0);assert.equal(rooms,1);assert.ok(!calls.some(x=>x.startsWith('mic:')));
 await s.join(1);assert.ok(calls.indexOf('disconnect')<calls.lastIndexOf('voice.join'));
 await s.leave();assert.equal(calls.at(-1),'voice.leave');
});
test('cancelled pending admission never connects and releases server seat',async()=>{
 let resolve:(x:unknown)=>void=()=>{};let connected=0;const calls:string[]=[];
 const s=new VoiceSession(async(type)=>{calls.push(type);if(type==='voice.join')return new Promise(r=>{resolve=r});return{}},async()=>({connect:async()=>{connected++},disconnect:async()=>{},mic:async()=>{}}));
 const joining=s.join(0);await new Promise(r=>setTimeout(r,0));const leaving=s.leave();resolve({id:'seat'});await Promise.all([joining,leaving]);assert.equal(connected,0);assert.equal(calls.at(-1),'voice.leave');
});
test('permission, device and network failures stay distinct',()=>{assert.match(mediaError({name:'NotAllowedError'}),/izin/i);assert.match(mediaError({name:'NotFoundError'}),/perangkat/i);assert.match(mediaError(Error('network')),/jaringan/i)});
test('table geometry is separate and nonoverlapping',()=>{assert.equal(TABLES.mapId,'kampung-senja-v2');assert.equal(TABLES.mapChecksum,data.checksum);for(const t of TABLES.tables){assert.ok(InTable(t.id,t.x,t.y));assert.ok(!InTable(1-t.id,t.x,t.y))}});
test('audio network failure keeps the authorized table chat seat until explicit leave',async()=>{
 const calls:string[]=[];const s=new VoiceSession(async(type)=>{calls.push(type);return type==='voice.token'?{url:'wss://example',token:'opaque'}:{id:'seat'}},async()=>({connect:async()=>{throw Error('network blocked')},disconnect:async()=>{calls.push('disconnect')},mic:async()=>{}}));
 await assert.rejects(s.join(0));assert.ok(!calls.includes('voice.leave'));await s.leave();assert.equal(calls.at(-1),'voice.leave');
});
test('join acknowledgement timeout is cleaned up as a possibly allocated seat',async()=>{
 const calls:string[]=[];const s=new VoiceSession(async(type)=>{calls.push(type);if(type==='voice.join')throw Error('ack timeout');return{}},async()=>{throw Error('must not connect')});await assert.rejects(s.join(0));assert.equal(calls.at(-1),'voice.leave');
});

import {updateMicrophone} from '../src/features/voice/microphone';
test('successful publication remains visibly on when device enumeration fails, and explicit mute still stops it',async()=>{
 let publishing=false,visibleMic=false;const states:boolean[]=[];
 const session=new VoiceSession(async(type)=>type==='voice.token'?{url:'wss://example',token:'opaque'}:{id:'seat'},async()=>({connect:async()=>{},disconnect:async()=>{publishing=false},mic:async(on)=>{publishing=on}}));
 await session.join(0);
 const view={mic:(on:boolean)=>{visibleMic=on;states.push(on)},devices:()=>{},status:()=>{}};
 const unavailableDevices=async():Promise<MediaDeviceInfo[]>=>{throw Error('enumeration unavailable')};
 await updateMicrophone(session,true,'',unavailableDevices,view);
 assert.equal(publishing,true);assert.equal(visibleMic,true);assert.deepEqual(states,[true]);
 await updateMicrophone(session,false,'',unavailableDevices,view);
 assert.equal(publishing,false);assert.equal(visibleMic,false);assert.deepEqual(states,[true,false]);await session.leave();
});

import {VoicePanel} from '../src/features/voice/VoicePanel';
import {componentLifecycle} from './component-lifecycle';
function panelFixture(){
 const saved={window:globalThis.window,document:globalThis.document,fetch:globalThis.fetch,setInterval:globalThis.setInterval,clearInterval:globalThis.clearInterval};
 const events=new EventTarget(),document=Object.assign(new EventTarget(),{hidden:false}),timers=new Map<number,()=>Promise<void>>();let timerID=0;
 Object.assign(globalThis,{window:events,document,fetch:async()=>new Response(JSON.stringify({status:'ready'})),setInterval:(fn:()=>Promise<void>)=>{timers.set(++timerID,fn);return timerID},clearInterval:(id:number)=>timers.delete(id)});
 const component=componentLifecycle(()=>VoicePanel());component.flush();
 const requests:{payload:any;resolve:(value:any)=>void;reject:(error:Error)=>void}[]=[];
 function socket(admissionID:string){const state={admission:{id:admissionID,identity:admissionID,userId:'u',name:'User',table:0,muted:false},participants:[],messages:[],nextMessageSeq:1};return {state,status:'online',selfId:'u',worldId:admissionID,gameCommand:async(type:string,payload:any)=>{if(type==='voice.state')return {...state};if(type==='voice.chat')return new Promise((resolve,reject)=>requests.push({payload,resolve,reject}));return{}}}}
 function find(type:string,node:any=component.tree):any{if(!node)return undefined;if(Array.isArray(node)){for(const child of node){const found=find(type,child);if(found)return found}return}if(node.type===type)return node;return node.props?.children===undefined?undefined:find(type,node.props.children)}
 return {component,requests,socket,find,async attach(value:any){events.dispatchEvent(new CustomEvent('patra-world-attachment',{detail:value}));component.flush();await this.poll()},async poll(){for(const fn of [...timers.values()])await fn();await component.settle()},edit(value:string){find('textarea').props.onChange({target:{value}});component.flush()},submit(){const result=find('form').props.onSubmit({preventDefault(){}});component.flush();return result},dispose(){component.dispose();Object.assign(globalThis,saved)}};
}
test('VoicePanel delayed ACK preserves the edited next draft and uses monotonic admission sequences',async()=>{
 const f=panelFixture();try{const s=f.socket('a'.repeat(48));await f.attach(s);f.edit('draft A');const sending=f.submit();f.edit('draft B');f.requests[0].resolve({});await sending;await f.component.settle();assert.equal(f.find('textarea').props.value,'draft B');assert.equal(f.requests[0].payload.clientMessageId,s.state.admission.id+':1');
  await f.poll();const next=f.submit();assert.equal(f.requests[1].payload.clientMessageId,s.state.admission.id+':2','stale state must not rewind next sequence');f.requests[1].resolve({});await next;await f.component.settle();assert.equal(f.find('textarea').props.value,'');
 }finally{f.dispose()}
});
test('VoicePanel obsolete attachment/admission completions cannot change a new draft or pending submission',async()=>{
 for(const sameSocket of [false,true])for(const reject of [false,true]){const f=panelFixture();try{const original=f.socket('a'.repeat(48));await f.attach(original);f.edit('old');const old=f.submit();if(sameSocket){original.state.admission.id='b'.repeat(48);await f.poll()}else await f.attach(f.socket('b'.repeat(48)));f.edit('new');const current=f.submit();if(reject)f.requests[0].reject(Error('obsolete failure'));else f.requests[0].resolve({});await old;await f.component.settle();assert.equal(f.find('textarea').props.value,'new');assert.equal(f.find('form').props.children[1].props.disabled,true);assert.doesNotMatch(String(f.find('p').props.children),/obsolete failure/);f.requests[1].resolve({});await current;await f.component.settle();assert.equal(f.find('textarea').props.value,'');}finally{f.dispose()}}
});
test('VoicePanel explicit retry keeps submitted ID/text; expired retry resyncs without reposting edited draft',async()=>{
 const f=panelFixture();try{const s=f.socket('c'.repeat(48));await f.attach(s);f.edit('A');const first=f.submit();f.edit('B');f.requests[0].reject(Error('timeout'));await first;await f.component.settle();const retry=f.submit();assert.deepEqual(f.requests[1].payload,f.requests[0].payload);s.state.nextMessageSeq=40;f.requests[1].reject(Object.assign(Error('already processed'),{code:'voice_retry_expired'}));await retry;await f.component.settle();assert.equal(f.find('textarea').props.value,'B');assert.equal(f.requests.length,2,'resync must not automatically repost');assert.match(String(f.find('p').props.children),/sudah diproses/);const next=f.submit();assert.deepEqual(f.requests[2].payload,{clientMessageId:s.state.admission.id+':40',text:'B'});f.requests[2].resolve({});await next;}finally{f.dispose()}
});

import {validateCommand} from '../src/world/avatars';
test('voice wire contract accepts only canonical admission sequence IDs in JS safe range',()=>{
 const prefix='a'.repeat(48)+':';const command=(id:string)=>({v:2,commandId:'c',seq:1,worldId:'w',type:'voice.chat',payload:{clientMessageId:id,text:'text'}});
 for(const suffix of ['1','9007199254740991'])assert.equal(validateCommand(command(prefix+suffix)).type,'voice.chat');
 for(const id of ['random',prefix+'0',prefix+'01',prefix+'+1',prefix+'1e0',prefix+'9007199254740992','A'.repeat(48)+':1'])assert.throws(()=>validateCommand(command(id)));
});
