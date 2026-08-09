import { expect, test, type APIRequestContext } from '@playwright/test';

const ADMIN = { email: 'admin@aeroxe.io', password: 'ChangeMe123!' };

interface LoginData {
  code: number;
  data: { token: string; csrf?: string; user: { id: string } };
}

/** Seeds a document + approval chain through the API using a bearer token. */
async function seedApproval(api: APIRequestContext): Promise<{ documentId: string; approvalId: string; title: string }> {
  const login = await api.post('/api/v1/auth/login', { data: { email: ADMIN.email, password: ADMIN.password } });
  const body = (await login.json()) as LoginData;
  if (body.code !== 0) throw new Error(`seed login failed: ${JSON.stringify(body)}`);
  // The API context stores the session cookie, so the CSRF guard expects the
  // double-submit value delivered in the login body — echo it back.
  const headers = {
    Authorization: `Bearer ${body.data.token}`,
    'X-CSRF-Token': body.data.csrf ?? '',
    'Content-Type': 'application/json',
  };

  const title = `E2E Approval Doc ${Date.now()}`;
  const doc = await api.post('/api/v1/documents/create', { headers, data: { title, description: 'Seeded by the E2E suite' } });
  const docBody = (await doc.json()) as { code: number; data: { id: string } };
  if (docBody.code !== 0) throw new Error(`seed document failed: ${JSON.stringify(docBody)}`);

  const chain = await api.post('/api/v1/approvals/create', {
    headers,
    data: { document_id: docBody.data.id, approver_ids: [body.data.user.id] },
  });
  const chainBody = (await chain.json()) as { code: number; data: { id: string } };
  if (chainBody.code !== 0) throw new Error(`seed approval failed: ${JSON.stringify(chainBody)}`);

  return { documentId: docBody.data.id, approvalId: chainBody.data.id, title };
}

test('approver can decide a pending approval end to end', async ({ page, request }) => {
  const seeded = await seedApproval(request);

  // Sign in through the UI (the browser session uses the HttpOnly cookie).
  await page.goto('/login');
  await page.getByLabel(/email address/i).fill(ADMIN.email);
  await page.getByLabel(/password/i).fill(ADMIN.password);
  await page.getByRole('button', { name: /sign in|log in/i }).click();
  await expect(page.getByRole('link', { name: /approvals/i })).toBeVisible();

  await page.getByRole('link', { name: /approvals/i }).click();

  // The seeded pending approval shows up (filter defaults to pending). The
  // row identifies its document by the short id, not the title — and short ids
  // share a time prefix with older runs, so target the newest matching row.
  const row = page.getByRole('row', { name: new RegExp(`Document #${seeded.documentId.slice(0, 8)}`) }).first();
  await expect(row).toBeVisible();

  await row.getByRole('button', { name: /decide/i }).click();
  await page.getByLabel(/comment/i).fill('Looks good — approved by E2E.');
  await page.getByRole('button', { name: /approve/i }).click();

  // Status flips to closed and the toast confirms.
  await expect(page.getByText(/decision recorded/i)).toBeVisible();
  await expect(row.getByText(/closed/i)).toBeVisible();
});
