import {defineConfig,devices} from '@playwright/test';

const baseURL=process.env.AUTH_E2E_BASE_URL;
if(!/^auth_e2e_[0-9a-f]{16}$/.test(process.env.AUTH_E2E_SCHEMA||'')||baseURL!=='http://127.0.0.1:4177/'){
 throw new Error('Auth E2E requires AUTH_E2E_SCHEMA and AUTH_E2E_BASE_URL=http://127.0.0.1:4177/ backed by the isolated schema.');
}
export default defineConfig({
 testDir:'./tests',testMatch:'*.e2e.spec.ts',fullyParallel:false,retries:0,reporter:'line',
 use:{...devices['Desktop Chrome'],baseURL,trace:'off',screenshot:'off',video:'off'},
});
