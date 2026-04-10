import { apiClient } from '@/services/apiClient'
import { pickData, toPaginated } from '@/services/serviceUtils'
import type { PaginatedResponse } from '@/types/api'
import type {
  Conversation,
  ConversationMessage,
  ConversationReadState,
  ConversationRole,
  UnreadCount,
} from '@/types/domain'
import {
  normalizeConversation,
  normalizeConversationMessage,
  normalizeConversationReadState,
  normalizeUnreadCount,
} from '@/utils/normalize'

interface ConversationFilters {
  role?: ConversationRole
  page?: number
  limit?: number
}

interface StartConversationPayload {
  productId: string
  message?: string
}

interface SendMessagePayload {
  body: string
}

export const messengerService = {
  async getConversations(
    filters: ConversationFilters = {},
  ): Promise<PaginatedResponse<Conversation>> {
    const params: Record<string, string | number> = {}

    if (filters.role) {
      params.role = filters.role
    }
    if (filters.page) {
      params.page = filters.page
    }
    if (filters.limit) {
      params.limit = filters.limit
    }

    const response = await apiClient.get('/v1/conversations', { params })
    const paginated = toPaginated<unknown>(response.data)

    return {
      ...paginated,
      items: paginated.items.map(normalizeConversation),
    }
  },

  async startConversation(payload: StartConversationPayload): Promise<Conversation> {
    const response = await apiClient.post('/v1/conversations', {
      product_id: payload.productId,
      message: payload.message,
    })
    return normalizeConversation(pickData(response.data))
  },

  async getConversation(id: string): Promise<Conversation> {
    const response = await apiClient.get(`/v1/conversations/${id}`)
    return normalizeConversation(pickData(response.data))
  },

  async getMessages(
    conversationId: string,
    page = 1,
    limit = 100,
  ): Promise<PaginatedResponse<ConversationMessage>> {
    const response = await apiClient.get(`/v1/conversations/${conversationId}/messages`, {
      params: { page, limit },
    })
    const paginated = toPaginated<unknown>(response.data)
    return {
      ...paginated,
      items: paginated.items.map(normalizeConversationMessage),
    }
  },

  async sendMessage(
    conversationId: string,
    payload: SendMessagePayload,
  ): Promise<ConversationMessage> {
    const response = await apiClient.post(`/v1/conversations/${conversationId}/messages`, {
      body: payload.body,
    })
    return normalizeConversationMessage(pickData(response.data))
  },

  async markAsRead(conversationId: string): Promise<ConversationReadState> {
    const response = await apiClient.post(`/v1/conversations/${conversationId}/read`)
    return normalizeConversationReadState(pickData(response.data))
  },

  async getUnreadCount(role?: ConversationRole): Promise<UnreadCount> {
    const response = await apiClient.get('/v1/conversations/unread-count', {
      params: role ? { role } : undefined,
    })
    return normalizeUnreadCount(pickData(response.data))
  },
}
