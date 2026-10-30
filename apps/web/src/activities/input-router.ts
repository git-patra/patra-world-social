export type InputMode='walk'|'ride'|'race'|'penalty'|'pingpong'|'claw'|'blaster'|'fishing'|'modal';
export type ActivityInput={x:number;y:number;throttle:number;brake:number;steer:number;aim?:number;fire?:boolean};
type ControlKey='throttle'|'brake'|'steer';
const neutral=():ActivityInput=>({x:0,y:0,throttle:0,brake:0,steer:0});
const directional=(x:number,y:number):ActivityInput=>({x:Math.sign(x),y:Math.sign(y),throttle:Number(y<0),brake:Number(y>0),steer:Math.sign(x)});

/** All keyboard/touch producers route to one owner; changing context releases held intent. */
export class InputRouter {
 private handlers=new Map<InputMode,(input:ActivityInput)=>void>();
 private owner:InputMode='walk';
 private blocks=new Set<string>();
 private held=neutral();
 private keyboard=neutral();
 private directed=neutral();
 private buttons={throttle:0,brake:0,steer:0};
 private pointers=new Map<number,{key:ControlKey;value:number}>();
 private blasterPointers=new Map<number,{role:'move'|'aim';x:number;y:number}>();
 private aim=0;private mouseFire=false;private keyboardFire=false;
 private clearKeys:(()=>void)|null=null;

