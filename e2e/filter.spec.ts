import { test, expect, type Locator } from '@playwright/test';

const LONG_FILTER =
    'tcp.port == 445 and ip.src eq 203.0.113.10 and transport eq "tcp" and end.reason eq "timeout"';

async function size(locator: Locator) {
    const box = await locator.boundingBox();
    if (!box) throw new Error('element is not rendered');
    return box;
}

test.describe('filter input', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated and accept the event stream.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', () => {});

        await page.goto('/');
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('widens when focused', async ({ page }) => {
        const field = page.locator('.filter-input');
        const before = await size(field);

        await field.focus();
        await expect.poll(async () => (await size(field)).width).toBeGreaterThan(before.width);
    });

    test('grows to fit a long filter while focused and shrinks back when blurred', async ({
        page,
    }) => {
        const field = page.locator('.filter-input');
        const singleLine = (await size(field)).height;

        await field.fill(LONG_FILTER);
        await expect.poll(async () => (await size(field)).height).toBeGreaterThan(singleLine);

        await field.blur();
        await expect.poll(async () => (await size(field)).height).toBe(singleLine);
        // Nothing was lost by collapsing it.
        await expect(field).toHaveValue(LONG_FILTER);
    });

    test('clear button empties the filter', async ({ page }) => {
        const field = page.locator('.filter-input');
        const clear = page.getByRole('button', { name: 'Clear filter' });

        await expect(clear).toHaveCount(0);

        await field.fill('tcp.port == 80');
        await expect(clear).toBeVisible();

        await clear.click();
        await expect(field).toHaveValue('');
        await expect(clear).toHaveCount(0);
        await expect(field).toBeFocused();
    });

    test('clear button works with a mouse while the field is not focused', async ({ page }) => {
        const field = page.locator('.filter-input');
        const clear = page.getByRole('button', { name: 'Clear filter' });
        const restingWidth = (await size(field)).width;

        // A saved query left in the field, with focus elsewhere.
        await field.fill('tcp.port == 80');
        await field.blur();
        await expect(clear).toBeVisible();
        // Let the field finish collapsing back before measuring where the button is.
        await expect.poll(async () => (await size(field)).width).toBe(restingWidth);

        // Press and release at human speed rather than Playwright's instant click. If the
        // press widens the field, the button slides out from under the pointer before the
        // release and the click never reaches it.
        const box = await size(clear);
        await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
        await page.mouse.down();
        await page.waitForTimeout(250);
        await page.mouse.up();

        await expect(field).toHaveValue('');
    });

    test('Enter does not add a newline', async ({ page }) => {
        const field = page.locator('.filter-input');

        await field.fill('tcp.port == 80');
        await field.press('Enter');

        await expect(field).toHaveValue('tcp.port == 80');
    });
});

// The message list moves the selection on ArrowUp/ArrowDown from anywhere on the page.
// The filter is a textarea now, where those keys move the caret between lines, so the
// list has to leave them alone while the filter has focus.
test.describe('filter input with events in the list', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

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

    test('arrow keys in the filter do not move the event selection', async ({ page }) => {
        const field = page.locator('.filter-input');
        const selected = page.locator('.message.selected');

        await field.fill('tcp.port == 445');
        await field.press('ArrowDown');
        await field.press('ArrowUp');
        await expect(selected).toHaveCount(0);

        // Outside the filter the same key does select, so the check above is not vacuous.
        await field.blur();
        await page.keyboard.press('ArrowDown');
        await expect(selected).toHaveCount(1);
    });
});
