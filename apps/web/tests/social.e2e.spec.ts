// AUTH_BROWSER_NOT_RUN: credential-driven multi-context acceptance authored only.
// Do not run until the user authorizes browser credential flows; config also
// requires an isolated auth_e2e schema and loopback4177, never preview accounts.
import {test,expect,type BrowserContext,type Page} from '@playwright/test';
import {randomBytes} from 'node:crypto';
async function account(context:BrowserContext,username:string,password:string):Promise<Page>{
 const page=await context.newPage();await page.goto('/');await page.locator('#account-button').click();await page.getByRole('button',{name:'Daftar',exact:true}).click();await page.getByLabel('Nama pengguna').fill(username);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Buat akun',exact:true}).click();await expect(page.getByRole('heading',{name:'Kode pemulihan'})).toBeVisible();await page.getByRole('button',{name:'Sudah kusimpan'}).click();await page.getByLabel('Nama pengguna').fill(username);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Masuk ke desa',exact:true}).click();await expect(page.locator('#account-label')).toHaveText(`@${username}`);return page;
}
test('friends invite play flow, scoped chat, typing, unread, reconnect and moderation',async({browser})=>{
 const suffix=randomBytes(4).toString('hex'),password=`LocalTest!${randomBytes(16).toString('base64url')}`;
 const owner=await browser.newContext(),friend=await browser.newContext(),outsider=await browser.newContext();
 try{
  const a=await account(owner,`owner_${suffix}`,password),b=await account(friend,`friend_${suffix}`,password),c=await account(outsider,`other_${suffix}`,password);
  await a.locator('#worlds-button').click();await a.getByLabel('Nama dunia').fill('T6 Pumpkin Grove');await a.getByRole('button',{name:'Buat dunia',exact:true}).click();await a.locator('#close-worlds').click();
  await a.locator('#social-button').click();await a.getByLabel('Cari exact username').fill(`friend_${suffix}`);await a.getByRole('button',{name:'Cari',exact:true}).click();await a.getByRole('button',{name:'Kirim permintaan',exact:true}).click();
  await b.locator('#social-button').click();await b.getByRole('button',{name:'Terima',exact:true}).click();await expect(b.locator('#social-content')).toContainText(`owner_${suffix}`);
  await a.locator('#social-refresh').click();const row=a.locator('#social-content li').filter({hasText:`@friend_${suffix}`});await row.getByRole('button',{name:'Undang',exact:true}).click();await a.getByRole('button',{name:'Kirim undangan 24 jam',exact:true}).click();await expect(a.locator('#social-content [role=status]')).toContainText('notifikasi');
  await b.locator('#social-refresh').click();await b.getByRole('button',{name:'Terima dan masuk',exact:true}).click();await expect(b.locator('#connection-summary')).toContainText('Online');
  await a.locator('#close-social').click();await a.locator('#worlds-button').click();await a.locator('[data-world]').filter({hasText:'T6 Pumpkin Grove'}).click();await a.getByRole('button',{name:'Masuk dunia privat',exact:true}).click();await expect(a.locator('#connection-summary')).toContainText('Online');
  await c.locator('#worlds-button').click();await c.getByLabel('Nama dunia').fill('T6 Other');await c.getByRole('button',{name:'Buat dunia',exact:true}).click();await c.getByRole('button',{name:'Masuk dunia privat',exact:true}).click();await c.locator('#chat-button').click();
  await a.locator('#chat-button').click();const input=a.locator('#chat-form textarea');const position=await a.locator('#world').getAttribute('data-position');await input.fill('wasd <script>plain text</script>');await a.waitForTimeout(250);expect(await a.locator('#world').getAttribute('data-position')).toBe(position);await a.locator('#chat-form button').click();await expect(a.locator('#chat-status')).toContainText('tersimpan');
  await expect(b.locator('#chat-badge')).toHaveText('1');await b.locator('#chat-button').click();await expect(b.locator('#chat-messages')).toContainText('wasd <script>plain text</script>');await expect(b.locator('#chat-messages script')).toHaveCount(0);await expect(c.locator('#chat-messages')).not.toContainText('plain text');
  await friend.setOffline(true);await friend.setOffline(false);await expect(b.locator('#connection-summary')).toContainText('Online');await expect(b.locator('#chat-messages p').filter({hasText:'plain text'})).toHaveCount(1);
  await a.locator('#chat-messages p').filter({hasText:'plain text'}).getByRole('button',{name:'Hapus',exact:true}).click();await expect(b.locator('#chat-messages')).toContainText('[Pesan dihapus]');
  await a.locator('#worlds-button').click();await a.locator('[data-world]').filter({hasText:'T6 Pumpkin Grove'}).click();const member=a.locator('.worlds-members li').filter({hasText:`@friend_${suffix}`});a.once('dialog',d=>d.accept());await member.getByRole('button',{name:'Mute chat',exact:true}).click();await b.locator('#chat-form textarea').fill('muted');await b.locator('#chat-form button').click();await expect(b.locator('#chat-status')).toContainText('ditolak');
  a.once('dialog',d=>d.accept());await a.locator('.worlds-members li').filter({hasText:`@friend_${suffix}`}).getByRole('button',{name:'Keluarkan',exact:true}).click();await expect(b.locator('#connection-summary')).toContainText('berakhir');
 }finally{await owner.close();await friend.close();await outsider.close()}
});
