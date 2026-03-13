import { apiClient } from '@/services/apiClient'
import { toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse } from '@/types/api'
import type { Product } from '@/types/domain'
import { normalizeProduct } from '@/utils/normalize'

export const favoritesService = {
  async list(page = 1, limit = 20): Promise<PaginatedResponse<Product>> {
    const response = await apiClient.get('/v1/favorites', { params: { page, limit } })
    const paginated = toPaginated<unknown>(response.data)
    return {
      ...paginated,
      items: paginated.items.map(normalizeProduct),
    }
  },

  async add(productId: string): Promise<void> {
    await apiClient.post(`/v1/favorites/${productId}`)
  },

  async remove(productId: string): Promise<void> {
    await apiClient.delete(`/v1/favorites/${productId}`)
  },
}

