import { test, expect } from '@playwright/test';

test.describe('event details pane', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated and send one event on the stream.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', (ws) => {
            ws.send(
                JSON.stringify({
                    sensorID: 'abcd1234',
                    srcHost: '203.0.113.10',
                    srcPort: '54321',
                    dstPort: 445,
                    transport: 'tcp',
                    rule: 'Rule: tcp dst port 445',
                    handler: 'smb',
                    timestamp: '2026-01-01T00:00:00Z',
                    payload: 'dGVzdA==',
                }),
            );
        });

        await page.goto('/');
        await expect(page.locator('.message')).toHaveCount(1);
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('explains what to do until an event is selected', async ({ page }) => {
        const hint = page.locator('#content .empty');
        await expect(hint).toHaveText(/Select an event to see its details/);

        await page.locator('.message').click();
        await expect(hint).toBeHidden();
        await expect(page.locator('#content')).toContainText('203.0.113.10');
    });
});
