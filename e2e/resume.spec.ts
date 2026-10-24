import { test, expect, type WebSocketRoute } from '@playwright/test';

let seq = 0;
function event() {
    seq++;
    return JSON.stringify({
        sensorID: 'abcd1234',
        srcHost: '203.0.113.10',
        srcPort: String(40000 + seq),
        dstPort: 445,
        transport: 'tcp',
        rule: 'Rule: tcp dst port 445',
        handler: 'smb',
        timestamp: '2026-01-01T00:00:00Z',
        payload: 'dGVzdA==',
    });
}

test.describe('resume button', () => {
    let pageErrors: Error[];
    let socket: WebSocketRoute;

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated and keep the event stream open.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', (ws) => {
            socket = ws;
            ws.send(event());
        });

        await page.goto('/');
        await expect(page.locator('.message')).toHaveCount(1);
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('counts the events that arrive while paused', async ({ page }) => {
        const resume = page.locator('#resume-btn');

        // Scrolling the list pauses it.
        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toHaveText('Resume');

        for (let i = 0; i < 3; i++) socket.send(event());
        await expect(page.locator('.message')).toHaveCount(4);
        await expect(resume).toHaveText('Resume (3 new)');

        // Resuming clears the count, so the next pause starts from zero.
        await resume.click();
        await expect(resume).toBeHidden();
        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toHaveText('Resume');
    });
});
