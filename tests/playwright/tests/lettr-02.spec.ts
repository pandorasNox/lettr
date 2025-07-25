
import { test, expect } from '@playwright/test';


// test('solution word from help hint should be successful solve inserted via keyboard press', async ({ page }, testInfo) => {
test('allows solving the word from help hint via keyboard input', async ({ page }, testInfo) => {
    // await page.goto('http://lettrapp.aliases.containernetwork/');
    console.log('Base URL:', testInfo.project.use?.baseURL);
    await page.goto('/');

    // toggle dark mode
    await expect(page.locator('#theme-toggle')).toBeVisible();
    await expect(page.locator('#theme-toggle')).toBeInViewport();
    await page.locator('#theme-toggle').click();

    // expect game form
    await page.waitForSelector('form'); // or: await page.locator('form').waitFor()
    const form = await page.getByRole('form');
    await expect(form).toHaveCount(1);
    await form.waitFor({ state: 'visible' }); // OR 'attached' if it's hidden initially

    await expect(form).toHaveCount(1);
    await expect(form).toBeVisible();

    await expect(page.locator('[data-testid="help-btn"]')).toHaveCount(1);

    await page.locator('[data-testid="help-btn"]').click();
    // await page.getByRole('button', { name: '?' }).click();

    await page.locator('label').filter({ hasText: 'Show solution' }).click()

    const maybeSolution = await page.locator('[data-testid="solution"]').textContent();
    expect(maybeSolution).not.toBeNull();
    const solution = maybeSolution as string;

    await page.locator('[data-testid="back-btn"]').click();

    await expect(page.getByRole('form')).toBeVisible();

    // for (const letter of solution) {
    for (const [i, letter] of Array.from(solution).entries()) {
        await page.keyboard.press(letter);
        await expect(page.locator('input[name="r0"]').nth(i)).toHaveValue(letter);
    }

    await page.keyboard.press("Enter");
});
