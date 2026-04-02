import { useEffect, useState } from 'react'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { SellerNav } from '@/components/seller/SellerNav'
import { sellerService } from '@/services/sellerService'
import type { SellerOrderSummary } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { formatCurrency, formatDate } from '@/utils/format'

import styles from '@/pages/SellerPage.module.scss'

type AsyncState = 'idle' | 'loading' | 'succeeded' | 'failed'

export const SellerOrdersPage = () => {
  const [items, setItems] = useState<SellerOrderSummary[]>([])
  const [status, setStatus] = useState<AsyncState>('loading')
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [totalPages, setTotalPages] = useState(1)

  useEffect(() => {
    let cancelled = false

    const loadOrders = async () => {
      setStatus('loading')
      setError(null)
      try {
        const response = await sellerService.getOrders(page, 12)
        if (cancelled) {
          return
        }
        setItems(response.items)
        setTotalPages(response.totalPages)
        setStatus('succeeded')
      } catch (loadError) {
        if (cancelled) {
          return
        }
        setStatus('failed')
        setError(getErrorMessage(loadError, 'Не удалось загрузить заказы продавца'))
      }
    }

    void loadOrders()

    return () => {
      cancelled = true
    }
  }, [page])

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroTop}>
          <div className={styles.heroCopy}>
            <span className={styles.heroBadge}>Заказы продавца</span>
            <h1>Заказы по вашему магазину</h1>
            <p>Здесь видна только часть заказа, которая относится к вашему магазину: позиции, адрес получения и выручка.</p>
          </div>
          <SellerNav />
        </div>
      </section>

      {status === 'loading' ? <AppLoader label="Загружаем заказы продавца..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <section className={styles.panel}>
        <div className={styles.panelHeader}>
          <div>
            <span className="badge-pill">Список</span>
            <h2>Последние заказы</h2>
          </div>
        </div>

        <div className={styles.orderGrid}>
          {items.length === 0 && status !== 'loading' ? (
            <div className="empty-state">
              <h2>Заказов пока нет</h2>
              <p>Когда покупатели оформят позиции вашего магазина, они появятся здесь.</p>
            </div>
          ) : (
            items.map((order) => (
              <article key={order.orderId} className={styles.orderCard}>
                <div className={styles.orderTop}>
                  <div>
                    <h3>Заказ #{order.orderId.slice(0, 8)}</h3>
                    <p>{formatDate(order.createdAt)}</p>
                  </div>
                  <div className={styles.badgeRow}>
                    <span className={styles.badgeMuted}>{order.itemsCount} поз.</span>
                    <span className={styles.badgePrice}>{formatCurrency(order.grossRevenue, order.currency)}</span>
                  </div>
                </div>
                <p className={styles.listMeta}>{order.placeTitle || 'Адрес покупателя'}</p>
              </article>
            ))
          )}
        </div>

        {totalPages > 1 ? (
          <div className={styles.pagination}>
            <button type="button" className="action-secondary" disabled={page <= 1} onClick={() => setPage((current) => current - 1)}>
              Назад
            </button>
            <span className={styles.helper}>
              Страница {page} из {totalPages}
            </span>
            <button
              type="button"
              className="action-secondary"
              disabled={page >= totalPages}
              onClick={() => setPage((current) => current + 1)}
            >
              Дальше
            </button>
          </div>
        ) : null}
      </section>
    </div>
  )
}
