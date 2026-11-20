import { test, expect, type Page, type WebSocketRoute } from '@playwright/test';

let seq = 0;
function event(dstPort = 445) {
    seq++;
    return JSON.stringify({
        sensorID: 'abcd1234',
        srcHost: '203.0.113.10',
        srcPort: String(40000 + seq),
        dstPort,
        transport: 'tcp',
        rule: `Rule: tcp dst port ${dstPort}`,
        handler: dstPort === 445 ? 'smb' : 'ssh',
        timestamp: '2026-01-01T00:00:00Z',
        payload: 'dGVzdA==',
    });
}

async function isScrollable(page: Page): Promise<boolean> {
    return page.locator('#message-log').evaluate((el) => el.scrollHeight > el.clientHeight);
}

// Fill the list past its viewport height without exceeding the default 50-event buffer,
// leaving headroom for follow-up events in the test.
async function fillUntilScrollable(
    page: Page,
    socket: WebSocketRoute,
    dstPort = 445,
): Promise<number> {
    let count = await page.locator('.message').count();
    while (!(await isScrollable(page)) && count < 45) {
        const batch = Math.min(5, 45 - count);
        for (let i = 0; i < batch; i++) socket.send(event(dstPort));
        count += batch;
        await expect(page.locator('.message')).toHaveCount(count);
    }
    expect(await isScrollable(page)).toBe(true);
    return count;
}

test.describe('live event list: pause, resume, and filtering', () => {
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
        const initialCount = await fillUntilScrollable(page, socket);

        // Scrolling a scrollable list pauses it.
        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toHaveText('Resume');

        for (let i = 0; i < 3; i++) socket.send(event());
        await expect(page.locator('.message')).toHaveCount(initialCount + 3);
        await expect(resume).toHaveText('Resume (3 new)');

        // Resuming clears the count, so the next pause starts from zero.
        await resume.click();
        await expect(resume).toBeHidden();
        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toHaveText('Resume');
    });

    test('wheel gesture does not pause when the list cannot scroll', async ({ page }) => {
        const resume = page.locator('#resume-btn');
        expect(await isScrollable(page)).toBe(false);

        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toBeHidden();

        // Also stays unpaused when a filter hides every row (empty list).
        const filter = page.locator('.filter-input');
        await filter.fill('tcp.port == 66');
        await filter.press('Enter');
        await expect(page.locator('.message')).toHaveCount(0);

        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -10);
        await expect(resume).toBeHidden();
    });

    test('applying a filter hides events instead of deleting them', async ({ page }) => {
        // Start with 2 events (1 from beforeEach + 1 here), matching the issue repro.
        socket.send(event(445));
        await expect(page.locator('.message')).toHaveCount(2);

        const filter = page.locator('.filter-input');
        await filter.fill('tcp.port == 66');
        await filter.press('Enter');
        await expect(page.locator('.message')).toHaveCount(0);

        // Events arriving while the filter is active are still buffered.
        for (let i = 0; i < 3; i++) socket.send(event(445));
        for (let i = 0; i < 2; i++) socket.send(event(22));
        await expect(page.locator('.message')).toHaveCount(0);

        // Switching to another filter reveals matching buffered events.
        await filter.fill('tcp.port == 22');
        await filter.press('Enter');
        await expect(page.locator('.message')).toHaveCount(2);

        // Clearing the filter restores all 7 events from before and during the filter.
        await page.getByRole('button', { name: 'Clear filter' }).click();
        await expect(page.locator('.message')).toHaveCount(7);
    });

    test('paused count only includes events matching the active filter', async ({ page }) => {
        const resume = page.locator('#resume-btn');
        const initialCount = await fillUntilScrollable(page, socket, 445);

        const filter = page.locator('.filter-input');
        await filter.fill('tcp.port == 445');
        await filter.press('Enter');
        await expect(page.locator('.message')).toHaveCount(initialCount);

        await page.locator('#message-log').hover();
        await page.mouse.wheel(0, -100);
        await expect(resume).toHaveText('Resume');

        // Non-matching events are buffered without inflating the paused count.
        socket.send(event(22));
        socket.send(event(22));
        await expect(page.locator('.message')).toHaveCount(initialCount);
        await expect(resume).toHaveText('Resume');

        // Matching events increment the paused count.
        socket.send(event(445));
        await expect(page.locator('.message')).toHaveCount(initialCount + 1);
        await expect(resume).toHaveText('Resume (1 new)');

        // Clearing the filter shows all buffered events, including the 2 port-22 events.
        await page.getByRole('button', { name: 'Clear filter' }).click();
        await expect(page.locator('.message')).toHaveCount(initialCount + 3);
    });
});
