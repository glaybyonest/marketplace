import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { AuthResponse } from '@/types/api'
import type { User } from '@/types/domain'
import { normalizeUser } from '@/utils/normalize'
import { storage } from '@/utils/storage'

interface Credentials {
  email: string
  password: string
}

interface RegisterPayload extends Credentials {
  name: string
}

const normalizeAuthResponse = (raw: unknown): AuthResponse<User> => {
  const source = (pickData<Record<string, unknown>>(raw) ?? {}) as Record<string, unknown>
  const tokens = (source.tokens ?? {}) as Record<string, unknown>

  const token = (tokens.access_token ?? source.token ?? source.accessToken ?? '') as string
  const refreshToken = (tokens.refresh_token ?? source.refreshToken ?? '') as string
  const tokenType = (tokens.token_type ?? 'Bearer') as string
  const expiresIn = Number(tokens.expires_in ?? 0) || 0
  const user = normalizeUser(source.user ?? source.profile ?? {})

  return { token, refreshToken, tokenType, expiresIn, user }
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

  async logout(): Promise<void> {
    const refreshToken = storage.getRefreshToken()
    if (!refreshToken) {
      return
    }

    await apiClient.post('/v1/auth/logout', {
      refresh_token: refreshToken,
    })
  },
}

