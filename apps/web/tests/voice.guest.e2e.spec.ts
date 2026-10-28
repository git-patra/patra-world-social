// Guest-only authored regression. Provider/device/replay acceptance is manual
// and remains pending; this test never signs in or requests microphone access.
import {test,expect} from '@playwright/test';
test('guest Warkop explains unavailable voice without requesting a microphone',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/');
 await page.getByRole('button',{name:/Warkop/}).first().click();
 const panel=page.getByRole('region',{name:'Warkop voice dan chat meja'});
 await expect(panel).toContainText('Meja Depan');await expect(panel).toContainText('Meja Pojok');
 await expect(panel).toContainText('Guest hanya menjelajah solo');
 for(const button of await panel.getByRole('button',{name:'Gabung obrolan'}).all())await expect(button).toBeDisabled();
 await expect(panel.getByRole('button',{name:'Nyalakan mic'})).toHaveCount(0);
 const box=await panel.boundingBox();expect(box!.x).toBeGreaterThanOrEqual(0);expect(box!.x+box!.width).toBeLessThanOrEqual(390);
});
