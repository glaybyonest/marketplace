import { apiClient } from '@/services/apiClient'
import { pickData, toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse, ProductFilters } from '@/types/api'
import type { PopularSearch, Product, SearchSuggestion } from '@/types/domain'
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
  const params: Record<string, string | number | boolean> = {}

  const q = filters.q ?? filters.query
  const categoryId = filters.category_id ?? filters.category
  const minPrice = filters.min_price ?? filters.minPrice
  const maxPrice = filters.max_price ?? filters.maxPrice
  const inStock = filters.in_stock ?? filters.inStock
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
  if (typeof minPrice === 'number') {
    params.min_price = minPrice
  }
  if (typeof maxPrice === 'number') {
    params.max_price = maxPrice
  }
  if (typeof inStock === 'boolean') {
    params.in_stock = inStock
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

  async getSearchSuggestions(query: string, limit = 8): Promise<SearchSuggestion[]> {
    const response = await apiClient.get('/v1/search/suggestions', {
      params: { q: query, limit },
    })
    const items = pickData<unknown[]>(response.data) ?? []
    return Array.isArray(items)
      ? items.map((item) => {
          const source = (item as Record<string, unknown>) ?? {}
          return {
            text: typeof source.text === 'string' ? source.text : '',
            kind: typeof source.kind === 'string' ? source.kind : 'query',
          }
        }).filter((item) => item.text.length > 0)
      : []
  },

  async getPopularSearches(limit = 6): Promise<PopularSearch[]> {
    const response = await apiClient.get('/v1/search/popular', { params: { limit } })
    const items = pickData<unknown[]>(response.data) ?? []
    return Array.isArray(items)
      ? items.map((item) => {
          const source = (item as Record<string, unknown>) ?? {}
          return {
            query: typeof source.query === 'string' ? source.query : '',
            searchCount: Number(source.search_count ?? source.searchCount ?? 0) || 0,
          }
        }).filter((item) => item.query.length > 0)
      : []
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
