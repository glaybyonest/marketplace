import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { AccountMessagesPage } from '@/pages/AccountMessagesPage'
import { messengerService } from '@/services/messengerService'
import authReducer from '@/store/slices/authSlice'

vi.mock('@/services/messengerService', () => ({
  messengerService: {
    getConversations: vi.fn(),
    getConversation: vi.fn(),
    getMessages: vi.fn(),
    sendMessage: vi.fn(),
    markAsRead: vi.fn(),
    getUnreadCount: vi.fn(),
  },
}))

const mockedMessengerService = vi.mocked(messengerService)

const createStore = () => {
  const authState: ReturnType<typeof authReducer> = {
    token: 'token',
    refreshToken: 'refresh',
    user: {
      id: 'buyer-1',
      email: 'buyer@test.local',
      fullName: 'Buyer One',
      role: 'customer',
      isEmailVerified: true,
    },
    isAuthenticated: true,
    status: 'idle',
    error: null,
    errorCode: null,
    notice: null,
    requiresEmailVerification: false,
    sessionBootstrapped: true,
  }

  return configureStore({
    reducer: {
      auth: authReducer,
    },
    preloadedState: {
      auth: authState,
    },
  })
}

describe('AccountMessagesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mockedMessengerService.getConversations.mockResolvedValue({
      items: [
        {
          id: 'conversation-1',
          productId: 'product-1',
          productName: 'Smart Lamp',
          productImageUrl: '',
          sellerId: 'seller-1',
          sellerName: 'Seller One',
          sellerStoreName: 'Bright Store',
          buyerId: 'buyer-1',
          buyerName: 'Buyer One',
          lastMessageAt: '2026-04-09T12:00:00.000Z',
          lastMessagePreview: 'Здравствуйте! Товар ещё в наличии?',
          unreadCount: 2,
          currentUserRole: 'buyer',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 50,
      totalPages: 1,
    })

    mockedMessengerService.getMessages.mockResolvedValue({
      items: [
        {
          id: 'message-1',
          conversationId: 'conversation-1',
          senderId: 'buyer-1',
          body: 'Здравствуйте! Товар ещё в наличии?',
          createdAt: '2026-04-09T12:00:00.000Z',
        },
      ],
      total: 1,
      page: 1,
      pageSize: 100,
      totalPages: 1,
    })

    mockedMessengerService.getConversation.mockResolvedValue({
      id: 'conversation-1',
      productId: 'product-1',
      productName: 'Smart Lamp',
      productImageUrl: '',
      sellerId: 'seller-1',
      sellerName: 'Seller One',
      sellerStoreName: 'Bright Store',
      buyerId: 'buyer-1',
      buyerName: 'Buyer One',
      lastMessageAt: '2026-04-09T12:00:00.000Z',
      lastMessagePreview: 'Здравствуйте! Товар ещё в наличии?',
      unreadCount: 2,
      currentUserRole: 'buyer',
    })

    mockedMessengerService.markAsRead.mockResolvedValue({
      conversationId: 'conversation-1',
      userId: 'buyer-1',
      lastReadMessageId: 'message-1',
      lastReadAt: '2026-04-09T12:00:00.000Z',
    })
  })

  it('renders conversations and active messages thread', async () => {
    const store = createStore()

    render(
      <Provider store={store}>
        <MemoryRouter initialEntries={['/account/messages?conversation=conversation-1']}>
          <AccountMessagesPage />
        </MemoryRouter>
      </Provider>,
    )

    expect(screen.getByText('Локальный чат')).toBeInTheDocument()
    expect(screen.getByText('Диалогов')).toBeInTheDocument()
    expect((await screen.findAllByText('Smart Lamp')).length).toBeGreaterThan(0)
    expect((await screen.findAllByText('Bright Store')).length).toBeGreaterThan(0)
    expect((await screen.findAllByText('Здравствуйте! Товар ещё в наличии?')).length).toBeGreaterThan(0)
    expect(mockedMessengerService.getConversations).toHaveBeenCalledWith({
      role: 'buyer',
      page: 1,
      limit: 50,
    })
  })
})
