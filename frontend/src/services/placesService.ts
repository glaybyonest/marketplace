import { apiClient } from '@/services/apiClient'
import { pickData } from '@/services/serviceUtils'
import type { Place } from '@/types/domain'
import { normalizePlace } from '@/utils/normalize'

interface PlacePayload {
  title: string
  addressText: string
  lat?: number
  lon?: number
}

export const placesService = {
  async list(): Promise<Place[]> {
    const response = await apiClient.get('/v1/places')
    const source = pickData<unknown>(response.data)
    if (!Array.isArray(source)) {
      return []
    }
    return source.map(normalizePlace)
  },

  async create(payload: PlacePayload): Promise<Place> {
    const response = await apiClient.post('/v1/places', {
      title: payload.title,
      address_text: payload.addressText,
      lat: payload.lat,
      lon: payload.lon,
    })
    return normalizePlace(pickData(response.data))
  },

  async update(id: string, payload: Partial<PlacePayload>): Promise<Place> {
    const response = await apiClient.patch(`/v1/places/${id}`, {
      title: payload.title,
      address_text: payload.addressText,
      lat: payload.lat,
      lon: payload.lon,
    })
    return normalizePlace(pickData(response.data))
  },

  async remove(id: string): Promise<void> {
    await apiClient.delete(`/v1/places/${id}`)
  },
}

