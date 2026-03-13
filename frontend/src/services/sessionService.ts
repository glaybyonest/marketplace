import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { SessionInfo } from '@/types/domain'
import { normalizeSession } from '@/utils/normalize'

export const sessionService = {
  async list(): Promise<SessionInfo[]> {
    const response = await apiClient.get('/v1/auth/sessions')
    const items = pickData<unknown[]>(response.data)
    return (Array.isArray(items) ? items : []).map(normalizeSession)
  },

  async revoke(sessionID: string): Promise<void> {
    await apiClient.delete(`/v1/auth/sessions/${sessionID}`)
  },

  async revokeAll(): Promise<void> {
    await apiClient.post('/v1/auth/logout-all')
  },
}
