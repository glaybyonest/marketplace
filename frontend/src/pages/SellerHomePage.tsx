import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { SellerNav } from '@/components/seller/SellerNav'
import { sellerService } from '@/services/sellerService'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchProfileThunk } from '@/store/slices/userSlice'
import type { SellerDashboard } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { formatCurrency, formatDate } from '@/utils/format'
import { resolveProductImage, resolveProductImageFallback, resolveSellerBanner, resolveSellerLogo, swapImageToFallback } from '@/utils/media'

import styles from '@/pages/SellerPage.module.scss'

type AsyncState = 'idle' | 'loading' | 'succeeded' | 'failed'

const statusLabels = {
  pending: 'На проверке',
  active: 'Активен',
  paused: 'На паузе',
} as const

const createInitialForm = () => ({
  storeName: '',
  storeSlug: '',
  legalName: '',
  city: '',
  supportEmail: '',
  supportPhone: '',
  description: '',
})

export const SellerHomePage = () => {
  const dispatch = useAppDispatch()
  const authUser = useAppSelector((state) => state.auth.user)

  const [dashboard, setDashboard] = useState<SellerDashboard | null>(null)
  const [status, setStatus] = useState<AsyncState>('idle')
  const [submitStatus, setSubmitStatus] = useState<AsyncState>('idle')
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [formState, setFormState] = useState(createInitialForm())

  const isSeller = authUser?.role === 'seller'

  useEffect(() => {
    if (!isSeller) {
      setDashboard(null)
      setStatus('idle')
      return
    }

    let cancelled = false

    const loadDashboard = async () => {
      setStatus('loading')
      setError(null)

      try {
        const data = await sellerService.getDashboard()
        if (cancelled) {
          return
        }
        setDashboard(data)
        setStatus('succeeded')
      } catch (loadError) {
        if (cancelled) {
          return
        }
        setStatus('failed')
        setError(getErrorMessage(loadError, 'Не удалось загрузить кабинет продавца'))
      }
    }

    void loadDashboard()

    return () => {
      cancelled = true
    }
  }, [isSeller])

  const previewStoreName = formState.storeName.trim() || 'Ваш магазин'
  const previewDescription =
    formState.description.trim() || 'Расскажите, какие товары вы продаёте и почему покупателям удобно возвращаться именно к вам.'

  const previewLogo = resolveSellerLogo({
    storeSlug: formState.storeSlug || previewStoreName,
    storeName: previewStoreName,
    city: formState.city || undefined,
    status: 'active',
  })
  const previewBanner = resolveSellerBanner({
    storeSlug: formState.storeSlug || previewStoreName,
    storeName: previewStoreName,
    description: formState.description || undefined,
    city: formState.city || undefined,
    status: 'active',
  })
  const dashboardLogo = dashboard
    ? resolveSellerLogo({
        storeSlug: dashboard.profile.storeSlug,
        storeName: dashboard.profile.storeName,
        city: dashboard.profile.city,
        status: dashboard.profile.status,
        logoUrl: dashboard.profile.logoUrl,
      })
    : ''
  const dashboardBanner = dashboard
    ? resolveSellerBanner({
        storeSlug: dashboard.profile.storeSlug,
        storeName: dashboard.profile.storeName,
        description: dashboard.profile.description,
        city: dashboard.profile.city,
        status: dashboard.profile.status,
        bannerUrl: dashboard.profile.bannerUrl,
      })
    : ''

  const metricCards = useMemo(() => {
    if (!dashboard) {
      return []
    }

    return [
      { label: 'Товаров в магазине', value: String(dashboard.metrics.productsTotal) },
      { label: 'Активных карточек', value: String(dashboard.metrics.activeProducts) },
      { label: 'Заказов с витрины', value: String(dashboard.metrics.ordersTotal) },
      { label: 'Выручка', value: formatCurrency(dashboard.metrics.grossRevenue, 'RUB') },
    ]
  }, [dashboard])

  const handleBecomeSeller = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitStatus('loading')
    setError(null)
    setNotice(null)

    try {
      await sellerService.upsertProfile({
        storeName: formState.storeName,
        storeSlug: formState.storeSlug || undefined,
        legalName: formState.legalName || undefined,
        city: formState.city || undefined,
        supportEmail: formState.supportEmail || undefined,
        supportPhone: formState.supportPhone || undefined,
        description: formState.description || undefined,
      })

      const result = await dispatch(fetchProfileThunk())
      if (!fetchProfileThunk.fulfilled.match(result)) {
        throw new Error('profile_refresh_failed')
      }

      setNotice('Магазин активирован. Открываем кабинет продавца...')
      setSubmitStatus('succeeded')
    } catch (submitError) {
      setSubmitStatus('failed')
      setError(getErrorMessage(submitError, 'Не удалось открыть кабинет продавца'))
    }
  }

  if (isSeller && status === 'loading') {
    return <AppLoader label="Загружаем кабинет продавца..." />
  }

  if (!isSeller) {
    return (
      <div className={`${styles.page} ${styles.setupPage}`}>
        <section className={styles.hero}>
          <div className={styles.heroTop}>
            <div className={styles.heroCopy}>
              <span className={styles.heroBadge}>Кабинет продавца</span>
              <h1>Откройте магазин и управляйте продажами из одного кабинета</h1>
              <p>
                Продавец получает отдельную рабочую зону: витрину магазина, свои карточки товаров, управление остатками,
                видимостью и сводку по заказам без перехода в админку.
              </p>
              <div className={styles.heroActions}>
                <Link to="/" className="action-secondary">
                  Посмотреть витрину
                </Link>
                <Link to="/account/orders" className="action-ghost">
                  История ваших покупок
                </Link>
              </div>
            </div>
          </div>

          <div className={styles.heroMeta}>
            <div className={styles.heroMetaCard}>
              <span>Свой магазин</span>
              <strong>Название, описание, контакты и статус магазина</strong>
            </div>
            <div className={styles.heroMetaCard}>
              <span>Управление товарами</span>
              <strong>Цена, остатки, видимость и состав карточек</strong>
            </div>
            <div className={styles.heroMetaCard}>
              <span>Сводка по заказам</span>
              <strong>Сколько заказов пришло и какие позиции продаются лучше</strong>
            </div>
          </div>
        </section>

        {notice ? <div className="page-card">{notice}</div> : null}
        {error ? <ErrorMessage message={error} /> : null}

        <div className={styles.contentGrid}>
          <section className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <span className="badge-pill">Старт</span>
                <h2>Создайте магазин продавца</h2>
                <p>После активации у вас появятся метрики магазина, каталог продавца и отдельный раздел заказов.</p>
              </div>
            </div>

            <form className={styles.form} onSubmit={handleBecomeSeller}>
              <div className={styles.formGrid}>
                <label className={styles.field}>
                  Название магазина
                  <input
                    value={formState.storeName}
                    onChange={(event) => setFormState((current) => ({ ...current, storeName: event.target.value }))}
                    placeholder="North House"
                    required
                  />
                </label>
                <label className={styles.field}>
                  Slug магазина
                  <input
                    value={formState.storeSlug}
                    onChange={(event) => setFormState((current) => ({ ...current, storeSlug: event.target.value }))}
                    placeholder="north-house"
                  />
                </label>
                <label className={styles.field}>
                  Юридическое название
                  <input
                    value={formState.legalName}
                    onChange={(event) => setFormState((current) => ({ ...current, legalName: event.target.value }))}
                    placeholder="North House LLC"
                  />
                </label>
                <label className={styles.field}>
                  Город
                  <input
                    value={formState.city}
                    onChange={(event) => setFormState((current) => ({ ...current, city: event.target.value }))}
                    placeholder="Москва"
                  />
                </label>
                <label className={styles.field}>
                  Email поддержки
                  <input
                    type="email"
                    value={formState.supportEmail}
                    onChange={(event) => setFormState((current) => ({ ...current, supportEmail: event.target.value }))}
                    placeholder="seller@example.com"
                  />
                </label>
                <label className={styles.field}>
                  Телефон поддержки
                  <input
                    value={formState.supportPhone}
                    onChange={(event) => setFormState((current) => ({ ...current, supportPhone: event.target.value }))}
                    placeholder="+7 999 123-45-67"
                  />
                </label>
                <label className={`${styles.field} ${styles.fullWidth}`}>
                  Описание магазина
                  <textarea
                    value={formState.description}
                    onChange={(event) => setFormState((current) => ({ ...current, description: event.target.value }))}
                    placeholder="Кратко расскажите о специализации магазина, скорости отгрузки и сильных сторонах ассортимента."
                  />
                </label>
              </div>

              <p className={styles.helper}>
                После сохранения вы сразу попадёте в кабинет продавца и сможете добавлять товары, менять витрину и
                отслеживать свои заказы.
              </p>

              <div className={styles.inlineActions}>
                <button type="submit" className="action-primary" disabled={submitStatus === 'loading'}>
                  {submitStatus === 'loading' ? 'Открываем магазин...' : 'Стать продавцом'}
                </button>
                <Link to="/account" className="action-secondary">
                  Вернуться в аккаунт
                </Link>
              </div>
            </form>
          </section>

          <aside className={styles.panel}>
            <div className={styles.panelHeader}>
              <div>
                <span className="badge-pill">Превью</span>
                <h2>Как будет выглядеть магазин</h2>
                <p>Название, описание и контакты сразу используются в профиле продавца.</p>
              </div>
            </div>

            <div className={styles.previewCard}>
              <div
                className={`${styles.previewBanner} ${styles.previewBannerSetup}`}
                style={{
                  backgroundImage: `linear-gradient(rgb(27 10 13 / 0.56), rgb(27 10 13 / 0.7)), url("${previewBanner}")`,
                  backgroundSize: 'cover',
                  backgroundPosition: 'center',
                }}
              >
                <div className={styles.previewIdentity}>
                  <img src={previewLogo} alt={previewStoreName} className={styles.previewLogo} />
                  <div>
                    <h3>{previewStoreName}</h3>
                    <p>{formState.city.trim() || 'Город продавца'}</p>
                  </div>
                </div>
              </div>

              <div className={styles.previewStats}>
                <p>{previewDescription}</p>
                <div className={styles.previewPills}>
                  <span className={styles.pill}>Контакты продавца</span>
                  <span className={styles.pill}>Собственная витрина</span>
                  <span className={styles.pill}>Управление остатками</span>
                </div>
              </div>
            </div>
          </aside>
        </div>
      </div>
    )
  }

  if (!dashboard) {
    return <ErrorMessage message={error || 'Не удалось загрузить кабинет продавца'} />
  }

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroTop}>
          <div className={styles.heroCopy}>
            <span className={styles.heroBadge}>Кабинет продавца</span>
            <h1>{dashboard.profile.storeName}</h1>
            <p>
              Управляйте ассортиментом, отслеживайте динамику магазина и держите под рукой заказы, которым нужен
              контроль по остаткам и отгрузке.
            </p>
            <div className={styles.heroActions}>
              <Link to="/seller/products" className="action-primary">
                Открыть товары
              </Link>
              <Link to="/seller/storefront" className="action-secondary">
                Настроить магазин
              </Link>
              <Link to="/seller/orders" className="action-ghost">
                Заказы продавца
              </Link>
            </div>
          </div>

          <SellerNav />
        </div>

        <div className={styles.heroMeta}>
          <div className={styles.heroMetaCard}>
            <span>Статус магазина</span>
            <strong>{statusLabels[dashboard.profile.status]}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Город</span>
            <strong>{dashboard.profile.city || 'Не указан'}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Контакт поддержки</span>
            <strong>{dashboard.profile.supportEmail || dashboard.profile.supportPhone || 'Не указан'}</strong>
          </div>
        </div>
      </section>

      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.metricGrid}>
        {metricCards.map((card) => (
          <article key={card.label} className={styles.metricCard}>
            <span>{card.label}</span>
            <strong className={styles.metricValue}>{card.value}</strong>
          </article>
        ))}
      </section>

      <div className={styles.contentGrid}>
        <section className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">Быстрые действия</span>
              <h2>Что важно сегодня</h2>
              <p>Кабинет продавца связан с реальными товарами и реальными заказами магазина.</p>
            </div>
          </div>

          <div className={styles.actionGrid}>
            <Link to="/seller/products" className={styles.actionCard}>
              <strong>Добавить товар</strong>
              <p>Откройте форму, соберите карточку и сразу задайте цену, остаток и видимость.</p>
            </Link>
            <Link to="/seller/orders" className={styles.actionCard}>
              <strong>Проверить заказы</strong>
              <p>Посмотрите последние заказы продавца и сверьте выручку по своей части корзины.</p>
            </Link>
            <Link to="/seller/storefront" className={styles.actionCard}>
              <strong>Обновить витрину</strong>
              <p>Подтяните описание магазина, контакты и статус, чтобы профиль всегда оставался актуальным.</p>
            </Link>
          </div>

          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">Ассортимент</span>
              <h2>Свежие карточки магазина</h2>
            </div>
            <Link to="/seller/products" className="action-secondary">
              Все товары
            </Link>
          </div>

          <div className={styles.list}>
            {dashboard.recentProducts.length === 0 ? (
              <div className="empty-state">
                <h2>Пока нет товаров</h2>
                <p>Добавьте первую карточку, чтобы магазин появился в рабочем ритме.</p>
              </div>
            ) : (
              dashboard.recentProducts.map((product) => (
                <article key={product.id} className={styles.listCard}>
                  <img
                    className={styles.mediaThumb}
                    src={resolveProductImage(product)}
                    alt={product.title}
                    onError={(event) => swapImageToFallback(event.currentTarget, resolveProductImageFallback(product))}
                  />
                  <div className={styles.listHeader}>
                    <div>
                      <h3>{product.title}</h3>
                      <p className={styles.listMeta}>
                        {product.categoryName || 'Каталог'} • {formatCurrency(product.price, product.currency)}
                      </p>
                    </div>
                    <div className={styles.badgeRow}>
                      <span className={product.isActive ? styles.badge : styles.badgeDanger}>
                        {product.isActive ? 'В продаже' : 'Скрыт'}
                      </span>
                      <span className={styles.badgeMuted}>Остаток: {product.stock ?? 0}</span>
                    </div>
                  </div>
                </article>
              ))
            )}
          </div>
        </section>

        <aside className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">Магазин</span>
              <h2>Ваша витрина</h2>
              <p>Превью магазина и текущие сигналы по остаткам.</p>
            </div>
          </div>

          <div className={styles.previewCard}>
            <div
              className={styles.previewBanner}
              style={{
                backgroundImage: `linear-gradient(rgb(15 71 61 / 0.46), rgb(15 71 61 / 0.56)), url("${dashboardBanner}")`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
              }}
            >
              <div className={styles.previewIdentity}>
                <img src={dashboardLogo} alt={dashboard.profile.storeName} className={styles.previewLogo} />
                <div>
                  <h3>{dashboard.profile.storeName}</h3>
                  <p>{dashboard.profile.city || 'Онлайн-магазин'}</p>
                </div>
              </div>
            </div>

            <div className={styles.previewStats}>
              <p>{dashboard.profile.description || 'Добавьте описание магазина, чтобы рассказать о специализации и уровне сервиса.'}</p>
              <div className={styles.previewPills}>
                <span className={styles.pill}>{statusLabels[dashboard.profile.status]}</span>
                <span className={styles.pill}>{dashboard.metrics.activeProducts} активных карточек</span>
                <span className={styles.pill}>{dashboard.metrics.lowStockProducts} позиций требуют пополнения</span>
              </div>
            </div>
          </div>

          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">Остатки</span>
              <h2>Нужно пополнить</h2>
            </div>
          </div>

          <div className={styles.list}>
            {dashboard.lowStock.length === 0 ? (
              <div className="empty-state">
                <h2>Низких остатков нет</h2>
                <p>Самые чувствительные позиции уже под контролем.</p>
              </div>
            ) : (
              dashboard.lowStock.map((product) => (
                <article key={product.id} className={styles.listCard}>
                  <div className={styles.listHeader}>
                    <div>
                      <h3>{product.title}</h3>
                      <p className={styles.listMeta}>{product.categoryName || 'Каталог'}</p>
                    </div>
                    <span className={styles.badgeWarn}>Осталось: {product.stock ?? 0}</span>
                  </div>
                </article>
              ))
            )}
          </div>
        </aside>
      </div>

      <section className={styles.panel}>
        <div className={styles.panelHeader}>
          <div>
            <span className="badge-pill">Заказы</span>
            <h2>Последние продажи магазина</h2>
          </div>
          <Link to="/seller/orders" className="action-secondary">
            Все заказы
          </Link>
        </div>

        <div className={styles.orderGrid}>
          {dashboard.recentOrders.length === 0 ? (
            <div className="empty-state">
              <h2>Пока нет заказов</h2>
              <p>Как только покупатели оформят товары вашего магазина, они появятся здесь.</p>
            </div>
          ) : (
            dashboard.recentOrders.map((order) => (
              <article key={order.orderId} className={styles.orderCard}>
                <div className={styles.orderTop}>
                  <div>
                    <h3>Заказ #{order.orderId.slice(0, 8)}</h3>
                    <p>{formatDate(order.createdAt)}</p>
                  </div>
                  <div className={styles.badgeRow}>
                    <span className={styles.badgeMuted}>{order.itemsCount} поз.</span>
                    <span className={styles.badge}>{formatCurrency(order.grossRevenue, order.currency)}</span>
                  </div>
                </div>
                <p className={styles.listMeta}>{order.placeTitle || 'Адрес покупателя'}</p>
              </article>
            ))
          )}
        </div>
      </section>
    </div>
  )
}
