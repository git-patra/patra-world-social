// Guest UI regression only. Run through the existing isolated E2E harness.
// Physical on-screen keyboard and browser zoom 200% remain manual acceptance.
import {test,expect,type Page,type Locator} from '@playwright/test';

async function centered(page:Page,panel:Locator){
 await expect(panel).toBeVisible();
 const box=await panel.boundingBox(),viewport=page.viewportSize()!;
 expect(box).not.toBeNull();
 expect(box!.x).toBeGreaterThanOrEqual(0);expect(box!.y).toBeGreaterThanOrEqual(0);
 expect(box!.x+box!.width).toBeLessThanOrEqual(viewport.width+1);
 expect(box!.y+box!.height).toBeLessThanOrEqual(viewport.height+1);
 expect(Math.abs(box!.x+box!.width/2-viewport.width/2)).toBeLessThanOrEqual(2);
 expect(Math.abs(box!.y+box!.height/2-viewport.height/2)).toBeLessThanOrEqual(2);
}

for(const viewport of [{width:320,height:568},{width:390,height:844},{width:768,height:1024},{width:844,height:390},{width:1440,height:900}]){
 test(`minimap, chat and menu center at ${viewport.width}x${viewport.height}`,async({page})=>{
  await page.setViewportSize(viewport);await page.goto('/');
  const world=page.locator('#world'),before=await world.getAttribute('data-position');
  const minimap=page.locator('#minimap-button');await minimap.click();await centered(page,page.locator('#map-dialog'));
  await page.waitForTimeout(180);await expect(world).toHaveAttribute('data-position',before!);
  await page.keyboard.press('Tab');
  expect(await page.locator('#map-dialog').evaluate(dialog=>dialog.contains(document.activeElement))).toBe(true);
  await page.keyboard.press('Escape');await expect(page.locator('#map-dialog')).not.toBeVisible();await expect(minimap).toBeFocused();
  await page.locator('#chat-button').click();await centered(page,page.locator('#chat-panel'));
  await page.locator('#chat-form textarea').fill('WASD test');await page.keyboard.type('wasd');
  await page.waitForTimeout(180);await expect(world).toHaveAttribute('data-position',before!);
  await page.keyboard.press('Escape');await expect(page.locator('#chat-button')).toBeFocused();
  await page.locator('#menu-button').click();await centered(page,page.locator('#menu-dialog'));
  await page.keyboard.press('Escape');await expect(page.locator('#menu-button')).toBeFocused();
  await expect(page.locator('dialog[open]')).toHaveCount(0);
 });
}

test('full map click closes then walks toward the point without teleporting',async({page})=>{
 await page.goto('/');const world=page.locator('#world');const beforeText=(await world.getAttribute('data-position'))!,before=beforeText.split(',').map(Number);
 await page.locator('#list-button').click();const map=page.locator('#village-map'),box=(await map.boundingBox())!;
 const scale=Math.min(box.width/2560,box.height/1920),left=(box.width-2560*scale)/2,top=(box.height-1920*scale)/2;
 await map.click({position:{x:left+1820*scale,y:top+1500*scale}});await expect(page.locator('#map-dialog')).not.toBeVisible();
 const after=(await world.getAttribute('data-position'))!.split(',').map(Number);
 expect(Math.hypot(after[0]-before[0],after[1]-before[1])).toBeLessThan(2);
 await expect.poll(()=>world.getAttribute('data-position')).not.toBe(beforeText);
});

test('all guest popup entrypoints stay centered with their close controls reachable',async({page})=>{
 await page.setViewportSize({width:390,height:844});await page.goto('/');
 for(const [opener,panel] of [['#account-button','#account-dialog'],['#worlds-button','#worlds-dialog'],['#social-button','#social-dialog']] as const){
  await page.locator(opener).click();await centered(page,page.locator(panel));await page.keyboard.press('Escape');
 }
 for(const [opener,panel] of [['#settings-button','#settings-dialog'],['#history-button','#history-dialog'],['#podium-button','#podium-dialog'],['[data-project="duck"]','#project-dialog']] as const){
  await page.locator('#menu-button').click();await page.locator('#menu-dialog').locator(opener).click();await centered(page,page.locator(panel));
  expect(await page.locator(panel).evaluate(dialog=>dialog.contains(document.activeElement))).toBe(true);
  await page.keyboard.press('Escape');await expect(page.locator('#menu-button')).toBeFocused();
 }
});
