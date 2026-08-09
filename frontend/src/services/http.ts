import { ApiError, type ApiEnvelope } from '@/types/api';

const CSRF_HEADER = 'X-CSRF-Token';

/** In-memory 401 handler registry so the client stays store-agnostic. */
type UnauthorizedHandler = () => void;
let onUnauthorized: UnauthorizedHandler | null = null;

/**
 * The session token lives in an HttpOnly, SameSite=Lax cookie the browser
 * manages — JS can never read it. The CSRF double-submit value travels in the
 * login/refresh response body and is held in memory only (never persisted, so
 * it disappears with the tab).
 */
let csrfToken: string | null = null;

export function setUnauthorizedHandler(fn: UnauthorizedHandler | null) {
  onUnauthorized = fn;
}

/** Stores the CSRF value delivered by the auth endpoints. */
export function setCsrfToken(value: string | null) {
  csrfToken = value || null;
}

interface RequestOptions {
  method?: 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  /** Skip the Authorization/CSRF handling (login/register). */
  public?: boolean;
  /** Abort signal for React Query. */
  signal?: AbortSignal;
  /** Allow one 401-triggered refresh retry (all authed calls). */
  allowRefresh?: boolean;
}

// Single-flight refresh: concurrent 401s share one refresh instead of racing.
let refreshInFlight: Promise<boolean> | null = null;
// After a failed refresh, hold the failure for a short window so a burst of
// 401s during an outage doesn't hammer /auth/refresh once per request.
let lastRefreshFailureAt = 0;
const REFRESH_COOLDOWN_MS = 5_000;

async function doRefresh(): Promise<boolean> {
  try {
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include', // the HttpOnly session cookie travels automatically
    });
    if (!res.ok) return false;
    const envelope = (await res.json()) as ApiEnvelope<{ csrf?: string }>;
    if (envelope.code !== 0) return false;
    // A fresh token means a fresh CSRF binding — swap it for the retried call.
    if (envelope.data?.csrf) setCsrfToken(envelope.data.csrf);
    return true;
  } catch {
    return false;
  }
}

function refreshTokenOnce(): Promise<boolean> {
  if (!refreshInFlight) {
    if (Date.now() - lastRefreshFailureAt < REFRESH_COOLDOWN_MS) {
      return Promise.resolve(false);
    }
    refreshInFlight = doRefresh()
      .then((ok) => {
        // Stamp the cooldown only when a refresh was actually attempted.
        if (!ok && csrfToken !== null) lastRefreshFailureAt = Date.now();
        return ok;
      })
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
}

async function doRequest<T>(path: string, options: RequestOptions): Promise<T> {
  const { method = 'POST', body, public: isPublic, signal, allowRefresh = true } = options;

  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (csrfToken) headers[CSRF_HEADER] = csrfToken;

  let res: Response;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
      credentials: 'include', // same-origin via the Vite/nginx proxy
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiError('Network error — check your connection', 0, 0);
  }

  // Any non-2xx is abnormal for this API, but handle it defensively.
  if (!res.ok) {
    throw new ApiError(`Unexpected HTTP ${res.status}`, res.status, res.status);
  }

  let envelope: ApiEnvelope<T>;
  try {
    envelope = (await res.json()) as ApiEnvelope<T>;
  } catch {
    throw new ApiError('Malformed response from server', 500, res.status);
  }

  // Expired session: try one silent refresh, then retry the request once.
  // The auth layer is notified whenever the final outcome is an unhandled
  // 401 — including a retry that 401s again — so a stale session always
  // clears instead of leaving the UI stuck on a dead token.
  if (envelope.code === 401 && !isPublic) {
    if (allowRefresh) {
      const refreshed = await refreshTokenOnce();
      if (refreshed) {
        return doRequest<T>(path, { ...options, allowRefresh: false });
      }
    }
    onUnauthorized?.();
  }

  if (envelope.code !== 0) {
    throw new ApiError(envelope.message || 'Request failed', envelope.code);
  }
  return envelope.data as T;
}

/** POST helper (most endpoints). */
export function post<T>(path: string, body?: unknown, options?: Partial<RequestOptions>): Promise<T> {
  return doRequest<T>(path, { ...options, method: 'POST', body });
}

/** PATCH helper (updates). */
export function patch<T>(path: string, body?: unknown, options?: Partial<RequestOptions>): Promise<T> {
  return doRequest<T>(path, { ...options, method: 'PATCH', body });
}

/** DELETE helper (soft-delete by id in body). */
export function del<T>(path: string, body?: unknown): Promise<T> {
  return doRequest<T>(path, { method: 'DELETE', body });
}

/**
 * Test-only: clears the single-flight/cooldown state between test cases.
 */
export function __resetRefreshStateForTests() {
  refreshInFlight = null;
  lastRefreshFailureAt = 0;
}
