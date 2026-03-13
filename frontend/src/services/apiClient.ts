import axios, { AxiosError } from 'axios'
import type { InternalAxiosRequestConfig } from 'axios'

import { storage } from '@/utils/storage'

export const AUTH_UNAUTHORIZED_EVENT = 'auth:unauthorized'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api'

const isAuthEndpoint = (url?: string) => {
  if (!url) {
    return false
  }
  return url.includes('/v1/auth/login') || url.includes('/v1/auth/register') || url.includes('/v1/auth/refresh')
}

type RetryableConfig = InternalAxiosRequestConfig & {
  _retry?: boolean
  skipAuthRefresh?: boolean
}

const parseRefreshTokenPair = (raw: unknown): { accessToken: string; refreshToken: string } | null => {
  const envelope = (raw as { data?: unknown }) ?? {}
  const payload = (envelope.data ?? raw) as Record<string, unknown>

  const accessToken = typeof payload.access_token === 'string' ? payload.access_token : ''
  const refreshToken = typeof payload.refresh_token === 'string' ? payload.refresh_token : ''

  if (!accessToken || !refreshToken) {
    return null
  }

  return { accessToken, refreshToken }
}

let refreshPromise: Promise<string | null> | null = null

const refreshAccessToken = async (): Promise<string | null> => {
  if (refreshPromise) {
    return refreshPromise
  }

  refreshPromise = (async () => {
    const refreshToken = storage.getRefreshToken()
    if (!refreshToken) {
      return null
    }

    try {
      const response = await apiClient.post(
        '/v1/auth/refresh',
        { refresh_token: refreshToken },
        { skipAuthRefresh: true } as RetryableConfig,
      )
      const tokenPair = parseRefreshTokenPair(response.data)
      if (!tokenPair) {
        return null
      }
      storage.setTokens(tokenPair.accessToken, tokenPair.refreshToken)
      return tokenPair.accessToken
    } catch {
      return null
    }
  })()

  try {
    return await refreshPromise
  } finally {
    refreshPromise = null
  }
}

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 15000,
})

apiClient.interceptors.request.use((config) => {
  const token = storage.getAccessToken()

  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = (error.config ?? {}) as RetryableConfig

    if (error.response?.status !== 401 || originalRequest.skipAuthRefresh || originalRequest._retry || isAuthEndpoint(originalRequest.url)) {
      if (error.response?.status === 401 && (originalRequest.skipAuthRefresh || isAuthEndpoint(originalRequest.url))) {
        storage.clearTokens()
        window.dispatchEvent(new Event(AUTH_UNAUTHORIZED_EVENT))
      }
      return Promise.reject(error)
    }

    originalRequest._retry = true

    const nextAccessToken = await refreshAccessToken()

    if (!nextAccessToken) {
      storage.clearTokens()
      window.dispatchEvent(new Event(AUTH_UNAUTHORIZED_EVENT))
      return Promise.reject(error)
    }

    if (!originalRequest.headers) {
      originalRequest.headers = {} as RetryableConfig['headers']
    }
    originalRequest.headers.Authorization = `Bearer ${nextAccessToken}`
    return apiClient.request(originalRequest)
  },
)

