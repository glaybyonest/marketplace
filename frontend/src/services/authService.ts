import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { AuthResponse } from '@/types/api'
import type { User } from '@/types/domain'
import { normalizeUser } from '@/utils/normalize'
import { storage } from '@/utils/storage'
import { isCookieAuthMode } from '@/config/auth'

interface Credentials {
  email: string
  password: string
}

interface RegisterPayload extends Credentials {
  name: string
}

interface ActionAcceptedResponse {
  accepted: boolean
}

interface VerifyEmailResult {
  verified: boolean
  user: User
}

const normalizeAuthResponse = (raw: unknown): AuthResponse<User> => {
  const source = (pickData<Record<string, unknown>>(raw) ?? {}) as Record<string, unknown>
  const tokens = (source.tokens ?? {}) as Record<string, unknown>

  const token =
    typeof tokens.access_token === 'string' && tokens.access_token.trim().length > 0 ? tokens.access_token : null
  const refreshToken =
    typeof tokens.refresh_token === 'string' && tokens.refresh_token.trim().length > 0 ? tokens.refresh_token : null
  const tokenType = typeof tokens.token_type === 'string' ? tokens.token_type : null
  const expiresIn = Number(tokens.expires_in ?? 0) || 0
  const user = normalizeUser(source.user ?? source.profile ?? {})
  const requiresEmailVerification = Boolean(source.requires_email_verification)
  const message = typeof source.message === 'string' ? source.message : undefined

  return { token, refreshToken, tokenType, expiresIn, user, requiresEmailVerification, message }
}

export const authService = {
  async login(payload: Credentials): Promise<AuthResponse<User>> {
    const response = await apiClient.post('/v1/auth/login', payload)
    return normalizeAuthResponse(response.data)
  },

  async register(payload: RegisterPayload): Promise<AuthResponse<User>> {
    const response = await apiClient.post('/v1/auth/register', {
      email: payload.email,
      password: payload.password,
      full_name: payload.name,
    })
    return normalizeAuthResponse(response.data)
  },

  async requestEmailVerification(email: string): Promise<ActionAcceptedResponse> {
    const response = await apiClient.post('/v1/auth/verify-email/request', { email })
    const source = (pickData<Record<string, unknown>>(response.data) ?? {}) as Record<string, unknown>
    return { accepted: Boolean(source.accepted) }
  },

  async confirmEmailVerification(token: string): Promise<VerifyEmailResult> {
    const response = await apiClient.post('/v1/auth/verify-email/confirm', { token })
    const source = (pickData<Record<string, unknown>>(response.data) ?? {}) as Record<string, unknown>
    return {
      verified: Boolean(source.verified),
      user: normalizeUser(source.user ?? {}),
    }
  },

  async requestPasswordReset(email: string): Promise<ActionAcceptedResponse> {
    const response = await apiClient.post('/v1/auth/password-reset/request', { email })
    const source = (pickData<Record<string, unknown>>(response.data) ?? {}) as Record<string, unknown>
    return { accepted: Boolean(source.accepted) }
  },

  async confirmPasswordReset(token: string, newPassword: string): Promise<boolean> {
    const response = await apiClient.post('/v1/auth/password-reset/confirm', {
      token,
      new_password: newPassword,
    })
    const source = (pickData<Record<string, unknown>>(response.data) ?? {}) as Record<string, unknown>
    return Boolean(source.password_reset)
  },

  async logout(): Promise<void> {
    if (isCookieAuthMode) {
      await apiClient.post('/v1/auth/logout')
      return
    }

    const refreshToken = storage.getRefreshToken()
    if (!refreshToken) {
      return
    }

    await apiClient.post('/v1/auth/logout', {
      refresh_token: refreshToken,
    })
  },
}
