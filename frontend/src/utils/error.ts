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
    return 'Пользователь с таким email уже существует'
  }

  if (backendCode === 'invalid_input') {
    return 'Проверьте корректность введенных данных'
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
