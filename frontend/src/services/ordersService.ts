import { apiClient } from '@/services/apiClient'
import { pickData, toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse } from '@/types/api'
import type { Order } from '@/types/domain'
import { normalizeOrder } from '@/utils/normalize'

export const ordersService = {
  async checkout(placeId: string): Promise<Order> {
    const response = await apiClient.post('/v1/orders', { place_id: placeId })
    return normalizeOrder(pickData(response.data))
  },

  async list(page = 1, limit = 20): Promise<PaginatedResponse<Order>> {
    const response = await apiClient.get('/v1/orders', { params: { page, limit } })
    const paginated = toPaginated<unknown>(response.data)
    return {
      ...paginated,
      items: paginated.items.map(normalizeOrder),
    }
  },

  async getById(id: string): Promise<Order> {
    const response = await apiClient.get(`/v1/orders/${id}`)
    return normalizeOrder(pickData(response.data))
  },
}
