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

// The routed page's body inside the application shell.
const pageContent = (page: Page) => page.locator('.shell__body:not(.home .shell__body)');

async function authenticate(page: Page, userPermissions: string[] = []) {
    await page.addInitScript(() => {
        localStorage.setItem('isAuthenticated', 'true');
        localStorage.setItem('token', 'test-token');
    });
    await page.route('**/api/session', (route) =>
        route.fulfill({
            json: {
                user: { email: 'user@example.com' },
                permissions: userPermissions,
                token: 'test-token',
            },
        }),
    );
}

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

    test('authenticated user can navigate directly between views via header links', async ({
        page,
    }) => {
        await authenticate(page, ['users:list']);

        await page.goto('/myevents');
        const nav = page.locator('.header__nav');

        // All navigation links are visible on /myevents
        await expect(nav.getByRole('link', { name: 'Live' })).toBeVisible();
        await expect(nav.getByRole('link', { name: 'Queries' })).toBeVisible();
        await expect(nav.getByRole('link', { name: 'Events' })).toBeVisible();
        await expect(nav.getByRole('link', { name: 'Sensors' })).toBeVisible();
        await expect(nav.getByRole('link', { name: 'Users' })).toBeVisible();

        // Direct navigation to Sensors from Events
        await nav.getByRole('link', { name: 'Sensors' }).click();
        await expect(page).toHaveURL('/mysensors');
        await expect(pageContent(page)).toContainText('Sensors');

        // Direct navigation to Queries from Sensors
        await nav.getByRole('link', { name: 'Queries' }).click();
        await expect(page).toHaveURL('/myqueries');
        await expect(pageContent(page)).toContainText('Saved Queries');

        // Direct navigation to Live from Queries
        await nav.getByRole('link', { name: 'Live' }).click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.home')).toBeVisible();

        // Brand link navigates home as well
        await page.goto('/myevents');
        await page.locator('.header__brand').click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.home')).toBeVisible();
    });

    test('active link state reflects current page visually and via aria-current="page"', async ({
        page,
    }) => {
        await authenticate(page);
        await page.goto('/');

        const nav = page.locator('.header__nav');
        const liveLink = nav.getByRole('link', { name: 'Live' });
        const queriesLink = nav.getByRole('link', { name: 'Queries' });

        await expect(liveLink).toHaveClass(/header__link--active/);
        await expect(liveLink).toHaveAttribute('aria-current', 'page');
        await expect(queriesLink).not.toHaveClass(/header__link--active/);
        await expect(queriesLink).not.toHaveAttribute('aria-current', 'page');

        await queriesLink.click();
        await expect(page).toHaveURL('/myqueries');
        await expect(queriesLink).toHaveClass(/header__link--active/);
        await expect(queriesLink).toHaveAttribute('aria-current', 'page');
        await expect(liveLink).not.toHaveClass(/header__link--active/);
        await expect(liveLink).not.toHaveAttribute('aria-current', 'page');
    });

    test('Users link is permission-gated behind users:list', async ({ page }) => {
        // Authenticate without users:list permission
        await authenticate(page, []);
        await page.goto('/');

        const nav = page.locator('.header__nav');
        await expect(nav.getByRole('link', { name: 'Live' })).toBeVisible();
        await expect(nav.getByRole('link', { name: 'Users' })).toHaveCount(0);
    });

    test('logged-out visitors on subpages see Go back fallback and no main nav', async ({
        page,
    }) => {
        await page.goto('/myqueries');

        // Authenticated nav is hidden
        await expect(page.locator('.header__nav')).toHaveCount(0);

        // "Go back" link is displayed
        const goBack = page.locator('.header__left a.header__link', { hasText: 'Go back' });
        await expect(goBack).toBeVisible();

        // Clicking "Go back" navigates home
        await goBack.click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.home')).toBeVisible();

        // On home, "Go back" is not rendered
        await expect(
            page.locator('.header__left a.header__link', { hasText: 'Go back' }),
        ).toHaveCount(0);
    });

    test('shared event page shows Go back fallback for both logged-in and logged-out users without main nav', async ({
        page,
    }) => {
        // Logged-in user on /events/abc
        await authenticate(page, ['users:list']);
        await page.goto('/events/abc');

        // Main nav is not rendered on shared event link
        await expect(page.locator('.header__nav')).toHaveCount(0);

        // "Go back" link is displayed
        const goBack = page.locator('.header__left a.header__link', { hasText: 'Go back' });
        await expect(goBack).toBeVisible();

        // Clicking "Go back" navigates home to Live stream
        await goBack.click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.home')).toBeVisible();
        await expect(page.locator('.header__nav')).toBeVisible();

        // Logged-out visitor on /events/abc
        await page.evaluate(() => {
            localStorage.clear();
        });
        await page.goto('/events/abc');
        await expect(page.locator('.header__nav')).toHaveCount(0);
        const loggedOutGoBack = page.locator('.header__left a.header__link', {
            hasText: 'Go back',
        });
        await expect(loggedOutGoBack).toBeVisible();
        await loggedOutGoBack.click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.home')).toBeVisible();
    });

    test('filter component mounts on Live view and unmounts on subpages', async ({ page }) => {
        await authenticate(page);
        await page.goto('/');

        // Filter is mounted on home
        await expect(page.locator('.filter-input')).toBeVisible();

        // Navigate to queries page - Filter should be unmounted
        await page.locator('.header__nav').getByRole('link', { name: 'Queries' }).click();
        await expect(page).toHaveURL('/myqueries');
        await expect(page.locator('.filter-input')).toHaveCount(0);

        // Navigate back to Live - Filter should be mounted again
        await page.locator('.header__nav').getByRole('link', { name: 'Live' }).click();
        await expect(page).toHaveURL('/');
        await expect(page.locator('.filter-input')).toBeVisible();
    });

    test('pages with many items scroll properly in shell body (queries, sensors, users)', async ({
        page,
    }) => {
        const mockQueries = Array.from({ length: 40 }, (_, i) => ({
            id: `query-${i}`,
            content: `tcp.port == ${i}`,
            description: `Query description ${i}`,
        }));
        const mockSensors = Array.from({ length: 40 }, (_, i) => ({
            id: `sensor-uuid-${i}`,
            name: `Sensor #${i}`,
        }));
        const mockUsers = Array.from({ length: 40 }, (_, i) => ({
            id: `user-uuid-${i}`,
            email: `user${i}@example.com`,
            roles: ['user'],
        }));

        await authenticate(page, ['users:list', 'sensors:add_by_id']);

        await page.route('**/api/**', (route) => {
            const url = route.request().url();
            if (url.endsWith('/api/session')) {
                return route.fulfill({
                    json: {
                        user: { email: 'admin@example.com' },
                        permissions: ['users:list', 'sensors:add_by_id'],
                        token: 'test-token',
                    },
                });
            }
            if (url.endsWith('/api/queries')) {
                return route.fulfill({ json: mockQueries });
            }
            if (url.endsWith('/api/sensors')) {
                return route.fulfill({ json: mockSensors });
            }
            if (url.endsWith('/api/users')) {
                return route.fulfill({ json: mockUsers });
            }
            return route.fulfill({ status: 401, json: {} });
        });

        // 1. Verify queries scrolling
        await page.goto('/myqueries');
        await expect(page.locator('.queryList__item')).toHaveCount(40);
        const body = page.locator('.shell__body');
        expect(await body.evaluate((el) => el.scrollHeight > el.clientHeight)).toBe(true);
        const lastQuery = page.locator('.queryList__item').last();
        await lastQuery.scrollIntoViewIfNeeded();
        await expect(lastQuery).toBeInViewport();

        // 2. Verify sensors scrolling
        await page.goto('/mysensors');
        await expect(page.locator('.sensorList__item')).toHaveCount(40);
        expect(await body.evaluate((el) => el.scrollHeight > el.clientHeight)).toBe(true);
        const lastSensor = page.locator('.sensorList__item').last();
        await lastSensor.scrollIntoViewIfNeeded();
        await expect(lastSensor).toBeInViewport();

        // 3. Verify users scrolling
        await page.goto('/users');
        await expect(page.locator('.userList__item')).toHaveCount(40);
        expect(await body.evaluate((el) => el.scrollHeight > el.clientHeight)).toBe(true);
        const lastUser = page.locator('.userList__item').last();
        await lastUser.scrollIntoViewIfNeeded();
        await expect(lastUser).toBeInViewport();
    });

    test('header preserves repository link to GitHub', async ({ page }) => {
        await page.goto('/');

        const githubLink = page.locator('.header__right a.header__link', { hasText: 'GitHub' });
        await expect(githubLink).toBeVisible();
        await expect(githubLink).toHaveAttribute('href', 'https://github.com/honeynet/ochi');
        await expect(githubLink).toHaveAttribute('target', '_blank');
    });
});
