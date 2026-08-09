import '@testing-library/jest-dom/vitest';
import { afterEach, beforeEach, vi } from 'vitest';

function clearStores() {
  try {
    sessionStorage.clear();
  } catch {
    /* jsdom may not expose a real Storage in some environments */
  }
  try {
    localStorage.clear();
  } catch {
    /* ignore */
  }
}

beforeEach(() => {
  clearStores();
});

afterEach(() => {
  vi.restoreAllMocks();
  clearStores();
});
