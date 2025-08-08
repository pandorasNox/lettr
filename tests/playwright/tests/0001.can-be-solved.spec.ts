
import { test, expect } from '@playwright/test';
import { ApplicationPage } from './page.application';

// let jsErrors: string[] = [];
let jsErrors: { console: string[], page: string[]} = { console: [], page: []};

test.beforeEach(async ({ page }) => {
    jsErrors = { console: [], page: []};

    page.on('console', msg => {
        if (msg.type() === 'error') {
            jsErrors.console.push(msg.text());
        }
    });

    page.on('pageerror', err => {
        jsErrors.page.push(err.message);
    });
});

test.afterEach(async () => {
    expect(jsErrors.console, "js console errors should be zero").toHaveLength(0);
    expect(jsErrors.page, "js page errors should be zero").toHaveLength(0);
});

// test('solution word from help hint should be successful solve inserted via keyboard press', async ({ page }, testInfo) => {
test('allows solving the word from help hint via keyboard input', async ({ page }, testInfo) => {
    // await page.goto('http://lettrapp.aliases.containernetwork/');
    console.log('Base URL:', testInfo.project.use?.baseURL);
    await page.goto('/');

    await test.step('toggle dark mode', async () => {
        const applicaltionPage = new ApplicationPage(page);
        await applicaltionPage.toggleTheme('dark');
        expect(applicaltionPage.theme).toBe('dark');
    });

    // await expect(page.getByRole('heading', { name: 'lettr' })).toBeVisible();
    // await expect(page.getByRole('heading', { name: 'lettr' })).toHaveText('lettr');

    await test.step('solve puzzle', async () => {
        await expect(page.getByTestId('show-result-unsolved')).toBeVisible();

        const solution = await test.step('fetch solution from help', async () => {
            await expect(page.getByTestId('help-btn')).toHaveCount(1);

            await page.getByTestId('help-btn').click();
            // await page.getByRole('button', { name: '?' }).click();

            await page.locator('label').filter({ hasText: 'Show solution' }).click()

            const maybeSolution = await page.getByTestId('solution').textContent();
            expect(maybeSolution).not.toBeNull();

            await page.getByTestId('back-btn').click();

            return maybeSolution as string;
        });

        await test.step('enter solution and solve puzzle', async () => {
            await page.waitForSelector('form');
            await expect(page.locator('form')).toBeVisible();

            // for (const letter of solution) {
            for (const [i, letter] of Array.from(solution).entries()) {
                await page.keyboard.press(letter);
                await expect(page.locator('input[name="r0"]').nth(i)).toHaveValue(letter);
            }

            await page.keyboard.press("Enter");
            // await expect(page.getByRole('heading', { name: 'SOLVED' })).toBeVisible();
            await expect(page.getByTestId('show-result-unsolved')).not.toBeVisible();
            await expect(page.getByTestId('show-result-solved')).toBeVisible();
        });
    });
});
