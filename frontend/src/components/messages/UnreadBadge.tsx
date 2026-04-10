import styles from '@/components/messages/MessagesInbox.module.scss'

interface UnreadBadgeProps {
  count: number
}

export const UnreadBadge = ({ count }: UnreadBadgeProps) => {
  if (count <= 0) {
    return null
  }

  return <span className={styles.unreadBadge}>{count > 99 ? '99+' : count}</span>
}
