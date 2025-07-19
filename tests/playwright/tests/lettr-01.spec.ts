
import { test, expect } from '@playwright/test';

test('has title', async ({ page }) => {
  await page.goto('http://lettrapp.aliases.containernetwork/');

  await expect(page.locator('nav h1.pl-2')).toBeVisible();
  await expect(page.locator('nav h1.pl-3')).toHaveCount(0);
//   const navHeadlineText = await page.locator('nav h1.pl-3').textContent();
  // Fill the input by targeting the label.
  await expect(page.locator('nav h1.pl-2')).toHaveText('lettr');

  // Expect a title "to contain" a substring.
//   await expect(page).toHaveTitle(/lettr/);
});

// `nav h1.pl-2`
