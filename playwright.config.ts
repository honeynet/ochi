import { defineConfig, devices } from '@playwright/test';

const port = 5055;

// Runs against the production bundle in public/build; `npm run test:e2e` builds it first.
export default defineConfig({
    testDir: './e2e',
    forbidOnly: !!process.env.CI,
    retries: process.env.CI ? 1 : 0,
    reporter: process.env.CI ? 'github' : 'list',
    use: {
        baseURL: `http://localhost:${port}`,
        trace: 'on-first-retry',
    },
    projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
    webServer: {
        command: `npx sirv public --single --port ${port}`,
        url: `http://localhost:${port}`,
        reuseExistingServer: !process.env.CI,
    },
});
