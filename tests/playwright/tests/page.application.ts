import { expect, type Locator, type Page } from '@playwright/test';

export class ApplicationPage {
  readonly page: Page;
  theme: 'light' | 'dark' = 'light';

  constructor(page: Page) {
    this.page = page;
  }

  async goto() {
    await this.page.goto('/');
  }

  async toggleTheme(theme: 'light' | 'dark') {
    const isDark = ! await this.page.getByTestId('theme-dark').isVisible();
    await expect(this.theme).toBe(isDark ? 'dark' : 'light');

    if (isDark && theme === 'light') {
        await this.page.getByTestId('theme-toggle').click();
        await expect(this.page.getByTestId('theme-dark')).toBeVisible();
    }

    if (!isDark && theme === 'dark') {
        await this.page.getByTestId('theme-toggle').click();
        await expect(this.page.getByTestId('theme-light')).toBeVisible();
    }

    this.theme = theme;
  }
}