import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { sessionService } from '@/services/sessionService'
import { useAppDispatch } from '@/store/hooks'
import { forceLogout } from '@/store/slices/authSlice'
import { clearFavorites } from '@/store/slices/favoritesSlice'
import type { SessionInfo } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'

import styles from '@/pages/SessionsPage.module.scss'

const formatDateTime = (value?: string) => {
  if (!value) {
    return '-'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed)
}

export const SessionsPage = () => {
  const navigate = useNavigate()
  const dispatch = useAppDispatch()

  const [sessions, setSessions] = useState<SessionInfo[]>([])
  const [status, setStatus] = useState<'idle' | 'loading' | 'succeeded' | 'failed'>('loading')
  const [error, setError] = useState<string | null>(null)
  const [busySessionID, setBusySessionID] = useState<string | null>(null)
  const [revokeAllBusy, setRevokeAllBusy] = useState(false)

  const loadSessions = useCallback(async () => {
    setStatus('loading')
    setError(null)
    try {
      const items = await sessionService.list()
      setSessions(items)
      setStatus('succeeded')
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to load sessions'))
      setStatus('failed')
    }
  }, [])

  useEffect(() => {
    void loadSessions()
  }, [loadSessions])

  const currentSession = useMemo(() => sessions.find((session) => session.isCurrent), [sessions])

  const handleRevoke = async (sessionID: string) => {
    setBusySessionID(sessionID)
    setError(null)
    try {
      await sessionService.revoke(sessionID)
      setSessions((current) => current.filter((session) => session.id !== sessionID))
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to revoke session'))
    } finally {
      setBusySessionID(null)
    }
  }

  const handleRevokeAll = async () => {
    setRevokeAllBusy(true)
    setError(null)
    try {
      await sessionService.revokeAll()
      dispatch(forceLogout())
      dispatch(clearFavorites())
      navigate('/login', {
        replace: true,
        state: { message: 'All sessions were revoked. Sign in again.' },
      })
    } catch (err) {
      setError(getErrorMessage(err, 'Failed to revoke all sessions'))
      setRevokeAllBusy(false)
    }
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1>Sessions</h1>
          <p>Review active devices and revoke refresh sessions you no longer trust.</p>
        </div>
        <div className={styles.actions}>
          <Link to="/account">Back to account</Link>
          <button type="button" onClick={handleRevokeAll} disabled={revokeAllBusy}>
            {revokeAllBusy ? 'Revoking...' : 'Sign out everywhere'}
          </button>
        </div>
      </header>

      {status === 'loading' ? <AppLoader label="Loading sessions..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {currentSession ? (
        <section className={styles.currentCard}>
          <strong>Current session</strong>
          <span>{currentSession.userAgent || 'Unknown device'}</span>
          <span>Last activity: {formatDateTime(currentSession.lastSeenAt)}</span>
        </section>
      ) : null}

      <section className={styles.list}>
        {sessions.map((session) => (
          <article key={session.id} className={styles.item}>
            <div className={styles.meta}>
              <div className={styles.titleRow}>
                <h2>{session.userAgent || 'Unknown device'}</h2>
                {session.isCurrent ? <span className={styles.badge}>Current</span> : null}
              </div>
              <p>IP: {session.ip || '-'}</p>
              <p>Started: {formatDateTime(session.createdAt)}</p>
              <p>Last seen: {formatDateTime(session.lastSeenAt)}</p>
              <p>Expires: {formatDateTime(session.expiresAt)}</p>
            </div>
            {!session.isCurrent ? (
              <button
                type="button"
                onClick={() => handleRevoke(session.id)}
                disabled={busySessionID === session.id}
              >
                {busySessionID === session.id ? 'Revoking...' : 'Revoke session'}
              </button>
            ) : null}
          </article>
        ))}

        {status === 'succeeded' && sessions.length === 0 ? (
          <div className={styles.empty}>No active sessions found.</div>
        ) : null}
      </section>
    </div>
  )
}
