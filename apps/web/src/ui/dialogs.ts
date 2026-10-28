const focusReturns=new WeakMap<HTMLDialogElement,{trigger:Element|null}>();

/** A menu child returns to the visible shell opener after the menu has closed. */
export function setDialogFocusReturn(dialog:HTMLDialogElement,trigger:Element|null){
 const pending=focusReturns.get(dialog);if(pending){pending.trigger=trigger;return}
 const state={trigger};focusReturns.set(dialog,state);
 const restore=()=>{
  dialog.removeEventListener('close',restore);focusReturns.delete(dialog);
  // Closing a parent must not move focus out of a newly opened child modal.
  if(document.querySelector('dialog[open]'))return;
  if(state.trigger?.isConnected)(state.trigger as HTMLElement).focus?.({preventScroll:true});
 };
 dialog.addEventListener('close',restore);
}

/** Native modal focus trapping and Escape stay intact; restore the actual opener. */
export function openCenteredDialog(dialog:HTMLDialogElement,trigger:Element|null=document.activeElement){
 if(dialog.open)return;
 setDialogFocusReturn(dialog,trigger);dialog.showModal();
}

/** A close event runs panel/input cleanup before the next destination frame. */
export function closeDialogThen(dialog:HTMLDialogElement,action:()=>void){
 if(!dialog.open){requestAnimationFrame(action);return}
 const closed=()=>{dialog.removeEventListener('close',closed);requestAnimationFrame(action)};
 dialog.addEventListener('close',closed);dialog.close();
}


/** VisualViewport follows the on-screen keyboard without scrolling the world. */
export function mountDialogViewport(){
 const viewport=window.visualViewport;if(!viewport)return()=>{};
 const update=()=>{document.documentElement.style.setProperty('--dialog-viewport-height',`${viewport.height}px`);document.documentElement.style.setProperty('--dialog-viewport-top',`${viewport.offsetTop}px`)};
 update();viewport.addEventListener('resize',update);viewport.addEventListener('scroll',update);
 return()=>{viewport.removeEventListener('resize',update);viewport.removeEventListener('scroll',update);document.documentElement.style.removeProperty('--dialog-viewport-height');document.documentElement.style.removeProperty('--dialog-viewport-top')};
}
