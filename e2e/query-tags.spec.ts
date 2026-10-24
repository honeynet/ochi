import { test, expect, type Page } from '@playwright/test';

const SAVED = {
    id: 'q1',
    content: 'tcp.port == 445',
    description: 'smb scans',
    active: false,
    tags: [{ type: 'malware', value: 'mirai' }],
};

// No backend: serve one saved query, record what an update sends, and answer anything
// else as unauthenticated. `reject` makes the update fail with that 400 message.
async function stubBackend(page: Page, reject?: string) {
    const updates: unknown[] = [];
    await page.route('**/api/**', async (route) => {
        const req = route.request();
        const url = req.url();
        if (url.endsWith('/api/queries') && req.method() === 'GET') {
            return route.fulfill({ json: [SAVED] });
        }
        if (url.endsWith(`/api/queries/${SAVED.id}`) && req.method() === 'PATCH') {
            updates.push(req.postDataJSON());
            return reject
                ? route.fulfill({ status: 400, body: reject + '\n' })
                : route.fulfill({ status: 200, body: '' });
        }
        return route.fulfill({ status: 401, json: {} });
    });
    await page.routeWebSocket('**/api/subscribe', () => {});
    return updates;
}

test.describe('query tags', () => {
    let pageErrors: Error[];

    test.beforeEach(({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('saved queries show their tags', async ({ page }) => {
        await stubBackend(page);
        await page.goto('/myqueries');

        await expect(page.locator('.queryList__tags .tag')).toHaveText(['malware: mirai']);
    });

    test('tags can be added and removed, and are sent on update', async ({ page }) => {
        const updates = await stubBackend(page);
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        const chips = dialog.locator('.tag span');
        const addButton = dialog.getByRole('button', { name: 'Add tag' });
        await expect(chips).toHaveText(['malware: mirai']);
        await expect(addButton).toBeDisabled();

        await dialog.getByLabel('Tag type').fill('protocol');
        await dialog.getByLabel('Tag value').fill('smb');
        await dialog.getByLabel('Tag value').press('Enter');
        await expect(chips).toHaveText(['malware: mirai', 'protocol: smb']);

        // Adding the same tag again changes nothing.
        await dialog.getByLabel('Tag type').fill('protocol');
        await dialog.getByLabel('Tag value').fill('smb');
        await addButton.click();
        await expect(chips).toHaveText(['malware: mirai', 'protocol: smb']);

        await dialog.getByRole('button', { name: 'Remove tag malware: mirai' }).click();
        await expect(chips).toHaveText(['protocol: smb']);

        await dialog.getByRole('button', { name: 'Update' }).click();
        await expect.poll(() => updates.length).toBe(1);
        expect(updates[0]).toMatchObject({
            id: SAVED.id,
            tags: [{ type: 'protocol', value: 'smb' }],
        });
    });

    test('cancelling the dialog discards tag changes', async ({ page }) => {
        const updates = await stubBackend(page);
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        await dialog.getByRole('button', { name: 'Remove tag malware: mirai' }).click();
        await dialog.getByRole('button', { name: 'Cancel' }).click();
        await expect(dialog).toBeHidden();

        await expect(page.locator('.queryList__tags .tag')).toHaveText(['malware: mirai']);
        await page.getByRole('button', { name: 'edit' }).click();
        await expect(dialog.locator('.tag span')).toHaveText(['malware: mirai']);
        expect(updates).toHaveLength(0);
    });
    test('a tag typed but not added is saved with the query', async ({ page }) => {
        const updates = await stubBackend(page);
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        await dialog.getByLabel('Tag type').fill('protocol');
        await dialog.getByLabel('Tag value').fill('smb');
        await dialog.getByRole('button', { name: 'Update' }).click();

        await expect.poll(() => updates.length).toBe(1);
        expect(updates[0]).toMatchObject({
            tags: [
                { type: 'malware', value: 'mirai' },
                { type: 'protocol', value: 'smb' },
            ],
        });
    });

    test('a half-typed tag blocks saving instead of being dropped', async ({ page }) => {
        const updates = await stubBackend(page);
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        const update = dialog.getByRole('button', { name: 'Update' });
        await dialog.getByLabel('Tag type').fill('protocol');
        await expect(update).toBeDisabled();
        await expect(dialog).toContainText('Finish or clear the tag before saving.');

        await dialog.getByLabel('Tag type').fill('');
        await expect(update).toBeEnabled();
        await update.click();
        await expect.poll(() => updates.length).toBe(1);
    });

    test("shows the server's error and keeps the dialog open", async ({ page }) => {
        await stubBackend(page, 'a query can have at most 20 tags');
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        await dialog.getByRole('button', { name: 'Update' }).click();

        await expect(dialog.getByRole('alert')).toHaveText('a query can have at most 20 tags');
        await expect(dialog).toBeVisible();
    });

    test('tag length is counted in characters, as the server counts it', async ({ page }) => {
        await stubBackend(page);
        await page.goto('/myqueries');
        await page.getByRole('button', { name: 'edit' }).click();

        const dialog = page.getByRole('dialog');
        const add = dialog.getByRole('button', { name: 'Add tag' });
        await dialog.getByLabel('Tag type').fill('emoji');

        // 64 characters outside the BMP are 128 UTF-16 units: allowed, as on the server.
        await dialog.getByLabel('Tag value').fill('\u{1F600}'.repeat(64));
        await expect(add).toBeEnabled();

        await dialog.getByLabel('Tag value').fill('\u{1F600}'.repeat(65));
        await expect(add).toBeDisabled();
        await expect(dialog).toContainText('at most 64 characters');
    });
});
