import { startTransition, useEffect, useEffectEvent, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'

import { ErrorMessage } from '@/components/common/ErrorMessage'
import { ConversationList } from '@/components/messages/ConversationList'
import { ConversationView } from '@/components/messages/ConversationView'
import { messengerService } from '@/services/messengerService'
import { useAppSelector } from '@/store/hooks'
import type { Conversation, ConversationMessage, ConversationRole } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'

import styles from '@/components/messages/MessagesInbox.module.scss'

interface MessagesInboxProps {
  role: Extract<ConversationRole, 'buyer' | 'seller'>
  nav?: ReactNode
}

type AsyncState = 'idle' | 'loading' | 'succeeded' | 'failed'

const mergeConversations = (items: Conversation[], extra: Conversation | null) => {
  const index = new Map<string, Conversation>()

  for (const item of items) {
    index.set(item.id, item)
  }

  if (extra) {
    index.set(extra.id, extra)
  }

  return Array.from(index.values()).sort(
    (left, right) =>
      new Date(right.lastMessageAt).getTime() - new Date(left.lastMessageAt).getTime(),
  )
}

const areConversationsEqual = (left: Conversation[], right: Conversation[]) =>
  left.length === right.length &&
  left.every((item, index) => {
    const other = right[index]
    return (
      other &&
      item.id === other.id &&
      item.lastMessageAt === other.lastMessageAt &&
      item.lastMessagePreview === other.lastMessagePreview &&
      item.unreadCount === other.unreadCount &&
      item.currentUserRole === other.currentUserRole
    )
  })

const areMessagesEqual = (left: ConversationMessage[], right: ConversationMessage[]) =>
  left.length === right.length &&
  left.every((item, index) => {
    const other = right[index]
    return (
      other &&
      item.id === other.id &&
      item.body === other.body &&
      item.createdAt === other.createdAt &&
      item.editedAt === other.editedAt
    )
  })

export const MessagesInbox = ({ role, nav }: MessagesInboxProps) => {
  const [searchParams, setSearchParams] = useSearchParams()
  const currentUser = useAppSelector((state) => state.auth.user)

  const [conversations, setConversations] = useState<Conversation[]>([])
  const [detachedConversation, setDetachedConversation] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<ConversationMessage[]>([])
  const [listStatus, setListStatus] = useState<AsyncState>('loading')
  const [messagesStatus, setMessagesStatus] = useState<AsyncState>('idle')
  const [listError, setListError] = useState<string | null>(null)
  const [messagesError, setMessagesError] = useState<string | null>(null)
  const [sending, setSending] = useState(false)

  const requestedConversationId = searchParams.get('conversation') ?? ''

  const activeConversation = useMemo(
    () => conversations.find((item) => item.id === requestedConversationId) ?? detachedConversation,
    [conversations, detachedConversation, requestedConversationId],
  )

  const totalUnread = useMemo(
    () =>
      conversations.reduce((sum, item) => sum + item.unreadCount, 0) +
      (detachedConversation && !conversations.some((item) => item.id === detachedConversation.id)
        ? detachedConversation.unreadCount
        : 0),
    [conversations, detachedConversation],
  )

  const hasRequestedConversationInList = useMemo(
    () => conversations.some((item) => item.id === requestedConversationId),
    [conversations, requestedConversationId],
  )

  const roleTitle = role === 'seller' ? 'Сообщения покупателей' : 'Сообщения'
  const roleDescription =
    role === 'seller'
      ? 'Диалоги по вашим товарам. Открывайте переписку и отвечайте без отдельного экрана товара.'
      : 'Ваши переписки с продавцами по конкретным товарам. Откройте диалог и продолжайте разговор с того же места.'
  const inboxTitle = role === 'buyer' ? 'Локальный чат' : roleTitle
  const inboxDescription =
    role === 'buyer'
      ? 'Ваши диалоги с продавцами в одном локальном пространстве. Откройте нужный чат и продолжайте обсуждение товара без поиска по страницам.'
      : roleDescription

  const openConversation = (conversationId: string, replace = false) => {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.set('conversation', conversationId)
    startTransition(() => {
      setSearchParams(nextParams, { replace })
    })
  }

  const clearConversationSelection = () => {
    const nextParams = new URLSearchParams(searchParams)
    nextParams.delete('conversation')
    startTransition(() => {
      setSearchParams(nextParams, { replace: true })
    })
  }

  const markConversationReadLocally = (conversationId: string) => {
    setConversations((current) =>
      current.map((item) => (item.id === conversationId ? { ...item, unreadCount: 0 } : item)),
    )
    setDetachedConversation((current) =>
      current?.id === conversationId ? { ...current, unreadCount: 0 } : current,
    )
  }

  const updateConversationAfterOutgoingMessage = (
    conversationId: string,
    body: string,
    createdAt: string,
  ) => {
    setConversations((current) =>
      mergeConversations(
        current.map((item) =>
          item.id === conversationId
            ? {
                ...item,
                lastMessageAt: createdAt,
                lastMessagePreview: body,
                unreadCount: 0,
              }
            : item,
        ),
        null,
      ),
    )

    setDetachedConversation((current) =>
      current?.id === conversationId
        ? {
            ...current,
            lastMessageAt: createdAt,
            lastMessagePreview: body,
            unreadCount: 0,
          }
        : current,
    )
  }

  const syncConversations = useEffectEvent(async (silent: boolean) => {
    if (!silent) {
      setListStatus('loading')
    }
    setListError(null)

    try {
      const response = await messengerService.getConversations({
        role,
        page: 1,
        limit: 50,
      })

      const nextItems = mergeConversations(response.items, detachedConversation)
      setConversations((current) =>
        areConversationsEqual(current, nextItems) ? current : nextItems,
      )
      setListStatus('succeeded')

      if (requestedConversationId) {
        if (response.items.some((item) => item.id === requestedConversationId)) {
          setDetachedConversation(null)
        }
        return
      }

      if (nextItems[0]) {
        openConversation(nextItems[0].id, true)
        return
      }

      clearConversationSelection()
      setMessages([])
      setMessagesStatus('idle')
    } catch (error) {
      setListStatus('failed')
      setListError(getErrorMessage(error, 'Не удалось загрузить список диалогов.'))
    }
  })

  const syncDetachedConversation = useEffectEvent(async (conversationId: string) => {
    try {
      const conversation = await messengerService.getConversation(conversationId)
      if (conversation.currentUserRole && conversation.currentUserRole !== role) {
        return
      }

      setDetachedConversation((current) =>
        current?.id === conversation.id &&
        current.lastMessageAt === conversation.lastMessageAt &&
        current.lastMessagePreview === conversation.lastMessagePreview &&
        current.unreadCount === conversation.unreadCount
          ? current
          : conversation,
      )
      setConversations((current) => {
        const nextItems = mergeConversations(current, conversation)
        return areConversationsEqual(current, nextItems) ? current : nextItems
      })
    } catch {
      setDetachedConversation(null)
    }
  })

  const syncMessages = useEffectEvent(async (conversationId: string, silent: boolean) => {
    if (!conversationId) {
      setMessages([])
      setMessagesStatus('idle')
      setMessagesError(null)
      return
    }

    if (!silent) {
      setMessagesStatus('loading')
    }
    setMessagesError(null)

    try {
      const response = await messengerService.getMessages(conversationId, 1, 100)
      setMessages((current) =>
        areMessagesEqual(current, response.items) ? current : response.items,
      )
      setMessagesStatus('succeeded')

      await messengerService.markAsRead(conversationId)
      markConversationReadLocally(conversationId)
    } catch (error) {
      setMessagesStatus('failed')
      setMessagesError(getErrorMessage(error, 'Не удалось загрузить сообщения.'))
    }
  })

  useEffect(() => {
    void syncConversations(false)

    const intervalId = window.setInterval(() => {
      if (document.visibilityState !== 'visible') {
        return
      }
      void syncConversations(true)
    }, 12000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [role])

  useEffect(() => {
    if (!requestedConversationId) {
      setDetachedConversation(null)
      setMessages([])
      setMessagesStatus('idle')
      return
    }

    if (!hasRequestedConversationInList) {
      void syncDetachedConversation(requestedConversationId)
    }

    void syncMessages(requestedConversationId, false)

    const intervalId = window.setInterval(() => {
      if (document.visibilityState !== 'visible') {
        return
      }
      void syncMessages(requestedConversationId, true)
    }, 4000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [hasRequestedConversationInList, requestedConversationId, role])

  const handleSendMessage = async (body: string) => {
    if (!activeConversation) {
      return false
    }

    setSending(true)

    try {
      const message = await messengerService.sendMessage(activeConversation.id, { body })
      setMessages((current) => [...current, message])
      updateConversationAfterOutgoingMessage(activeConversation.id, message.body, message.createdAt)
      setMessagesStatus('succeeded')
      setMessagesError(null)
      return true
    } catch (error) {
      setMessagesError(getErrorMessage(error, 'Не удалось отправить сообщение.'))
      return false
    } finally {
      setSending(false)
    }
  }

  if (!currentUser?.id) {
    return <ErrorMessage message="Нужно авторизоваться, чтобы открыть сообщения." />
  }

  return (
    <div className={styles.page}>
      {nav ? <div className={styles.navRow}>{nav}</div> : null}
      {listError ? <ErrorMessage message={listError} /> : null}

      <section className={styles.layout}>
        <div className={styles.sidebar}>
          <div className={styles.sidebarHeader}>
            <div>
              <h1>{inboxTitle}</h1>
              <p>{inboxDescription}</p>
            </div>

            <div className={styles.sidebarStats}>
              <div className={styles.sidebarStat}>
                <span>Диалогов</span>
                <strong>{conversations.length}</strong>
              </div>
              <div className={styles.sidebarStat}>
                <span>Непрочитанные</span>
                <strong>{totalUnread}</strong>
              </div>
            </div>
          </div>

          {listStatus === 'failed' ? (
            <div className={styles.listState}>
              <h3>Список временно недоступен</h3>
              <p>Попробуйте обновить страницу чуть позже.</p>
            </div>
          ) : (
            <ConversationList
              conversations={conversations}
              activeConversationId={activeConversation?.id}
              currentUserId={currentUser.id}
              onSelect={openConversation}
            />
          )}
        </div>

        <div className={styles.thread}>
          <ConversationView
            conversation={activeConversation ?? null}
            currentUserId={currentUser.id}
            messages={messages}
            status={messagesStatus}
            error={messagesError}
            sending={sending}
            onSend={handleSendMessage}
          />
        </div>
      </section>
    </div>
  )
}
