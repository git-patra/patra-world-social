// Source-authored only until an authorized guest browser runner executes it.
// Isolated browser contexts never reuse a person's IndexedDB or credentials.
import {test,expect} from '@playwright/test';
import {build} from 'esbuild';
test('IndexedDB atomic 100-record retention, duplicate receipt, reload and reset',async({page})=>{
 await page.goto('/');
 const bundled=await build({entryPoints:['src/guest/history.ts'],bundle:true,write:false,format:'iife',globalName:'GuestStorage',platform:'browser'});
 await page.addScriptTag({content:bundled.outputFiles[0].text});
 const name=`patra-guest-test-${Date.now()}`;
 const result=await page.evaluate(async name=>{const G=(window as unknown as {GuestStorage:{GuestHistory:new(name:string)=>{append:(r:unknown)=>Promise<void>;list:()=>Promise<{id:string}[]>;saveSettings:(p:unknown)=>Promise<void>}}}).GuestStorage;const h=new G.GuestHistory(name);for(let i=0;i<105;i++)await h.append({id:String(i),gameType:'duck',scope:'device',finishedAt:i,durationMs:60000,version:{mapId:'fixture',mapVersion:1,rulesId:'duck-v1',rulesVersion:1,protocolVersion:1},summary:`Skor ${i}`,detail:{}});await h.append({id:'104',gameType:'duck',scope:'device',finishedAt:999,summary:'duplicate'});await h.saveSettings({displayName:'Local fixture',avatar:'moon',audio:false,controls:'touch',reducedMotion:true});return h.list()},name);
 expect(result).toHaveLength(100);expect(result[0].id).toBe('104');expect(result.at(-1)?.id).toBe('5');
 await page.reload();await page.addScriptTag({content:bundled.outputFiles[0].text});
 const after=await page.evaluate(async name=>{const G=(window as unknown as {GuestStorage:{GuestHistory:new(name:string)=>{list:()=>Promise<unknown[]>;settings:()=>Promise<{displayName:string}>;reset:()=>Promise<void>}}}).GuestStorage;const h=new G.GuestHistory(name);const count=(await h.list()).length,p=await h.settings();await h.reset();const remaining=(await h.list()).length;await new Promise<void>((resolve,reject)=>{const r=indexedDB.deleteDatabase(name);r.onsuccess=()=>resolve();r.onerror=()=>reject(r.error)});return{count,name:p.displayName,remaining}},name);
 expect(after).toEqual({count:100,name:'Local fixture',remaining:0});
});
test('guest finishes all games, persists on reload, never requests socket/score/media, resets device',async({page})=>{
 test.setTimeout(120000);const forbidden:string[]=[];page.on('websocket',s=>forbidden.push(s.url()));page.on('request',r=>{if(/voice.*token|score|realtime/.test(r.url()))forbidden.push(r.url())});await page.goto('/');await page.clock.install();
 await page.getByRole('button',{name:'Main sendiri',exact:true}).click();await page.getByRole('textbox',{name:'Nama lokal / tampilan'}).fill('Guest fixture');await page.getByRole('button',{name:'Simpan pengaturan'}).click();await expect(page.getByRole('status').filter({hasText:'Pengaturan disimpan.'})).toBeVisible();await page.getByRole('button',{name:'Tutup pengaturan'}).click();
 const open=async(id:string)=>{await page.locator('#list-button').click();await page.locator(`#dialog-content [data-project="${id}"]`).click()};
 await open('race');await page.getByRole('button',{name:'Mulai race solo · 5 lap'}).click();await page.clock.fastForward(305000);await expect(page.getByText('FINISH',{exact:true})).toBeVisible();await page.locator('#close-dialog').click();
 await open('penalty');await page.getByRole('button',{name:'Mulai practice vs BOT'}).click();await page.clock.fastForward(95000);await expect(page.getByText('Practice selesai.',{exact:true})).toBeVisible();await page.locator('#close-dialog').click();
 await open('duck');await page.getByRole('button',{name:'Mulai Duck solo · 60 detik'}).click();await page.clock.fastForward(65000);await expect(page.getByText('SELESAI',{exact:true})).toBeVisible();await page.locator('#close-dialog').click();
 await page.locator('#history-button').click();await expect(page.locator('#history-content details')).toHaveCount(3);await page.reload();await page.locator('#history-button').click();await expect(page.locator('#history-content details')).toHaveCount(3);expect(forbidden).toEqual([]);await page.locator('#close-history').click();
 await page.locator('#settings-button').click();await page.getByText('Hapus data lokal',{exact:true}).click();await page.getByRole('button',{name:'Konfirmasi hapus data perangkat'}).click();await expect(page.getByText('Data perangkat dihapus.',{exact:true})).toBeVisible();await page.getByRole('button',{name:'Tutup pengaturan'}).click();await page.locator('#history-button').click();await expect(page.getByText('Belum ada hasil lokal tersimpan.')).toBeVisible();
});