 register(mode:InputMode,fn:(input:ActivityInput)=>void){
  this.handlers.set(mode,fn);
  return()=>{
   if(this.handlers.get(mode)!==fn)return;
   if(this.owner===mode)this.release();
   this.handlers.delete(mode);
  };
 }
 setOwner(mode:InputMode){
  if(mode===this.owner)return;
  this.release();
  this.owner=mode;
  this.release();
 }
 mode(){return this.blocks.size?'modal':this.owner}
 block(reason:string,value:boolean){
  if(value){
   if(!this.blocks.has(reason))this.release();
   this.blocks.add(reason);
  }else this.blocks.delete(reason);
 }
 direction(x:number,y:number){
  if(this.blocks.size)return;
  this.directed=directional(x,y);
  this.update();
 }
 control(key:ControlKey,value:number){
  if(this.blocks.size)return;
  this.buttons[key]=value;
  this.update();
 }
 pointerControl(pointerId:number,key:ControlKey,value:number){
  if(this.blocks.size)return;
  this.pointers.set(pointerId,{key,value});
  this.update();
 }
 blasterPointer(id:number,role:'move'|'aim',x:number,y:number,continuation=false){if(this.blocks.size||this.owner!=='blaster'||continuation&&!this.blasterPointers.has(id))return;if([...this.blasterPointers.entries()].some(([other,p])=>other!==id&&p.role===role))return;this.blasterPointers.set(id,{role,x,y});this.update()}
 blasterAim(aim:number,fire?:boolean){if(this.blocks.size||this.owner!=='blaster')return;this.aim=aim;if(fire!==undefined)this.mouseFire=fire;this.update()}
 releaseBlasterMouse(){this.mouseFire=false;this.update()}
 releasePointer(pointerId:number){
  if(this.blasterPointers.delete(pointerId))this.update();
  if(this.pointers.delete(pointerId))this.update();
 }
 private update(){
  const next={...this.directed,x:Math.sign(this.directed.x+this.keyboard.x),y:Math.sign(this.directed.y+this.keyboard.y)};
  for(const key of ['throttle','brake','steer'] as const){
   let value=this.directed[key]+this.keyboard[key]+this.buttons[key];
   for(const pointer of this.pointers.values())if(pointer.key===key)value+=pointer.value;
   next[key]=Math.max(key==='steer'?-1:0,Math.min(1,value));
  }
  if(this.owner==='blaster'){
   for(const p of this.blasterPointers.values()){if(p.role==='move'){next.x=p.x;next.y=p.y}else if(Math.hypot(p.x,p.y)>.1)this.aim=Math.atan2(p.y,p.x)}
   next.aim=this.aim;next.fire=this.mouseFire||this.keyboardFire||[...this.blasterPointers.values()].some(p=>p.role==='aim'&&Math.hypot(p.x,p.y)>.1);
  }
  this.held=next;
  if(!this.blocks.size)this.handlers.get(this.owner)?.({...this.held});
 }
 private nonNeutral(){return Object.values(this.held).some(value=>value!==0)}
 repeat(){
  if(!this.blocks.size&&this.nonNeutral())this.handlers.get(this.owner)?.({...this.held});
 }
 release(){
  this.clearKeys?.();
  this.keyboard=neutral();
  this.directed=neutral();
  this.buttons={throttle:0,brake:0,steer:0};
  this.pointers.clear();this.blasterPointers.clear();this.mouseFire=false;this.keyboardFire=false;
  this.held=neutral();if(this.owner==='blaster'){this.held.aim=this.aim;this.held.fire=false}
  this.handlers.get(this.owner)?.({...this.held});
 }
 install(){
  const pressed=new Set<string>();
  const directions:Record<string,[number,number]>={w:[0,-1],a:[-1,0],s:[0,1],d:[1,0],ArrowUp:[0,-1],ArrowLeft:[-1,0],ArrowDown:[0,1],ArrowRight:[1,0]};
  const resetKeys=()=>pressed.clear();
  this.clearKeys=resetKeys;
  const clear=()=>this.release();
  const pointer=(event:PointerEvent)=>this.releasePointer(event.pointerId);
  const text=(target:EventTarget|null)=>target instanceof Element&&Boolean(target.closest('input,textarea,select,[contenteditable="true"]'));
  const key=(event:KeyboardEvent)=>{
   const name=event.key.length===1?event.key.toLowerCase():event.key;
   if((!directions[name]&&!(this.owner==='blaster'&&name===' '))||text(event.target)||document.body.dataset.controls==='touch'||event.ctrlKey||event.altKey||event.metaKey||this.blocks.size)return;
   if(this.owner==='blaster'&&event.type==='keydown'&&event.repeat&&!pressed.has(name))return;
   event.preventDefault();
   event.stopImmediatePropagation();
   if(event.type==='keydown')pressed.add(name);else pressed.delete(name);
   let x=0,y=0,ax=0,ay=0;
   for(const name of pressed){if(name===' ')continue;if(this.owner==='blaster'&&name.startsWith('Arrow')){ax+=directions[name][0];ay+=directions[name][1]}else{x+=directions[name][0];y+=directions[name][1]}}
   if(this.owner==='blaster'){if(ax||ay)this.aim=Math.atan2(ay,ax);this.keyboardFire=pressed.has(' ')}
   this.keyboard=directional(x,y);
   this.update();
  };
  const visibility=()=>{clear();this.block('hidden',document.hidden)};
  const focus=(event:FocusEvent)=>{
   const editing=text(event.target);
   this.block('text',editing);
   if(editing)clear();
  };
  const modal=()=>{clear();this.block('modal',Boolean(document.querySelector('dialog[open]')))};
  const observer=new MutationObserver(modal);
  observer.observe(document.body,{subtree:true,attributes:true,attributeFilter:['open']});
  window.addEventListener('keydown',key,true);
  window.addEventListener('keyup',key,true);
  window.addEventListener('blur',clear);
  window.addEventListener('pointercancel',pointer);
  window.addEventListener('lostpointercapture',pointer);
  window.addEventListener('pointerup',pointer);
  document.addEventListener('visibilitychange',visibility);
  document.addEventListener('focusin',focus);
  modal();
  visibility();
  return()=>{
   clear();
   if(this.clearKeys===resetKeys)this.clearKeys=null;
   observer.disconnect();
   window.removeEventListener('keydown',key,true);
   window.removeEventListener('keyup',key,true);
   window.removeEventListener('blur',clear);
   window.removeEventListener('pointercancel',pointer);
   window.removeEventListener('lostpointercapture',pointer);
   window.removeEventListener('pointerup',pointer);
   document.removeEventListener('visibilitychange',visibility);
   document.removeEventListener('focusin',focus);
  };
 }
}
export const activityInput=new InputRouter();
