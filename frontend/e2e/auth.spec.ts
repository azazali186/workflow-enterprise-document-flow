import { expect, test } from '@playwright/test';

const ADMIN = { email: 'admin@aeroxe.io', password: 'ChangeMe123!' };

async function login(page: import('@playwright/test').Page) {
  await page.goto('/login');
  await page.getByLabel(/email address/i).fill(ADMIN.email);
  await page.getByLabel(/password/i).fill(ADMIN.password);
  await page.getByRole('button', { name: /sign in|log in/i }).click();
}

test('admin can sign in and the session survives a reload', async ({ page }) => {
  await login(page);

  // Lands on the dashboard with the app shell visible.
  await expect(page).toHaveURL(new RegExp('/$'));
  await expect(page.getByRole('link', { name: /documents/i })).toBeVisible();

  // The session lives in an HttpOnly cookie: a hard reload must restore it
  // without a login redirect (boot restore probes the cookie via refresh).
  await page.reload();
  await expect(page.getByRole('link', { name: /documents/i })).toBeVisible();

  // Sign out returns to the login screen and clears the session.
  await page.getByRole('button', { name: /sign out/i }).click();
  await expect(page).toHaveURL(/\/login/);
  await expect(page.getByLabel(/email address/i)).toBeVisible();
});

test('a rejected login shows the error and stays on the login page', async ({ page }) => {
  await page.goto('/login');
  await page.getByLabel(/email address/i).fill(ADMIN.email);
  await page.getByLabel(/password/i).fill('wrong-password');
  await page.getByRole('button', { name: /sign in|log in/i }).click();

  await expect(page.getByText(/invalid credentials/i)).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
});
