
import { test, expect } from '@playwright/test';

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

test('has title', async ({ page }, testInfo) => {
    console.log('Base URL:', testInfo.project.use?.baseURL);
//   const appUrl = process.env.APP_URL
    // await page.goto('http://lettrapp.aliases.containernetwork/');
    await page.goto('/');

    await page.locator('#theme-toggle').click()

    await expect(page.locator('nav h1.pl-2')).toBeVisible();
    await expect(page.locator('nav h1.pl-3')).toHaveCount(0);
//   const navHeadlineText = await page.locator('nav h1.pl-3').textContent();
  // Fill the input by targeting the label.
//   await expect(page.locator('nav h1.pl-2')).toHaveText('lettr2');
    await expect(page.getByRole('heading', { name: 'lettr' })).toHaveText('lettr');
    await expect(page.getByRole('heading', { name: 'lettr' })).toBeVisible();

  // Expect a title "to contain" a substring.
//   await expect(page).toHaveTitle(/lettr/);
});

// `nav h1.pl-2`
