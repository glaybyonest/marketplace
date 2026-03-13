import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { Product } from '@/types/domain'
import { normalizeProduct } from '@/utils/normalize'

export const recommendationsService = {
  async list(limit = 8): Promise<Product[]> {
    const response = await apiClient.get('/v1/recommendations', { params: { limit } })
    const source = pickData<unknown>(response.data)
    if (!Array.isArray(source)) {
      return []
    }
    return source.map(normalizeProduct)
  },
}

