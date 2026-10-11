import { test, expect, type Page } from '@playwright/test';

// Client-side navigation through an anchor inside the router's element, the same way
// in-app links are handled. Anchors outside it trigger a full page load.
async function navigate(page: Page, path: string) {
    await page.evaluate((href) => {
        const routerElem = [...document.querySelectorAll('div')].find(
            (el) => '__routify_meta' in el,
        );
        if (!routerElem) throw new Error('router element not found');
        const a = document.createElement('a');
        a.href = href;
        routerElem.append(a);
        a.click();
        a.remove();
    }, path);
    await expect(page).toHaveURL(path);
}

// The routed page's body; Home renders its own shell, which stays in the DOM while hidden.
const pageContent = (page: Page) => page.locator('.shell__body:not(.home .shell__body)');

test.describe('routing', () => {
    let pageErrors: Error[];
    let sockets: number;

    test.beforeEach(async ({ page }) => {
        pageErrors = [];
        sockets = 0;
        page.on('pageerror', (e) => pageErrors.push(e));

        // No backend: answer API calls as unauthenticated and accept the event stream.
        // Saved queries are fetched regardless of login and throw on errors, so return none.
        await page.route('**/api/**', (route) =>
            route.request().url().endsWith('/api/queries')
                ? route.fulfill({ json: [] })
                : route.fulfill({ status: 401, json: {} }),
        );
        await page.routeWebSocket('**/api/subscribe', () => {
            sockets++;
        });
    });

    test.afterEach(() => {
        expect(pageErrors).toEqual([]);
    });

    for (const [path, text] of [
        ['/myqueries', 'Saved Queries'],
        ['/mysensors', 'Sensors'],
        ['/myevents', 'Saved Events'],
    ]) {
        test(`direct load of ${path} renders the page and hides Home`, async ({ page }) => {
            await page.goto(path);
            await expect(pageContent(page)).toContainText(text);
            await expect(page.locator('.home')).toBeHidden();
        });
    }

    test('component styles are loaded', async ({ page }) => {
        // Pages and modules are dynamic imports, so their CSS is easy to drop from bundle.css.
        await page.goto('/');
        await expect(page.locator('.home main')).toHaveCSS('display', 'flex');
        await page.goto('/myqueries');
        await expect(pageContent(page)).toHaveCSS('display', 'flex');
    });

    test('event page loads directly', async ({ page }) => {
        await page.goto('/events/abc');
        await expect(page.locator('.home')).toBeHidden();
        await expect(pageContent(page)).toHaveCount(1);
    });

    test('Home stays mounted across navigation', async ({ page }) => {
        await page.goto('/');
        const home = page.locator('.home');
        await expect(home).toBeVisible();
        await home.evaluate((el) => (el.dataset.e2eMark = 'original'));

        for (const [path, text] of [
            ['/myqueries', 'Saved Queries'],
            ['/mysensors', 'Sensors'],
            ['/myevents', 'Saved Events'],
        ]) {
            await navigate(page, path);
            await expect(pageContent(page)).toContainText(text);
            await expect(home).toBeHidden();
        }

        await navigate(page, '/');
        await expect(home).toBeVisible();

        await page.goBack();
        await expect(page).toHaveURL('/myevents');
        await expect(home).toBeHidden();

        // Same element, never re-created, and the websocket was opened only once.
        await expect(page.locator('.home')).toHaveCount(1);
        await expect(home).toHaveAttribute('data-e2e-mark', 'original');
        expect(sockets).toBe(1);
    });
});
