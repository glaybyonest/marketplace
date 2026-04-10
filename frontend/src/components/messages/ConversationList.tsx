import clsx from 'clsx'

import { UnreadBadge } from '@/components/messages/UnreadBadge'
import type { Conversation } from '@/types/domain'
import { formatDate } from '@/utils/format'
import {
  resolveConversationProductImage,
  resolveConversationProductImageFallback,
  swapImageToFallback,
} from '@/utils/media'

import styles from '@/components/messages/MessagesInbox.module.scss'

interface ConversationListProps {
  conversations: Conversation[]
  activeConversationId?: string
  currentUserId: string
  onSelect: (conversationId: string) => void
}

const getCounterpartyLabel = (conversation: Conversation, currentUserId: string) => {
  if (conversation.buyerId === currentUserId) {
    return conversation.sellerStoreName || conversation.sellerName
  }
  return conversation.buyerName
}

const getConversationModeLabel = (conversation: Conversation, currentUserId: string) =>
  conversation.buyerId === currentUserId ? 'Локальный чат' : 'Запрос покупателя'

export const ConversationList = ({
  conversations,
  activeConversationId,
  currentUserId,
  onSelect,
}: ConversationListProps) => {
  if (conversations.length === 0) {
    return (
      <div className={styles.listState}>
        <h3>Диалогов пока нет</h3>
        <p>
          Как только появится первая переписка по товару, она отобразится в этом списке.
        </p>
      </div>
    )
  }

  return (
    <div className={styles.conversationList}>
      {conversations.map((conversation) => (
        <button
          key={conversation.id}
          type="button"
          className={clsx(
            styles.conversationButton,
            conversation.id === activeConversationId && styles.conversationActive,
          )}
          onClick={() => onSelect(conversation.id)}
        >
          <div className={styles.conversationVisual}>
            <img
              src={resolveConversationProductImage(conversation)}
              alt={conversation.productName}
              className={styles.conversationImage}
              loading="lazy"
              onError={(event) =>
                swapImageToFallback(
                  event.currentTarget,
                  resolveConversationProductImageFallback(conversation),
                )
              }
            />
            <span className={styles.conversationVisualBadge}>Инфо</span>
          </div>

          <div className={styles.conversationBody}>
            <div className={styles.conversationTop}>
              <div className={styles.conversationProduct}>
                <strong>{conversation.productName}</strong>
                <span className={styles.conversationMeta}>
                  {getCounterpartyLabel(conversation, currentUserId)}
                </span>
              </div>
              <UnreadBadge count={conversation.unreadCount} />
            </div>

            <p className={styles.conversationPreview}>
              {conversation.lastMessagePreview ||
                'Диалог создан. Можно написать первое сообщение.'}
            </p>

            <div className={styles.conversationChips}>
              <span className={styles.conversationChip}>
                {conversation.sellerStoreName || conversation.sellerName || 'Товар'}
              </span>
              <span className={styles.conversationChipMuted}>
                {getConversationModeLabel(conversation, currentUserId)}
              </span>
            </div>

            <div className={styles.conversationFoot}>
              <span className={styles.conversationDate}>
                {formatDate(conversation.lastMessageAt)}
              </span>
              <span className={styles.conversationMeta}>
                {conversation.sellerStoreName
                  ? `Магазин: ${conversation.sellerStoreName}`
                  : 'Переписка по товару'}
              </span>
            </div>
          </div>
        </button>
      ))}
    </div>
  )
}
