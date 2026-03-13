import type { AxiosError } from 'axios'

import type { ApiError } from '@/types/api'

interface BackendErrorBody {
  error?: {
    code?: string
    message?: string
    details?: unknown
  }
  message?: string
  error_message?: string
  errorCode?: string
}

export const getErrorMessage = (error: unknown, fallback = 'Request failed') => {
  if (typeof error === 'string') {
    return error
  }

  const axiosError = error as AxiosError<BackendErrorBody>
  const backendCode = axiosError.response?.data?.error?.code
  const backendMessage = axiosError.response?.data?.error?.message

  if (backendCode === 'conflict') {
    return 'This operation conflicts with existing data'
  }

  if (backendCode === 'invalid_input') {
    return 'Check the entered data and try again'
  }

  if (backendCode === 'unauthorized') {
    return 'Invalid email or password'
  }

  if (backendCode === 'email_not_verified') {
    return 'Verify your email before signing in'
  }

  if (backendCode === 'rate_limited') {
    return 'Too many requests. Wait a bit and try again'
  }

  if (backendCode === 'login_locked') {
    return 'Too many failed sign-in attempts. Try again later'
  }

  if (backendCode === 'csrf_invalid') {
    return 'Security token is missing or expired. Refresh the page and try again'
  }

  if (backendCode === 'invalid_token') {
    return 'This link is invalid or expired'
  }

  if (backendCode === 'inactive_user') {
    return 'This account is inactive'
  }

  if (backendCode === 'forbidden') {
    return 'You do not have access to this section'
  }

  if (backendCode === 'not_found') {
    return 'Resource not found'
  }

  if (backendCode === 'session_closed') {
    return 'Your session has expired. Sign in again'
  }

  if (backendCode === 'cart_empty') {
    return 'Cart is empty'
  }

  if (backendCode === 'insufficient_stock') {
    return 'Not enough stock for this product'
  }

  if (backendCode === 'product_unavailable') {
    return 'Product is unavailable'
  }

  if (backendMessage) {
    return backendMessage
  }

  const flatMessage = axiosError.response?.data?.message ?? axiosError.response?.data?.error_message
  if (flatMessage) {
    return flatMessage
  }

  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}

export const toApiError = (error: unknown): ApiError => {
  const axiosError = error as AxiosError<BackendErrorBody>
  return {
    message: getErrorMessage(error),
    statusCode: axiosError.response?.status,
    details: axiosError.response?.data?.error?.details ?? axiosError.response?.data ?? null,
    code: axiosError.response?.data?.error?.code,
  }
}
