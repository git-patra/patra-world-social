import {defineConfig,devices} from '@playwright/test';
// Guest-only tests against the existing local Vite process; no auth or backend fixture injection.
export default defineConfig({testDir:'./tests',testMatch:'*.guest.e2e.spec.ts',outputDir:'../../.cache/playwright-guest',fullyParallel:false,retries:0,timeout:90000,reporter:'line',use:{...devices['Desktop Chrome'],launchOptions:process.env.GUEST_BROWSER_EXECUTABLE?{executablePath:process.env.GUEST_BROWSER_EXECUTABLE}:undefined,baseURL:'http://127.0.0.1:4176/',trace:'off',screenshot:'off',video:'off'}});
