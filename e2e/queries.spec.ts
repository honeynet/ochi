import { test, expect } from '@playwright/test';

test.describe('saved queries list and modal', () => {
    let pageErrors: Error[];

    const mockQueries = [
        { id: 'query-1', content: 'tcp.port == 80', description: 'HTTP query', active: true },
        { id: 'query-2', content: 'tcp.port == 443', description: 'HTTPS query', active: false },
    ];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        await page.route('**/api/**', (route) => {
            const url = route.request().url();
            if (url.endsWith('/api/queries')) {
                return route.fulfill({ json: mockQueries });
            }
            return route.fulfill({ status: 401, json: {} });
        });
        await page.routeWebSocket('**/api/subscribe', () => {});
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('renders a single dialog instance outside list items', async ({ page }) => {
        await page.goto('/myqueries');

        // Verify query items are rendered
        await expect(page.locator('.queryList__item')).toHaveCount(2);

        // There must be no QueryModal dialog inside list items
        await expect(page.locator('.queryList__item dialog')).toHaveCount(0);
        // And exactly one QueryModal dialog in the queries page view
        const queryDialog = page.locator('dialog:has(.queryModal__form)');
        await expect(queryDialog).toHaveCount(1);
    });

    test('cancelling edit does not mutate query in the list', async ({ page }) => {
        await page.goto('/myqueries');

        const firstItem = page.locator('.queryList__item').first();
        await expect(firstItem.locator('.queryList__content')).toHaveText('tcp.port == 80');

        // Click edit on the first query
        await firstItem.getByRole('button', { name: 'edit' }).click();

        const dialog = page.locator('dialog:has(.queryModal__form)');
        await expect(dialog).toBeVisible();

        // Edit the content field
        const contentInput = dialog.locator('label', { hasText: 'Content' }).locator('input');
        await contentInput.fill('tcp.port == 8080');

        // Click Cancel
        await dialog.getByRole('button', { name: 'Cancel' }).click();
        await expect(dialog).not.toBeVisible();

        // List item must retain original value without in-memory mutation
        await expect(firstItem.locator('.queryList__content')).toHaveText('tcp.port == 80');
    });

    test('query modal inputs do not use duplicate messages-input-box ID', async ({ page }) => {
        await page.goto('/myqueries');

        await expect(page.locator('.queryModal__form #messages-input-box')).toHaveCount(0);
    });
});
