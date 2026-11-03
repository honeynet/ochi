import { test, expect } from '@playwright/test';

test.describe('toast notifications', () => {
    let pageErrors: Error[];

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        page.on('pageerror', (e) => pageErrors.push(e));

        await page.route('**/api/**', (route) => {
            if (route.request().url().endsWith('/api/queries')) {
                return route.fulfill({ json: [] });
            }
            return route.fulfill({ status: 401, json: {} });
        });
        await page.routeWebSocket('**/api/subscribe', () => {});
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    test('invalid filter shows an error toast with role=alert and dismiss button', async ({
        page,
    }) => {
        await page.goto('/');
        const filterInput = page.locator('.filter-input');
        await filterInput.fill('tcp.port ==');
        await filterInput.press('Enter');

        const toast = page.locator('.toast.toast--error');
        await expect(toast).toBeVisible();
        await expect(toast).toContainText('Invalid filter');
        await expect(toast).toHaveAttribute('role', 'alert');

        // Dismiss button dismisses the toast immediately
        await toast.locator('.toast__dismiss').click();
        await expect(toast).not.toBeVisible();
    });

    test('pressing Enter repeatedly on an invalid filter deduplicates the toast', async ({
        page,
    }) => {
        await page.goto('/');
        const filterInput = page.locator('.filter-input');
        await filterInput.fill('udp.port ==');
        await filterInput.press('Enter');
        await filterInput.press('Enter');
        await filterInput.press('Enter');

        const toasts = page.locator('.toast.toast--error');
        await expect(toasts).toHaveCount(1);
    });

    test('toast with timeout auto-dismisses using fast-forward clock', async ({ page }) => {
        await page.clock.install();
        await page.goto('/');

        const filterInput = page.locator('.filter-input');
        await filterInput.fill('udp.port ==');
        await filterInput.press('Enter');

        const toast = page.locator('.toast.toast--error');
        await expect(toast).toBeVisible();

        // Advance mock clock by 5s to trigger auto-dismissal
        await page.clock.fastForward(5000);
        await expect(toast).not.toBeVisible();
    });

    test('hovering on a toast pauses the auto-dismiss timer', async ({ page }) => {
        await page.clock.install();
        await page.goto('/');

        const filterInput = page.locator('.filter-input');
        await filterInput.fill('udp.port ==');
        await filterInput.press('Enter');

        const toast = page.locator('.toast.toast--error');
        await expect(toast).toBeVisible();

        // Hover over the toast to pause auto-dismissal
        await toast.hover();

        // Advance mock clock past the normal 5s timeout
        await page.clock.fastForward(6000);

        // Toast should remain visible while hovered
        await expect(toast).toBeVisible();

        // Mouse leaves toast
        await page.mouse.move(0, 0);

        // Advance remaining time to dismiss
        await page.clock.fastForward(5000);
        await expect(toast).not.toBeVisible();
    });

    test('deduplicating a toast while hovered does not restart timer until unhovered', async ({
        page,
    }) => {
        await page.clock.install();
        await page.goto('/');

        const filterInput = page.locator('.filter-input');
        await filterInput.fill('tcp.port ==');
        await filterInput.press('Enter');

        const toast = page.locator('.toast.toast--error');
        await expect(toast).toBeVisible();

        // Hover to pause
        await toast.hover();

        // Press Enter again to deduplicate while hovered
        await filterInput.press('Enter');

        // Fast-forward past 5s; toast must stay visible under cursor
        await page.clock.fastForward(6000);
        await expect(toast).toBeVisible();

        // Move mouse away to resume timer
        await page.mouse.move(0, 0);

        // Advance clock to trigger resumed timeout
        await page.clock.fastForward(5000);
        await expect(toast).not.toBeVisible();
    });
});
