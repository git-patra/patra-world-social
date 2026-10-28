import {authState} from './auth-state';
import {api, type User} from './net/api.ts';
import {passwordLengthError} from './password-validation.js';

type Mode='login'|'register'|'recover';

export function mountAuth(){
 const button=document.querySelector<HTMLButtonElement>('#account-button')!;
 const label=document.querySelector<HTMLElement>('#account-label')!;
 const dialog=document.querySelector<HTMLDialogElement>('#account-dialog')!;
 const body=document.querySelector<HTMLElement>('#account-content')!;
 let user:User|null=authState.snapshot().user;
 const escape=(s:string)=>s.replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
 const renderUser=(next:User|null)=>{user=next;label.textContent=user?`@${user.username}`:'Akun';button.setAttribute('aria-label',user?`Akun ${user.username}`:'Buka akun')};
 authState.subscribe(({user})=>renderUser(user));renderUser(user);
 const initial=authState.snapshot().generation;api.me().then(next=>authState.transition(next,initial)).catch(()=>authState.transition(null,initial));
 function message(text:string){const node=body.querySelector('.auth-message');if(node)node.textContent=text;}
 function form(mode:Mode){
  const title={login:'Masuk ke desa',register:'Buat akun',recover:'Pulihkan akun'}[mode];
  body.innerHTML=`<div class="auth-head"><span>✦ AKUN PENGUNJUNG</span><h2 id="account-title">${title}</h2><p>Jelajahi desa sebagai tamu, atau masuk untuk menyimpan identitasmu.</p></div>
  <div class="auth-tabs"><button type="button" data-auth-mode="login" aria-current="${mode==='login'}">Masuk</button><button type="button" data-auth-mode="register" aria-current="${mode==='register'}">Daftar</button><button type="button" data-auth-mode="recover" aria-current="${mode==='recover'}">Pulihkan</button></div>
  <form id="auth-form"><label>Nama pengguna<input name="username" autocomplete="username" minlength="3" maxlength="24" pattern="[A-Za-z0-9_]+" required></label>${mode==='recover'?'<label>Kode pemulihan<input name="code" autocomplete="off" required></label>':''}<label>${mode==='recover'?'Kata sandi baru':'Kata sandi'}<input name="password" type="password" autocomplete="${mode==='login'?'current-password':'new-password'}" required></label><button class="auth-primary" type="submit">${title}</button><p class="auth-message" role="alert"></p></form>`;
  body.querySelectorAll<HTMLButtonElement>('[data-auth-mode]').forEach(b=>b.onclick=()=>form(b.dataset.authMode as Mode));
  body.querySelector<HTMLFormElement>('#auth-form')!.onsubmit=async event=>{
   event.preventDefault();const f=event.currentTarget as HTMLFormElement;const submit=f.querySelector<HTMLButtonElement>('button[type=submit]')!;submit.disabled=true;message('Memproses...');
   const username=(f.elements.namedItem('username') as HTMLInputElement).value;
   const password=(f.elements.namedItem('password') as HTMLInputElement).value;
   const lengthError=passwordLengthError(password);if(lengthError){message(lengthError);submit.disabled=false;return}
   const generation=authState.begin();
   try{
    if(mode==='login'){const result=await api.login(username,password);if(authState.transition(result.user,generation))dialog.close();}
    if(mode==='register'){const result=await api.register(username,password);if(!authState.is(generation))return;showCode(result.recoveryCode,'Akun dibuat. Simpan kode ini sekarang. Setelah ditutup, kode tidak dapat dilihat lagi.');}
    if(mode==='recover'){const code=(f.elements.namedItem('code') as HTMLInputElement).value;const result=await api.recover(username,code,password);if(!authState.transition(null,generation))return;showCode(result.recoveryCode,'Kata sandi diganti dan semua sesi lama berakhir. Simpan kode baru ini.');}
   }catch(error){if(!authState.is(generation))return;message(error instanceof Error?error.message:'Layanan akun belum tersedia.');submit.disabled=false;}
  };
 }
 function showCode(code:string,text:string){
  body.innerHTML=`<div class="auth-head"><span>✦ SIMPAN SEKALI</span><h2 id="account-title">Kode pemulihan</h2><p>${escape(text)}</p></div><label>Kode rahasia<textarea id="recovery-output" readonly rows="3"></textarea></label><p class="auth-message">Simpan di tempat aman. Kode ini hanya ditampilkan sekali.</p><button class="auth-primary" id="recovery-done">Sudah kusimpan</button>`;
  const output=body.querySelector<HTMLTextAreaElement>('#recovery-output')!;output.value=code;
  body.querySelector<HTMLButtonElement>('#recovery-done')!.onclick=()=>{output.value='';form('login')};
 }
 function account(){
  if(user){body.innerHTML=`<div class="auth-head"><span>✦ WARGA DESA</span><h2 id="account-title">Halo, ${escape(user.displayName)}</h2><p>@${escape(user.username)}</p></div><button class="auth-primary" id="logout-button">Keluar akun</button><p class="auth-message" role="alert"></p>`;body.querySelector<HTMLButtonElement>('#logout-button')!.onclick=async()=>{const generation=authState.begin();try{await api.logout();if(authState.transition(null,generation))form('login')}catch(error){if(!authState.is(generation))return;message(error instanceof Error?error.message:'Layanan akun belum tersedia.')}}}
  else form('login');
  dialog.showModal();
 }
 button.onclick=account;
 window.addEventListener('patra-login-request',()=>{if(!dialog.open)account()});
 dialog.querySelector<HTMLButtonElement>('#close-account')!.onclick=()=>dialog.close();
 dialog.addEventListener('close',()=>{body.innerHTML='';button.focus({preventScroll:true})});
}
