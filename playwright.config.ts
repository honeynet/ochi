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
    projects: [
        {
            name: 'chromium',
            use: {
                ...devices['Desktop Chrome'],
                // index.html loads Google's Sign-In script for the login button. No test signs
                // in, but the script runs on every page and sometimes throws while a test is
                // navigating, which fails the run on an unrelated page error. Stopping the
                // browser from resolving the host keeps the tests off the network. The login
                // button already shows its own fallback until the script loads.
                launchOptions: {
                    args: ['--host-resolver-rules=MAP accounts.google.com ~NOTFOUND'],
                },
            },
        },
    ],
    webServer: {
        command: `npx sirv public --single --port ${port}`,
        url: `http://localhost:${port}`,
        reuseExistingServer: !process.env.CI,
    },
});
