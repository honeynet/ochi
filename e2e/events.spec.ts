import { test, expect } from '@playwright/test';

test.describe('saved events list and bulk delete confirmation', () => {
    let pageErrors: Error[];
    let deletedIds: string[] | null = null;

    const mockEvents = [
        {
            id: 'event-1',
            sensorID: 'sensor-1',
            srcHost: '192.0.2.1',
            srcPort: '12345',
            dstPort: 80,
            handler: 'http',
            timestamp: '2026-01-01T00:00:00Z',
        },
        {
            id: 'event-2',
            sensorID: 'sensor-1',
            srcHost: '192.0.2.2',
            srcPort: '12346',
            dstPort: 443,
            handler: 'tls',
            timestamp: '2026-01-01T00:01:00Z',
        },
    ];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        deletedIds = null;
        page.on('pageerror', (e) => pageErrors.push(e));

        await page.route('**/api/**', (route) => {
            const req = route.request();
            const url = new URL(req.url());
            if (url.pathname === '/api/events') {
                if (req.method() === 'GET') {
                    return route.fulfill({
                        status: 200,
                        headers: {
                            'X-Total-Count': String(mockEvents.length),
                        },
                        json: mockEvents,
                    });
                }
                if (req.method() === 'DELETE') {
                    const data = req.postDataJSON();
                    deletedIds = data?.ids ?? [];
                    return route.fulfill({ status: 200, json: {} });
                }
            }
            if (url.pathname === '/api/queries') {
                return route.fulfill({ json: [] });
            }
            return route.fulfill({ status: 401, json: {} });
        });
        await page.routeWebSocket('**/api/subscribe', () => {});
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('requires confirmation dialog before deleting selected events', async ({ page }) => {
        await page.goto('/myevents');

        // Check that "Select page" selects the events
        const deleteButton = page.getByRole('button', { name: /Delete selected/ });
        await expect(deleteButton).toBeDisabled();

        await page.getByLabel('Select page').check();
        await expect(deleteButton).toBeEnabled();
        await expect(deleteButton).toHaveText('Delete selected (2)');

        const dialog = page.locator('dialog:has(.confirm-dialog)');

        // 1. Clicking Cancel closes the dialog without firing DELETE request
        await deleteButton.click();
        await expect(dialog).toBeVisible();
        await expect(dialog).toContainText(
            'Delete 2 selected events? This action cannot be undone.',
        );
        await dialog.getByRole('button', { name: 'Cancel' }).click();
        await expect(dialog).not.toBeVisible();
        expect(deletedIds).toBeNull();

        // 2. Pressing Escape closes the dialog without firing DELETE request
        await deleteButton.click();
        await expect(dialog).toBeVisible();
        await page.keyboard.press('Escape');
        await expect(dialog).not.toBeVisible();
        expect(deletedIds).toBeNull();

        // 3. Clicking the backdrop closes the dialog without firing DELETE request
        await deleteButton.click();
        await expect(dialog).toBeVisible();
        // Click outside the dialog content box to hit the backdrop
        await page.mouse.click(10, 10);
        await expect(dialog).not.toBeVisible();
        expect(deletedIds).toBeNull();

        // 4. Confirming deletion sends DELETE request with selected IDs
        await deleteButton.click();
        await expect(dialog).toBeVisible();
        await dialog.getByRole('button', { name: 'Delete' }).click();
        await expect.poll(() => deletedIds).toEqual(['event-1', 'event-2']);
        await expect(dialog).not.toBeVisible();
    });
});
