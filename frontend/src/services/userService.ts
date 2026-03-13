import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { User } from '@/types/domain'
import { normalizeUser } from '@/utils/normalize'

interface UpdateProfilePayload {
  fullName?: string
  name?: string
}

export const userService = {
  async getMe(): Promise<User> {
    const response = await apiClient.get('/v1/profile')
    return normalizeUser(pickData(response.data))
  },

  async updateMe(payload: UpdateProfilePayload): Promise<User> {
    const fullName = payload.fullName ?? payload.name
    const response = await apiClient.patch('/v1/profile', {
      full_name: fullName,
    })
    return normalizeUser(pickData(response.data))
  },
}

