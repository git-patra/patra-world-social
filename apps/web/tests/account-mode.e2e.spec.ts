// AUTH_BROWSER_NOT_RUN: credential browser flows remain explicitly deferred.
// Authored for the guarded isolated-schema browser harness only; no injected
// cookies, local-storage session tokens, or copied production accounts.
import {test,expect} from '@playwright/test';
import {randomBytes} from 'node:crypto';
test('guest result stays local through login, online world, and logout',async({page})=>{
 test.setTimeout(180000);
 const name=`mode_${randomBytes(5).toString('hex')}`,password=`Synthetic!${randomBytes(16).toString('base64url')}`;
 await page.goto('/');await page.clock.install();
 await page.locator('#play-button').click();await page.getByRole('button',{name:'Mulai practice vs BOT'}).click();await page.clock.fastForward(95000);await expect(page.getByText('Practice selesai.',{exact:true})).toBeVisible();await page.locator('#close-dialog').click();
 await page.locator('#history-button').click();await expect(page.locator('#history-content details')).toHaveCount(1);await page.locator('#close-history').click();
 await page.locator('#account-button').click();await page.getByRole('button',{name:'Daftar',exact:true}).click();await page.getByLabel('Nama pengguna').fill(name);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Buat akun',exact:true}).click();await page.getByRole('button',{name:'Sudah kusimpan'}).click();await page.getByLabel('Nama pengguna').fill(name);await page.getByLabel('Kata sandi',{exact:true}).fill(password);await page.getByRole('button',{name:'Masuk ke desa',exact:true}).click();
 await expect(page.locator('#account-label')).toHaveText(`@${name}`);
 await page.locator('#history-button').click();await expect(page.locator('#history-content details')).toHaveCount(1);await expect(page.getByText('Belum ada pertandingan untuk filter ini.')).toBeVisible();await page.locator('#close-history').click();
 await page.locator('#worlds-button').click();await page.getByLabel('Nama dunia').fill('Synthetic mode world');await page.getByRole('button',{name:'Buat dunia',exact:true}).click();await page.getByRole('button',{name:/Masuk dunia/}).click();await expect(page.locator('.online-status')).toContainText('Online');
 await page.locator('#account-button').click();await page.getByRole('button',{name:'Keluar akun',exact:true}).click();await page.locator('#close-account').click();await expect(page.locator('.online-status')).toContainText('Solo lokal');
 await page.locator('#play-button').click();await expect(page.getByRole('button',{name:'Mulai practice vs BOT'})).toBeVisible();await expect(page.getByText('Practice selesai.',{exact:true})).toHaveCount(0);await page.locator('#close-dialog').click();await page.locator('#history-button').click();await expect(page.locator('#history-content details')).toHaveCount(1);expect(await page.locator('audio').count()).toBe(0);
});
