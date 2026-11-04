import { test, expect } from '@playwright/test';

test.describe('event list source column', () => {
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
                dstPort: 445,
                transport: 'tcp',
                rule: 'Rule: tcp dst port 445',
                handler: 'smb',
                timestamp: '2026-01-01T00:00:00Z',
                payload: 'dGVzdA==',
            };
            ws.send(JSON.stringify({ ...event, srcHost: '203.0.113.214', srcPort: '54321' }));
            ws.send(
                JSON.stringify({
                    ...event,
                    srcHost: '198.51.100.7',
                    srcPort: '40000',
                    srcPtr: 'scanner.example.',
                }),
            );
        });

        await page.goto('/');
        await expect(page.locator('.message')).toHaveCount(2);
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('shows the full source address and keeps the port in the tooltip', async ({ page }) => {
        const sources = page.locator('.message .cell.source');

        await expect(sources.nth(0)).toHaveText('203.0.113.214');
        await expect(sources.nth(0)).toHaveAttribute('title', '203.0.113.214:54321');

        // A reverse DNS name stays in the tooltip, next to the address and port.
        await expect(sources.nth(1)).toHaveText('198.51.100.7');
        await expect(sources.nth(1)).toHaveAttribute(
            'title',
            'scanner.example. (198.51.100.7:40000)',
        );
    });

    // The column has to stay wide enough for an address once the Time column is there too.
    // The check runs in a wider monospace font as well, like the ones Linux uses, because
    // the default one on Windows is narrower and hides the problem.
    test('does not cut the address off at the default width', async ({ page }) => {
        const source = page.locator('.message .cell.source').first();
        const log = page.locator('#message-log');

        for (const font of ['monospace', 'Courier New']) {
            await page.addStyleTag({
                content: `.event-head, .message { font-family: '${font}', monospace !important; }`,
            });
            expect(
                await source.evaluate((el) => el.scrollWidth <= el.clientWidth),
                `address is cut off in ${font}`,
            ).toBe(true);
            // Rows still fit, so the Details link at the end is never clipped.
            expect(
                await log.evaluate((el) => el.scrollWidth <= el.clientWidth),
                `rows overflow the list in ${font}`,
            ).toBe(true);
        }
    });
    // #218 gave Source a share of the spare space so an address would fit at the default
    // width. That share had no ceiling, so widening the pane kept stretching the column and
    // opened a gap between the address and the columns after it, which moved away from the
    // ones before it. The column now stops once it can show an address.
    test('widening the list does not stretch the source column', async ({ page }) => {
        await page.setViewportSize({ width: 1600, height: 900 });

        const handle = page.locator('.handle');
        const box = await handle.boundingBox();
        if (!box) throw new Error('split handle not found');
        await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
        await page.mouse.down();
        await page.mouse.move(1600 * 0.85, box.y + box.height / 2, { steps: 10 });
        await page.mouse.up();

        const log = page.locator('#message-log');
        await expect
            .poll(async () => Math.round((await log.boundingBox())!.width))
            .toBeGreaterThan(1100);

        // An address plus a port is 21 characters at most, so the track has no reason to be
        // this wide. Before the cap it took most of the spare space, roughly 600px here.
        const source = page.locator('.message .cell.source').first();
        const width = Math.round((await source.boundingBox())!.width);
        expect(width, `source column is ${width}px wide in a ${1600 * 0.85}px list`).toBeLessThan(
            260,
        );
    });
});
