// AUTH_BROWSER_NOT_RUN: authored for the isolated browser acceptance run after credential-flow approval.
import {test,expect,type BrowserContext,type Page} from '@playwright/test';
import {randomBytes} from 'node:crypto';

async function createAccount(context:BrowserContext,name:string,password:string):Promise<Page>{
 const page=await context.newPage();await page.goto('/');
 await page.locator('#account-button').click();await page.getByRole('button',{name:'Daftar',exact:true}).click();
 await page.getByLabel('Nama pengguna').fill(name);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Buat akun',exact:true}).click();
 await expect(page.getByRole('heading',{name:'Kode pemulihan'})).toBeVisible();await page.getByRole('button',{name:'Sudah kusimpan'}).click();
 await page.getByLabel('Nama pengguna').fill(name);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Masuk ke desa',exact:true}).click();
 await expect(page.locator('#account-label')).toHaveText(`@${name}`);return page;
}

test('three contexts keep a targeted invite private and revoke member access',async({browser})=>{
 const suffix=randomBytes(5).toString('hex'),password=`LocalTest!${randomBytes(16).toString('base64url')}`;
 const owner=await browser.newContext(),friend=await browser.newContext(),outsider=await browser.newContext();
 try{
  const ownerPage=await createAccount(owner,`owner_${suffix}`,password);
  const friendPage=await createAccount(friend,`friend_${suffix}`,password);
  const outsiderPage=await createAccount(outsider,`other_${suffix}`,password);
  await ownerPage.locator('#worlds-button').click();await ownerPage.getByLabel('Nama dunia').fill('Pumpkin Grove');await ownerPage.getByRole('button',{name:'Buat dunia'}).click();
  await expect(ownerPage.getByRole('heading',{name:'Pumpkin Grove'})).toBeVisible();
  await ownerPage.getByLabel('Username tujuan (opsional)').fill(`friend_${suffix}`);await ownerPage.getByRole('button',{name:'Buat tautan 24 jam'}).click();
  const link=await ownerPage.locator('#invite-link input').inputValue();expect(link).toContain('#invite=');
  await outsiderPage.goto(link);await expect(outsiderPage.getByRole('heading',{name:'Terima undangan'})).toBeVisible();
  await outsiderPage.getByRole('button',{name:'Terima undangan',exact:true}).click();await expect(outsiderPage.locator('.worlds-message')).toContainText('tidak tersedia');
  await friendPage.goto(link);await friendPage.getByRole('button',{name:'Terima undangan',exact:true}).click();
  await expect(friendPage.getByRole('heading',{name:'Pumpkin Grove'})).toBeVisible();
  await expect(friendPage.locator('.worlds-members')).toContainText(`friend_${suffix}`);
  await ownerPage.locator('#world-back').click();
  await ownerPage.locator('[data-world]').filter({hasText:'Pumpkin Grove'}).click();
  const friendRow=ownerPage.locator('.worlds-members li').filter({hasText:`@friend_${suffix}`});
  await expect(friendRow).toBeVisible();
  ownerPage.once('dialog',d=>d.accept());await friendRow.locator('[data-member-action="kick"]').click();
  await friendPage.locator('#worlds-dialog').evaluate((node:HTMLDialogElement)=>node.close());await friendPage.locator('#worlds-button').click();
  await expect(friendPage.locator('.worlds-list')).not.toContainText('Pumpkin Grove');
 }finally{await owner.close();await friend.close();await outsider.close()}
});
