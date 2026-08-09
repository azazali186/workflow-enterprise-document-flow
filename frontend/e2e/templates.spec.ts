import { expect, test } from '@playwright/test';

const ADMIN = { email: 'admin@aeroxe.io', password: 'ChangeMe123!' };

test('admin can create a template through the UI', async ({ page }) => {
  const stamp = Date.now();
  const name = `E2E Template ${stamp}`;
  const slug = `e2e-template-${stamp}`;

  await page.goto('/login');
  await page.getByLabel(/email address/i).fill(ADMIN.email);
  await page.getByLabel(/password/i).fill(ADMIN.password);
  await page.getByRole('button', { name: /sign in|log in/i }).click();
  await expect(page.getByRole('link', { name: /templates/i })).toBeVisible();

  await page.getByRole('link', { name: /templates/i }).click();
  await page.getByRole('button', { name: /new template/i }).click();

  await page.getByLabel(/name/i).fill(name);
  await page.getByLabel(/slug/i).fill(slug);
  await page.getByLabel(/description/i).fill('Created by the E2E suite');
  await page.getByRole('button', { name: /create template/i }).click();

  // Success toast + the new row is listed.
  await expect(page.getByText(/template created/i)).toBeVisible();
  await expect(page.getByText(name, { exact: true })).toBeVisible();
});
