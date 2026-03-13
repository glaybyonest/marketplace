export interface ApiError {
  message: string
  statusCode?: number
  details?: unknown
  code?: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export interface AuthResponse<TUser> {
  token: string | null
  refreshToken: string | null
  tokenType: string | null
  expiresIn: number
  user: TUser
  requiresEmailVerification: boolean
  message?: string
}

export type ProductSort = 'new' | 'price_asc' | 'price_desc'

export interface ProductFilters {
  q?: string
  category_id?: string
  min_price?: number
  max_price?: number
  in_stock?: boolean
  is_active?: boolean
  sort?: ProductSort
  page?: number
  limit?: number
  // legacy aliases for existing components/utilities
  query?: string
  category?: string
  minPrice?: number
  maxPrice?: number
  inStock?: boolean
  isActive?: boolean
  pageSize?: number
}
