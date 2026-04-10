import { useState } from 'react'

import styles from '@/components/messages/MessagesInbox.module.scss'

interface MessageComposerProps {
  disabled?: boolean
  sending?: boolean
  onSend: (body: string) => Promise<boolean>
  placeholder?: string
}

const MESSAGE_LIMIT = 2000

export const MessageComposer = ({
  disabled = false,
  sending = false,
  onSend,
  placeholder = 'Напишите сообщение...',
}: MessageComposerProps) => {
  const [body, setBody] = useState('')
  const [error, setError] = useState<string | null>(null)

  const trimmed = body.trim()
  const isOverLimit = body.length > MESSAGE_LIMIT
  const isBlocked = disabled || sending || trimmed.length === 0 || isOverLimit

  const handleSubmit = async () => {
    if (isBlocked) {
      if (trimmed.length === 0) {
        setError('Сообщение не может быть пустым.')
      } else if (isOverLimit) {
        setError('Сообщение слишком длинное.')
      }
      return
    }

    setError(null)
    const sent = await onSend(trimmed)
    if (sent) {
      setBody('')
    }
  }

  return (
    <div className={styles.composer}>
      <textarea
        value={body}
        maxLength={MESSAGE_LIMIT}
        disabled={disabled || sending}
        placeholder={placeholder}
        onChange={(event) => {
          setBody(event.target.value)
          if (error) {
            setError(null)
          }
        }}
        onKeyDown={(event) => {
          if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
            event.preventDefault()
            void handleSubmit()
          }
        }}
      />

      {error ? <span className={styles.composerError}>{error}</span> : null}

      <div className={styles.composerFooter}>
        <div className={styles.composerHint}>
          До 2000 символов. Пустые сообщения не отправляются. `Ctrl/Cmd + Enter` отправляет
          сообщение.
        </div>
        <div className={styles.composerCount}>
          {body.length} / {MESSAGE_LIMIT}
        </div>
        <button
          type="button"
          className="action-primary"
          disabled={isBlocked}
          onClick={() => void handleSubmit()}
        >
          {sending ? 'Отправляем...' : 'Отправить'}
        </button>
      </div>
    </div>
  )
}
