import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { Cart } from '@/types/domain'
import { normalizeCart } from '@/utils/normalize'

export const cartService = {
  async getCart(): Promise<Cart> {
    const response = await apiClient.get('/v1/cart')
    return normalizeCart(pickData(response.data))
  },

  async addItem(productId: string, quantity: number): Promise<Cart> {
    const response = await apiClient.post('/v1/cart/items', {
      product_id: productId,
      quantity,
    })
    return normalizeCart(pickData(response.data))
  },

  async updateItem(productId: string, quantity: number): Promise<Cart> {
    const response = await apiClient.patch(`/v1/cart/items/${productId}`, { quantity })
    return normalizeCart(pickData(response.data))
  },

  async removeItem(productId: string): Promise<Cart> {
    const response = await apiClient.delete(`/v1/cart/items/${productId}`)
    return normalizeCart(pickData(response.data))
  },

  async clear(): Promise<Cart> {
    const response = await apiClient.delete('/v1/cart')
    return normalizeCart(pickData(response.data))
  },
}
