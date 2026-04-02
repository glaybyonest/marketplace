import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { SellerNav } from '@/components/seller/SellerNav'
import { sellerService } from '@/services/sellerService'
import type { SellerProfile, SellerStatus } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { isGeneratedMediaSource } from '@/utils/media'

import styles from '@/pages/SellerPage.module.scss'

type AsyncState = 'idle' | 'loading' | 'succeeded' | 'failed'

const createFormState = (profile?: SellerProfile) => ({
  storeName: profile?.storeName ?? '',
  storeSlug: profile?.storeSlug ?? '',
  legalName: profile?.legalName ?? '',
  description: profile?.description ?? '',
  logoUrl: isGeneratedMediaSource(profile?.logoUrl) ? '' : (profile?.logoUrl ?? ''),
  bannerUrl: isGeneratedMediaSource(profile?.bannerUrl) ? '' : (profile?.bannerUrl ?? ''),
  supportEmail: profile?.supportEmail ?? '',
  supportPhone: profile?.supportPhone ?? '',
  city: profile?.city ?? '',
  status: (profile?.status ?? 'active') as SellerStatus,
})

const statusLabels: Record<SellerStatus, string> = {
  pending: 'На проверке',
  active: 'Активен',
  paused: 'На паузе',
}

export const SellerStorefrontPage = () => {
  const [profile, setProfile] = useState<SellerProfile | null>(null)
  const [status, setStatus] = useState<AsyncState>('loading')
  const [submitStatus, setSubmitStatus] = useState<AsyncState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [formState, setFormState] = useState(createFormState())

  useEffect(() => {
    let cancelled = false

    const loadProfile = async () => {
      setStatus('loading')
      setError(null)
      try {
        const data = await sellerService.getProfile()
        if (cancelled) {
          return
        }
        setProfile(data)
        setFormState(createFormState(data))
        setStatus('succeeded')
      } catch (loadError) {
        if (cancelled) {
          return
        }
        setStatus('failed')
        setError(getErrorMessage(loadError, 'Не удалось загрузить магазин продавца'))
      }
    }

    void loadProfile()

    return () => {
      cancelled = true
    }
  }, [])

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitStatus('loading')
    setError(null)
    setNotice(null)

    try {
      const nextProfile = await sellerService.upsertProfile({
        storeName: formState.storeName,
        storeSlug: formState.storeSlug || undefined,
        legalName: formState.legalName || undefined,
        description: formState.description || undefined,
        logoUrl: formState.logoUrl || undefined,
        bannerUrl: formState.bannerUrl || undefined,
        supportEmail: formState.supportEmail || undefined,
        supportPhone: formState.supportPhone || undefined,
        city: formState.city || undefined,
        status: formState.status,
      })

      setProfile(nextProfile)
      setFormState(createFormState(nextProfile))
      setSubmitStatus('succeeded')
      setNotice('Профиль магазина сохранён.')
    } catch (submitError) {
      setSubmitStatus('failed')
      setError(getErrorMessage(submitError, 'Не удалось сохранить магазин'))
    }
  }

  if (status === 'loading') {
    return <AppLoader label="Загружаем профиль магазина..." />
  }

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroTop}>
          <div className={styles.heroCopy}>
            <span className={styles.heroBadge}>Витрина магазина</span>
            <h1>{profile?.storeName || 'Магазин продавца'}</h1>
            <p>Поддерживайте витрину в актуальном состоянии: контакты, описание, визуал и статус магазина.</p>
          </div>
          <SellerNav />
        </div>

        <div className={styles.heroMeta}>
          <div className={styles.heroMetaCard}>
            <span>Slug магазина</span>
            <strong>{profile?.storeSlug || 'Не задан'}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Статус</span>
            <strong>{statusLabels[profile?.status ?? 'active']}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Контакт поддержки</span>
            <strong>{profile?.supportEmail || profile?.supportPhone || 'Не указан'}</strong>
          </div>
        </div>
      </section>

      {notice ? <div className="page-card">{notice}</div> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.panel}>
        <div className={styles.panelHeader}>
          <div>
            <span className="badge-pill">Редактирование</span>
            <h2>Параметры магазина</h2>
            <p>Эти данные формируют профиль продавца и используются во всех разделах кабинета магазина.</p>
          </div>
        </div>

        <form className={styles.form} onSubmit={handleSubmit}>
          <div className={styles.formGrid}>
            <label className={styles.field}>
              Название магазина
              <input
                value={formState.storeName}
                onChange={(event) => setFormState((current) => ({ ...current, storeName: event.target.value }))}
                required
              />
            </label>
            <label className={styles.field}>
              Slug
              <input
                value={formState.storeSlug}
                onChange={(event) => setFormState((current) => ({ ...current, storeSlug: event.target.value }))}
              />
            </label>
            <label className={styles.field}>
              Юридическое название
              <input
                value={formState.legalName}
                onChange={(event) => setFormState((current) => ({ ...current, legalName: event.target.value }))}
              />
            </label>
            <label className={styles.field}>
              Город
              <input
                value={formState.city}
                onChange={(event) => setFormState((current) => ({ ...current, city: event.target.value }))}
              />
            </label>
            <label className={styles.field}>
              Email поддержки
              <input
                type="email"
                value={formState.supportEmail}
                onChange={(event) => setFormState((current) => ({ ...current, supportEmail: event.target.value }))}
              />
            </label>
            <label className={styles.field}>
              Телефон поддержки
              <input
                value={formState.supportPhone}
                onChange={(event) => setFormState((current) => ({ ...current, supportPhone: event.target.value }))}
              />
            </label>
            <label className={styles.field}>
              Логотип
              <input
                value={formState.logoUrl}
                onChange={(event) => setFormState((current) => ({ ...current, logoUrl: event.target.value }))}
                placeholder="Оставьте пустым для автоматического логотипа"
              />
            </label>
            <label className={styles.field}>
              Баннер
              <input
                value={formState.bannerUrl}
                onChange={(event) => setFormState((current) => ({ ...current, bannerUrl: event.target.value }))}
                placeholder="Оставьте пустым для локального баннера магазина"
              />
            </label>
            <label className={styles.field}>
              Статус магазина
              <select
                value={formState.status}
                onChange={(event) => setFormState((current) => ({ ...current, status: event.target.value as SellerStatus }))}
              >
                <option value="active">Активен</option>
                <option value="paused">На паузе</option>
                <option value="pending">На проверке</option>
              </select>
            </label>
            <label className={`${styles.field} ${styles.fullWidth}`}>
              Описание
              <textarea
                value={formState.description}
                onChange={(event) => setFormState((current) => ({ ...current, description: event.target.value }))}
              />
            </label>
          </div>

          <div className={styles.inlineActions}>
            <button type="submit" className="action-primary" disabled={submitStatus === 'loading'}>
              {submitStatus === 'loading' ? 'Сохраняем...' : 'Сохранить магазин'}
            </button>
          </div>
        </form>
      </section>
    </div>
  )
}
