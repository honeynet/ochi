import { test, expect } from '@playwright/test';

// Pin the browser's timezone so the expected local times are the UTC ones.
test.use({ timezoneId: 'UTC' });

const now = new Date().toISOString();
const older = '2026-01-01T12:34:56Z';

test.describe('event list time column', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated and send two events on the stream.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', (ws) => {
            const event = {
                sensorID: 'abcd1234',
                srcHost: '203.0.113.10',
                srcPort: '54321',
                dstPort: 445,
                transport: 'tcp',
                rule: 'Rule: tcp dst port 445',
                handler: 'smb',
                payload: 'dGVzdA==',
            };
            ws.send(JSON.stringify({ ...event, timestamp: older }));
            ws.send(JSON.stringify({ ...event, timestamp: now }));
        });

        await page.goto('/');
        await expect(page.locator('.message')).toHaveCount(2);
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('shows when each event happened', async ({ page }) => {
        await expect(page.locator('#message-log .event-head span').first()).toHaveText('Time');

        const times = page.locator('.message .cell.time');
        // Older events show the date, today's show the time; the tooltip has the full timestamp.
        await expect(times.nth(0)).toHaveText('Jan 01');
        await expect(times.nth(0)).toHaveAttribute('title', older);
        await expect(times.nth(1)).toHaveText(now.slice(11, 19));
        await expect(times.nth(1)).toHaveAttribute('title', now);

        // The extra column still fits: no row is wider than the list, which would clip the
        // last column (Details). Check again with a wider monospace font, like the ones Linux
        // uses, since the default one on Windows is narrower.
        const log = page.locator('#message-log');
        expect(await log.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
        await page.addStyleTag({
            content: `.event-head, .message { font-family: 'Courier New', monospace !important; }`,
        });
        expect(await log.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
    });
});
