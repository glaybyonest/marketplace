export const AUTH_MODE = (import.meta.env.VITE_AUTH_MODE ?? 'token').toLowerCase() === 'cookie' ? 'cookie' : 'token'

export const isCookieAuthMode = AUTH_MODE === 'cookie'

export const AUTH_CSRF_COOKIE_NAME = 'mp_csrf_token'
export const AUTH_CSRF_HEADER_NAME = 'X-CSRF-Token'
