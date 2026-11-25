import { test, expect, type Page } from '@playwright/test';

const event = (dstPort: number, handler: string) =>
    JSON.stringify({
        sensorID: 'abcd1234',
        srcHost: '203.0.113.10',
        srcPort: '54321',
        dstPort,
        transport: 'tcp',
        rule: `Rule: tcp dst port ${dstPort}`,
        handler,
        timestamp: '2026-01-01T00:00:00Z',
        payload: 'dGVzdA==',
    });

const rows = (page: Page, table: string) => page.locator(`#${table} tr`);

test.describe('statistics page', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated. The stream sends three events on
        // its first connection only, so a reload shows whether the counts were kept.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        let connections = 0;
        await page.routeWebSocket('**/api/subscribe', (ws) => {
            if (connections++ === 0) {
                ws.send(event(445, 'smb'));
                ws.send(event(445, 'smb'));
                ws.send(event(80, 'http'));
            }
        });

        await page.goto('/');
        await expect(page.locator('.message')).toHaveCount(3);
        await page.getByRole('link', { name: 'Stats' }).click();
        await expect(page).toHaveURL('/stats');
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('counts the live events', async ({ page }) => {
        await expect(page.locator('#stats-total')).toHaveText('3');
        await expect(page.locator('#stats-rate')).toHaveText('3');
        await expect(rows(page, 'stats-ports').first()).toHaveText('tcp/4452');
        await expect(rows(page, 'stats-ports').nth(1)).toHaveText('tcp/801');
        await expect(rows(page, 'stats-handlers').first()).toHaveText('smb2');
        await expect(page.getByText('Sign in and save a query')).toBeVisible();
    });

    test('keeps the counts across a reload, and reset clears them', async ({ page }) => {
        await page.reload();
        await expect(page.locator('#stats-total')).toHaveText('3');
        // the rate is only for this session
        await expect(page.locator('#stats-rate')).toHaveText('0');

        await page.getByRole('button', { name: 'Reset' }).click();
        await page.locator('dialog').getByRole('button', { name: 'Reset' }).click();
        await expect(page.locator('#stats-total')).toHaveText('0');
        await expect(page.locator('#stats-ports')).toHaveCount(0);

        await page.reload();
        await expect(page.locator('#stats-total')).toHaveText('0');
    });
});
