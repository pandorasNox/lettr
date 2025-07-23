
import { test, expect } from '@playwright/test';


test('has title', async ({ page }, testInfo) => {
    // await page.goto('http://lettrapp.aliases.containernetwork/');
    console.log('Base URL:', testInfo.project.use?.baseURL);
    await page.goto('/');

    await page.keyboard.press('ArrowDown');

    await page.getByRole('button').locator('[data-testid="help-btn"]').click();
    const solution = await page.locator('[data-testid="solution"]').textContent();
    
    for (let i = 0; i < solution.length; i++) {
        await page.keyboard.press(solution[i]);
    }

});
