import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import type { User } from '@/types/entities';
import { authService } from '@/services/auth.service';

interface AuthState {
  /** Token expiry as epoch ms — the token itself lives in an HttpOnly cookie. */
  expiresAt: number | null;
  user: User | null;
  status: 'idle' | 'loading' | 'authenticated' | 'anonymous';
  error: string | null;
}

const initialState: AuthState = {
  expiresAt: null,
  user: null,
  status: 'idle', // boot: restoreSession decides authenticated vs anonymous
  error: null,
};

// Terminal signed-out state: the login page renders, no boot probe is needed
// (the session cookie is already gone).
function anonymousState(): AuthState {
  return { ...initialState, status: 'anonymous' };
}

export const login = createAsyncThunk('auth/login', async (input: { email: string; password: string }) => {
  return authService.login(input);
});

/**
 * Boot-time session restore. The session lives in an HttpOnly cookie that JS
 * cannot read, so we probe it by rotating: /auth/refresh returns the user and
 * a fresh CSRF value when a valid session cookie exists, and 401s otherwise.
 */
export const restoreSession = createAsyncThunk('auth/restore', async () => {
  return authService.refresh();
});

export const refreshSession = createAsyncThunk('auth/refresh', async () => {
  const session = await authService.refresh();
  if (!session) {
    throw Object.assign(new Error('session expired'), { code: 401 });
  }
  return session;
});

export const logout = createAsyncThunk('auth/logout', async () => {
  await authService.logout();
  return null;
});

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    clearSession() {
      return anonymousState();
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(login.pending, (state) => {
        state.status = 'loading';
        state.error = null;
      })
      .addCase(login.fulfilled, (state, action) => {
        state.status = 'authenticated';
        state.expiresAt = action.payload.expiresAt;
        state.user = action.payload.user;
      })
      .addCase(login.rejected, (state, action) => {
        state.status = 'anonymous';
        state.error = action.error.message ?? 'Login failed';
      })
      .addCase(restoreSession.pending, (state) => {
        state.status = 'loading';
      })
      .addCase(restoreSession.fulfilled, (state, action) => {
        if (action.payload) {
          state.status = 'authenticated';
          state.expiresAt = action.payload.expiresAt;
          state.user = action.payload.user;
        } else {
          state.status = 'anonymous';
        }
      })
      .addCase(restoreSession.rejected, (state) => {
        state.status = 'anonymous';
      })
      .addCase(refreshSession.fulfilled, (state, action) => {
        state.status = 'authenticated';
        state.expiresAt = action.payload.expiresAt;
        state.user = action.payload.user;
      })
      .addCase(refreshSession.rejected, (state, action) => {
        // A network blip (ApiError code 0) must not destroy a valid session —
        // the next focus/timer attempt can succeed. Only a real auth failure
        // (401 / invalid session) logs the user out.
        const code = (action.error as { code?: number } | undefined)?.code;
        if (code === 0) {
          if (state.status === 'idle') state.status = 'anonymous';
          return;
        }
        return anonymousState();
      })
      .addCase(logout.fulfilled, () => anonymousState());
  },
});

export const { clearSession } = authSlice.actions;

export default authSlice.reducer;
