import { apiClient } from '@/services/apiClient'
import { pickData, toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse, ProductFilters } from '@/types/api'
import type { PopularSearch, Product, SearchSuggestion } from '@/types/domain'
import { normalizeProduct } from '@/utils/normalize'

interface ProductPayload {
  title?: string
  name?: string
  slug?: string
  description: string
  price: number
  categoryId: string
  currency?: string
  sku?: string
  imageUrl?: string
  stock?: number
  images?: string[]
  brand?: string
  unit?: string
  specs?: Product['specs']
  isActive?: boolean
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
  const isActive = filters.is_active ?? filters.isActive
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
  if (typeof isActive === 'boolean') {
    params.is_active = isActive
  }
  if (filters.page) {
    params.page = filters.page
  }
  if (limit) {
    params.limit = limit
  }

  return params
}

const toProductRequestBody = (payload: ProductPayload) => ({
  category_id: payload.categoryId,
  name: payload.name ?? payload.title ?? 'Product',
  slug: payload.slug,
  description: payload.description,
  price: payload.price,
  currency: payload.currency,
  sku: payload.sku,
  image_url: payload.imageUrl,
  images: payload.images ?? [],
  brand: payload.brand,
  unit: payload.unit,
  specs: payload.specs ?? {},
  stock_qty: payload.stock ?? 0,
  is_active: payload.isActive,
})

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

  async getAdminProducts(filters: ProductFilters = {}): Promise<PaginatedResponse<Product>> {
    const response = await apiClient.get('/v1/admin/products', { params: mapFilters(filters) })
    return toProductList(response.data)
  },

  async createProduct(payload: ProductPayload): Promise<Product> {
    const response = await apiClient.post('/v1/admin/products', toProductRequestBody(payload))
    return normalizeProduct(pickData(response.data))
  },

  async updateProduct(id: string, payload: Partial<ProductPayload>): Promise<Product> {
    const current = payload as ProductPayload
    const response = await apiClient.patch(`/v1/admin/products/${id}`, toProductRequestBody({
      title: current.title,
      name: current.name,
      slug: current.slug,
      description: current.description ?? '',
      price: current.price ?? 0,
      categoryId: current.categoryId ?? '',
      currency: current.currency,
      sku: current.sku,
      imageUrl: current.imageUrl,
      stock: current.stock,
      images: current.images,
      brand: current.brand,
      unit: current.unit,
      specs: current.specs,
      isActive: current.isActive,
    }))
    return normalizeProduct(pickData(response.data))
  },

  async updateProductStock(id: string, stockQty: number): Promise<Product> {
    const response = await apiClient.patch(`/v1/admin/products/${id}/stock`, {
      stock_qty: stockQty,
    })
    return normalizeProduct(pickData(response.data))
  },

  async deleteProduct(id: string): Promise<void> {
    await apiClient.delete(`/v1/admin/products/${id}`)
  },
}
