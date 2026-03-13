import { apiClient } from '@/services/apiClient'
import { pickData, toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse, ProductFilters } from '@/types/api'
import type { Product } from '@/types/domain'
import { normalizeProduct } from '@/utils/normalize'

interface ProductPayload {
  title: string
  description: string
  price: number
  categoryId: string
  stock?: number
  images?: string[]
}

const toProductList = (raw: unknown): PaginatedResponse<Product> => {
  const paginated = toPaginated<unknown>(raw)
  return {
    ...paginated,
    items: paginated.items.map(normalizeProduct),
  }
}

const mapFilters = (filters: ProductFilters = {}) => {
  const params: Record<string, string | number> = {}

  const q = filters.q ?? filters.query
  const categoryId = filters.category_id ?? filters.category
  const limit = filters.limit ?? filters.pageSize

  if (q) {
    params.q = q
  }
  if (categoryId) {
    params.category_id = categoryId
  }
  if (filters.sort) {
    params.sort = filters.sort
  }
  if (filters.page) {
    params.page = filters.page
  }
  if (limit) {
    params.limit = limit
  }

  return params
}

export const productService = {
  async getProducts(filters: ProductFilters = {}): Promise<PaginatedResponse<Product>> {
    const response = await apiClient.get('/v1/products', { params: mapFilters(filters) })
    return toProductList(response.data)
  },

  async searchProducts(query: string): Promise<PaginatedResponse<Product>> {
    const response = await apiClient.get('/v1/products', {
      params: mapFilters({ q: query, page: 1, limit: 20 }),
    })
    return toProductList(response.data)
  },

  async getProductById(id: string): Promise<Product> {
    const response = await apiClient.get(`/v1/products/${id}`)
    return normalizeProduct(pickData(response.data))
  },

  async createProduct(_payload: ProductPayload): Promise<Product> {
    void _payload
    throw new Error('Product creation is not supported by current backend API')
  },

  async updateProduct(_id: string, _payload: Partial<ProductPayload>): Promise<Product> {
    void _id
    void _payload
    throw new Error('Product update is not supported by current backend API')
  },

  async deleteProduct(_id: string): Promise<void> {
    void _id
    throw new Error('Product deletion is not supported by current backend API')
  },
}

