import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { post, setCsrfToken, setUnauthorizedHandler, __resetRefreshStateForTests } from './http';

function okEnvelope(data: unknown, code = 0) {
  return new Response(JSON.stringify({ code, message: '', data }), { status: 200, headers: { 'Content-Type': 'application/json' } });
}

function refreshEnvelope(csrf?: string) {
  return okEnvelope(csrf ? { csrf } : {});
}

describe('http client envelope handling', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    setCsrfToken('csrf-abc');
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    setCsrfToken(null);
    __resetRefreshStateForTests();
  });

  it('unwraps the envelope data on success', async () => {
    vi.mocked(fetch).mockResolvedValue(okEnvelope({ id: 'u1' }));
    await expect(post('/x', {}, { public: true })).resolves.toEqual({ id: 'u1' });
  });

  it('sends the CSRF header when a session csrf is held', async () => {
    vi.mocked(fetch).mockResolvedValue(okEnvelope(null));
    await post('/x', {});
    const init = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
    expect(init.headers).toMatchObject({ 'X-CSRF-Token': 'csrf-abc' });
    expect(init.credentials).toBe('include');
  });

  it('omits the CSRF header when no session csrf is held', async () => {
    setCsrfToken(null);
    vi.mocked(fetch).mockResolvedValue(okEnvelope(null));
    await post('/x', {}, { public: true });
    const init = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
    expect(init.headers).not.toHaveProperty('X-CSRF-Token');
  });

  it('throws ApiError with the business code when code !== 0', async () => {
    vi.mocked(fetch).mockResolvedValue(okEnvelope(null, 403));
    await expect(post('/x', {}, { public: true })).rejects.toMatchObject({ code: 403 });
  });

  it('normalizes a network failure into a friendly ApiError', async () => {
    vi.mocked(fetch).mockRejectedValue(new TypeError('Failed to fetch'));
    await expect(post('/x', {}, { public: true })).rejects.toMatchObject({ code: 0 });
  });

  it('rethrows abort errors without converting them', async () => {
    vi.mocked(fetch).mockRejectedValue(new DOMException('aborted', 'AbortError'));
    await expect(post('/x', {}, { public: true, signal: new AbortController().signal })).rejects.toMatchObject({ name: 'AbortError' });
  });

  it('throws ApiError on malformed JSON responses', async () => {
    vi.mocked(fetch).mockResolvedValue(new Response('not-json', { status: 200 }));
    await expect(post('/x', {}, { public: true })).rejects.toMatchObject({ code: 500 });
  });

  describe('single-flight 401 refresh', () => {
    it('refreshes once via the session cookie and retries the request on 401', async () => {
      const onUnauthorized = vi.fn();
      setUnauthorizedHandler(onUnauthorized);

      vi.mocked(fetch)
        .mockResolvedValueOnce(okEnvelope(null, 401))
        .mockResolvedValueOnce(refreshEnvelope('csrf-fresh'))
        .mockResolvedValueOnce(okEnvelope({ ok: true }));

      await expect(post('/x', {})).resolves.toEqual({ ok: true });
      expect(onUnauthorized).not.toHaveBeenCalled();

      // The refresh call itself carries no CSRF header (public + exempt) and
      // relies on the HttpOnly cookie; its response updates the held csrf.
      const refreshCall = vi.mocked(fetch).mock.calls[1];
      expect(refreshCall[0]).toBe('/api/v1/auth/refresh');
      const refreshInit = refreshCall[1] as RequestInit;
      expect(refreshInit.headers).not.toHaveProperty('X-CSRF-Token');
      const retryInit = vi.mocked(fetch).mock.calls[2][1] as RequestInit;
      expect(retryInit.headers).toMatchObject({ 'X-CSRF-Token': 'csrf-fresh' });
    });

    it('does not retry after a failed refresh', async () => {
      const onUnauthorized = vi.fn();
      setUnauthorizedHandler(onUnauthorized);
      // Fresh Response per call: a shared instance drains its body on first read.
      vi.mocked(fetch).mockImplementation(async () => okEnvelope(null, 401));
      await expect(post('/x', {})).rejects.toMatchObject({ code: 401 });
      expect(onUnauthorized).toHaveBeenCalledTimes(1);
      // Exactly: original request + one refresh attempt. No retry, no loop.
      expect(vi.mocked(fetch).mock.calls.length).toBe(2);
    });

    it('notifies the auth layer when refresh fails', async () => {
      const onUnauthorized = vi.fn();
      setUnauthorizedHandler(onUnauthorized);
      vi.mocked(fetch).mockResolvedValue(okEnvelope(null, 401));
      await expect(post('/x', {})).rejects.toMatchObject({ code: 401 });
      expect(onUnauthorized).toHaveBeenCalledTimes(1);
    });

    it('shares a single refresh across concurrent 401s', async () => {
      // mockImplementation mints a fresh Response per call; a shared instance
      // would have its body consumed on the first read.
      vi.mocked(fetch)
        .mockResolvedValueOnce(okEnvelope(null, 401))
        .mockResolvedValueOnce(okEnvelope(null, 401))
        .mockResolvedValueOnce(refreshEnvelope('csrf-fresh'))
        .mockImplementation(async () => okEnvelope({ ok: true }));

      const [a, b] = await Promise.all([post('/a', {}), post('/b', {})]);
      expect(a).toEqual({ ok: true });
      expect(b).toEqual({ ok: true });

      const refreshCalls = vi.mocked(fetch).mock.calls.filter(([url]) => url === '/api/v1/auth/refresh');
      expect(refreshCalls.length).toBe(1);
    });

    it('notifies the auth layer when the retried request also 401s', async () => {
      const onUnauthorized = vi.fn();
      setUnauthorizedHandler(onUnauthorized);
      // Refresh succeeds, but the retried request is rejected too.
      vi.mocked(fetch)
        .mockResolvedValueOnce(okEnvelope(null, 401))
        .mockResolvedValueOnce(refreshEnvelope('csrf-fresh'))
        .mockImplementation(async () => okEnvelope(null, 401));

      await expect(post('/x', {})).rejects.toMatchObject({ code: 401 });
      expect(onUnauthorized).toHaveBeenCalledTimes(1);
    });

    it('coalesces refresh attempts after a failure (no retry storm)', async () => {
      // Fresh Response per call: a shared instance drains its body on first read.
      vi.mocked(fetch).mockImplementation(async () => okEnvelope(null, 401));

      // First request: 401 + failed refresh → notify auth layer.
      await expect(post('/a', {})).rejects.toMatchObject({ code: 401 });
      const callsAfterFirst = vi.mocked(fetch).mock.calls.length;

      // Immediately-after request: cooldown window is active → no new refresh.
      await expect(post('/b', {})).rejects.toMatchObject({ code: 401 });
      const callsAfterSecond = vi.mocked(fetch).mock.calls.length;

      expect(callsAfterSecond - callsAfterFirst).toBe(1); // only the request itself
    });
  });
});
