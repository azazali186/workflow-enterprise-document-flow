import { refreshSession } from '@/store/authSlice';
import { store } from '@/store';

let timer: ReturnType<typeof setTimeout> | null = null;

const MIN_DELAY_MS = 60_000; // never poll more than once a minute
const MAX_DELAY_MS = 30 * 60_000; // and never sleep longer than 30 min

function scheduleNext() {
  if (timer) clearTimeout(timer);
  // The token itself is HttpOnly (unreadable from JS); its expiry travels in
  // the login/refresh response body and lives in the store.
  const expiresAt = store.getState().auth.expiresAt;
  if (!expiresAt) return;
  const remaining = expiresAt - Date.now();
  if (remaining <= 0) {
    void store.dispatch(refreshSession());
    return;
  }
  // Refresh at half the remaining life, clamped to [1 min, 30 min] so a 24h
  // token still rotates at least every 30 minutes (keeps the Redis SSO entry
  // alive and bounds the exposure window of a stolen token).
  const delay = Math.min(MAX_DELAY_MS, Math.max(MIN_DELAY_MS, remaining / 2));
  timer = setTimeout(async () => {
    const result = await store.dispatch(refreshSession());
    if (result.meta.requestStatus === 'rejected') return; // session gone; guard handles it
    scheduleNext();
  }, delay);
}

/** Keeps the session token fresh while the app is open. Call once at boot. */
export function startSessionRefresher(): () => void {
  const onFocus = () => {
    // After a long sleep the token may be expired — try a silent rotation.
    const expiresAt = store.getState().auth.expiresAt;
    if (!expiresAt) return;
    if (expiresAt - Date.now() < 10 * 60_000) {
      void store.dispatch(refreshSession());
    }
    scheduleNext();
  };

  window.addEventListener('focus', onFocus);
  scheduleNext();

  return () => {
    if (timer) clearTimeout(timer);
    window.removeEventListener('focus', onFocus);
  };
}
