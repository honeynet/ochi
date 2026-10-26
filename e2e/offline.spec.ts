import { test, expect } from '@playwright/test';

// The page loads Google's Sign-In script, which runs on every e2e page and has thrown
// while tests navigate, failing unrelated runs on a page error. The browser is configured
// not to resolve that host (see playwright.config.ts), so the tests stay off the network.
test.describe('e2e isolation', () => {
    test('no request to Google Sign-In succeeds', async ({ page }) => {
        const succeeded: string[] = [];
        page.on('requestfinished', (r) => {
            if (r.url().includes('accounts.google.com')) succeeded.push(r.url());
        });

        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', () => {});

        await page.goto('/');
        // The script tag is still in the page; what matters is that it never loads.
        await expect(page.locator('script[src*="accounts.google.com"]')).toHaveCount(1);
        // The login button keeps its fallback, so the header still works without the script.
        await expect(page.getByRole('button', { name: 'Sign in with Google' })).toBeVisible();

        expect(succeeded).toEqual([]);
    });
});
