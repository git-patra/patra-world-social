// AUTH_BROWSER_NOT_RUN: authored only. Execute with approved disposable accounts
// in an isolated acceptance environment; never inject sessions into live preview.
import {test,expect,type BrowserContext,type Page} from '@playwright/test';
import {randomBytes} from 'node:crypto';
async function account(context:BrowserContext,name:string,password:string):Promise<Page>{
 const p=await context.newPage();await p.goto('/');await p.locator('#account-button').click();await p.getByRole('button',{name:'Daftar',exact:true}).click();await p.getByLabel('Nama pengguna').fill(name);await p.getByLabel('Kata sandi',{exact:true}).fill(password);await p.getByRole('button',{name:'Buat akun',exact:true}).click();await p.getByRole('button',{name:'Sudah kusimpan'}).click();await p.getByLabel('Nama pengguna').fill(name);await p.getByLabel('Kata sandi',{exact:true}).fill(password);await p.getByRole('button',{name:'Masuk ke desa',exact:true}).click();await expect(p.locator('#account-label')).toHaveText(`@${name}`);return p;
}
test('committed first Duck PB appears once for two world members, never another world, with private history',async({browser})=>{
 test.setTimeout(150000);const suffix=randomBytes(5).toString('hex'),password=`Isolated!${randomBytes(20).toString('base64url')}`;const contexts=await Promise.all([browser.newContext(),browser.newContext(),browser.newContext()]);
 try{
 const [owner,member,other]=await Promise.all(contexts.map((c,i)=>account(c,`podium${i}_${suffix}`,password)));
 await owner.locator('#worlds-button').click();await owner.getByLabel('Nama dunia').fill('Podium fixture');await owner.getByRole('button',{name:'Buat dunia',exact:true}).click();await owner.getByLabel('Username tujuan (opsional)').fill(`podium1_${suffix}`);await owner.getByRole('button',{name:'Buat tautan 24 jam'}).click();const link=await owner.locator('#invite-link input').inputValue();await member.goto(link);await member.getByRole('button',{name:'Terima undangan',exact:true}).click();
 for(const p of[owner,member]){await p.locator('#join-world').click();await expect(p.locator('.online-status')).toContainText('Online');}
 await other.locator('#worlds-button').click();await other.getByLabel('Nama dunia').fill('Other fixture');await other.getByRole('button',{name:'Buat dunia',exact:true}).click();await other.locator('#join-world').click();await expect(other.locator('.online-status')).toContainText('Online');
 for(const p of[member,other]){await p.locator('#podium-button').click();await expect(p.locator('.podium li')).toHaveCount(3);await expect(p.locator('.podium')).toContainText('Belum ada skor sah');}
 await owner.getByRole('button',{name:/Duck Arcade/}).first().click();const cabinet=owner.locator('.lobby-panel article').filter({has:owner.getByRole('heading',{name:'Kabinet 1',exact:true})});await cabinet.getByRole('button',{name:'Ikut permainan',exact:true}).click();await cabinet.getByRole('button',{name:'Ready',exact:true}).click();await cabinet.getByRole('button',{name:'Mulai',exact:true}).click();
 // Natural server completion with zero shots is a legitimate first PB; no score upload.
 await expect(member.locator('.podium')).toContainText(`podium0_${suffix}`,{timeout:70000});await expect(member.locator('.podium a')).toHaveCount(1);await expect(other.locator('.podium')).not.toContainText(`podium0_${suffix}`);
 await owner.getByRole('button',{name:'Tutup lokasi'}).click();await owner.locator('#podium-button').click();await expect(owner.locator('.podium a')).toHaveCount(1);const detail=await member.locator('.podium a').getAttribute('href');await member.locator('.podium a').click();await expect(member.getByRole('heading',{name:'Ringkasan hasil dunia'})).toBeVisible();
 const resultId=detail!.split('/').at(-1)!;expect((await contexts[1].request.get(`/api/v1/history/${resultId}`)).status()).toBe(404);expect((await contexts[0].request.get(`/api/v1/history/${resultId}`)).status()).toBe(200);
 await member.reload();await expect(member.locator('.podium a')).toHaveCount(1);await other.goto('/'+detail);await expect(other.locator('#podium-dialog')).toContainText('Keanggotaan aktif atau hasil tidak tersedia');
 }finally{for(const c of contexts)await c.close()}
});
