import { post, setCsrfToken } from './http';
import { ApiError } from '@/types/api';
import type { AuthResult, User } from '@/types/entities';

const BASE = '/api/v1/auth';

export interface LoginInput {
  email: string;
  password: string;
}

export interface Session {
  /** Token expiry as epoch ms (in-memory only — the token itself is HttpOnly). */
  expiresAt: number | null;
  user: User;
}

/** Converts the wire AuthResult into the store shape and keeps the CSRF value. */
function toSession(result: AuthResult): Session {
  const expiresAt = result.expires_at ? Date.parse(result.expires_at) : null;
  setCsrfToken(result.csrf ?? null);
  return { expiresAt: Number.isFinite(expiresAt) ? expiresAt : null, user: result.user };
}

export const authService = {
  async login(input: LoginInput): Promise<Session> {
    const result = await post<AuthResult>(`${BASE}/login`, input, { public: true });
    return toSession(result);
  },

  async logout(): Promise<void> {
    try {
      await post<void>(`${BASE}/logout`);
    } finally {
      setCsrfToken(null);
    }
  },

  /**
   * Rotates the session. The HttpOnly cookie is sent automatically; the
   * endpoint is public and CSRF-exempt, so no bearer or CSRF header is
   * needed. Returns null when the session is genuinely gone. A transient
   * network failure propagates as an ApiError so callers can distinguish
   * "expired" from "blip" and keep the session.
   */
  async refresh(): Promise<Session | null> {
    try {
      const result = await post<AuthResult>(`${BASE}/refresh`, undefined, { allowRefresh: false });
      return toSession(result);
    } catch (err) {
      if (err instanceof ApiError && err.code === 0) {
        throw err; // network failure — session may still be valid
      }
      return null;
    }
  },
};
