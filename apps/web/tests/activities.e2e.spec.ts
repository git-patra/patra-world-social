import {test,expect} from '@playwright/test';
test('combined production-renderer smoke tears down four A/B lifecycles',async({page})=>{
 await page.goto('/tests/activities-combined.html');
 await page.getByRole('button',{name:'Quick smoke 20 s / 4 cycles'}).click();
 await expect(page.locator('output')).toContainText('"done": true',{timeout:35000});
 const result=JSON.parse(await page.locator('output').innerText());
 expect(result.cyclesCompleted).toBe(4);
 expect(result.hiddenMs).toBe(0);
 expect(result.renderCallbackCPU.samples).toBeGreaterThan(100);
 for(const cleanup of result.cleanupCounts)for(const key of ['raf','intervals','timeouts','listeners','audioContexts','oscillators'])expect(cleanup[key],`${key} after cycle ${cleanup.cycle}`).toBe(0);
});
