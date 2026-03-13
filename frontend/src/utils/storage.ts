import { AUTH_CSRF_COOKIE_NAME, isCookieAuthMode } from '@/config/auth'

const ACCESS_TOKEN_KEY = 'marketplace_access_token'
const REFRESH_TOKEN_KEY = 'marketplace_refresh_token'

const readCookie = (name: string): string | null => {
  if (typeof document === 'undefined') {
    return null
  }

  const prefix = `${encodeURIComponent(name)}=`
  for (const chunk of document.cookie.split(';')) {
    const candidate = chunk.trim()
    if (!candidate.startsWith(prefix)) {
      continue
    }
    return decodeURIComponent(candidate.slice(prefix.length))
  }
  return null
}

export const storage = {
  getToken(): string | null {
    return storage.getAccessToken()
  },
  setToken(token: string) {
    if (isCookieAuthMode) {
      return
    }
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
  },
  clearToken() {
    storage.clearTokens()
  },
  getAccessToken(): string | null {
    if (isCookieAuthMode) {
      return null
    }
    return localStorage.getItem(ACCESS_TOKEN_KEY)
  },
  getRefreshToken(): string | null {
    if (isCookieAuthMode) {
      return null
    }
    return localStorage.getItem(REFRESH_TOKEN_KEY)
  },
  setTokens(accessToken: string, refreshToken: string) {
    if (isCookieAuthMode) {
      return
    }
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  },
  clearTokens() {
    if (isCookieAuthMode) {
      return
    }
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  },
  getCSRFToken(): string | null {
    return readCookie(AUTH_CSRF_COOKIE_NAME)
  },
}
