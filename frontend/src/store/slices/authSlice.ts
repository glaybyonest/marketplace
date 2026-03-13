import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'

import { isCookieAuthMode } from '@/config/auth'
import { authService } from '@/services/authService'
import type { ApiError, AuthResponse } from '@/types/api'
import type { User } from '@/types/domain'
import { toApiError } from '@/utils/error'
import { storage } from '@/utils/storage'

import type { AsyncStatus } from '@/store/types'

interface AuthState {
  token: string | null
  refreshToken: string | null
  user: User | null
  isAuthenticated: boolean
  status: AsyncStatus
  error: string | null
  errorCode: string | null
  notice: string | null
  requiresEmailVerification: boolean
  sessionBootstrapped: boolean
}

const initialToken = storage.getAccessToken()
const initialRefreshToken = storage.getRefreshToken()
const shouldBootstrapSession = isCookieAuthMode || Boolean(initialToken)

const initialState: AuthState = {
  token: initialToken,
  refreshToken: initialRefreshToken,
  user: null,
  isAuthenticated: Boolean(initialToken) && !isCookieAuthMode,
  status: 'idle',
  error: null,
  errorCode: null,
  notice: null,
  requiresEmailVerification: false,
  sessionBootstrapped: !shouldBootstrapSession,
}

const applySuccessfulAuth = (state: AuthState, payload: AuthResponse<User>) => {
  state.user = payload.user
  state.notice = payload.message ?? null
  state.requiresEmailVerification = payload.requiresEmailVerification
  state.error = null
  state.errorCode = null
  state.sessionBootstrapped = true

  if (payload.token && payload.refreshToken) {
    state.token = payload.token
    state.refreshToken = payload.refreshToken
    state.isAuthenticated = true
    storage.setTokens(payload.token, payload.refreshToken)
    return
  }

  state.token = null
  state.refreshToken = null
  state.isAuthenticated = isCookieAuthMode && !payload.requiresEmailVerification && Boolean(payload.user?.id)
  storage.clearTokens()
}

const applyFailedAuth = (state: AuthState, payload: ApiError | undefined) => {
  state.status = 'failed'
  state.error = payload?.message ?? 'Request failed'
  state.errorCode = payload?.code ?? null
  state.notice = null
  state.requiresEmailVerification = payload?.code === 'email_not_verified'
}

export const loginThunk = createAsyncThunk<
  AuthResponse<User>,
  { email: string; password: string },
  { rejectValue: ApiError }
>('auth/login', async (payload, { rejectWithValue }) => {
  try {
    return await authService.login(payload)
  } catch (error) {
    return rejectWithValue(toApiError(error))
  }
})

export const registerThunk = createAsyncThunk<
  AuthResponse<User>,
  { name: string; email: string; password: string },
  { rejectValue: ApiError }
>('auth/register', async (payload, { rejectWithValue }) => {
  try {
    return await authService.register(payload)
  } catch (error) {
    return rejectWithValue(toApiError(error))
  }
})

export const logoutThunk = createAsyncThunk<void, void, { rejectValue: ApiError }>(
  'auth/logout',
  async (_, { rejectWithValue }) => {
    try {
      await authService.logout()
    } catch (error) {
      return rejectWithValue(toApiError(error))
    }
  },
)

const authSlice = createSlice({
  name: 'auth',
  initialState,
  reducers: {
    setUser(state, action: { payload: User | null }) {
      state.user = action.payload
      state.isAuthenticated = isCookieAuthMode ? Boolean(action.payload) : Boolean(state.token && action.payload)
      state.sessionBootstrapped = true
    },
    completeSessionBootstrap(state) {
      state.sessionBootstrapped = true
    },
    clearAuthFeedback(state) {
      state.error = null
      state.errorCode = null
      state.notice = null
      state.requiresEmailVerification = false
    },
    forceLogout(state) {
      storage.clearTokens()
      state.token = null
      state.refreshToken = null
      state.user = null
      state.isAuthenticated = false
      state.status = 'idle'
      state.error = null
      state.errorCode = null
      state.notice = null
      state.requiresEmailVerification = false
      state.sessionBootstrapped = true
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(loginThunk.pending, (state) => {
        state.status = 'loading'
        state.error = null
        state.errorCode = null
        state.notice = null
      })
      .addCase(loginThunk.fulfilled, (state, action) => {
        state.status = 'succeeded'
        applySuccessfulAuth(state, action.payload)
      })
      .addCase(loginThunk.rejected, (state, action) => {
        applyFailedAuth(state, action.payload)
      })
      .addCase(registerThunk.pending, (state) => {
        state.status = 'loading'
        state.error = null
        state.errorCode = null
        state.notice = null
      })
      .addCase(registerThunk.fulfilled, (state, action) => {
        state.status = 'succeeded'
        applySuccessfulAuth(state, action.payload)
      })
      .addCase(registerThunk.rejected, (state, action) => {
        applyFailedAuth(state, action.payload)
      })
      .addCase(logoutThunk.fulfilled, (state) => {
        state.status = 'idle'
        state.token = null
        state.refreshToken = null
        state.user = null
        state.isAuthenticated = false
        state.error = null
        state.errorCode = null
        state.notice = null
        state.requiresEmailVerification = false
        state.sessionBootstrapped = true
        storage.clearTokens()
      })
      .addCase(logoutThunk.rejected, (state, action) => {
        state.error = action.payload?.message ?? 'Failed to logout'
        state.errorCode = action.payload?.code ?? null
        state.token = null
        state.refreshToken = null
        state.user = null
        state.isAuthenticated = false
        state.notice = null
        state.requiresEmailVerification = false
        state.sessionBootstrapped = true
        storage.clearTokens()
      })
  },
})

export const { setUser, completeSessionBootstrap, clearAuthFeedback, forceLogout } = authSlice.actions
export default authSlice.reducer
