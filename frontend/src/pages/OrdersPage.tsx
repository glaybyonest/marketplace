import { useEffect } from 'react'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchOrdersThunk } from '@/store/slices/ordersSlice'
import { formatCurrency, formatDate } from '@/utils/format'

import styles from '@/pages/OrdersPage.module.scss'

export const OrdersPage = () => {
  const dispatch = useAppDispatch()
  const { items, status, error } = useAppSelector((state) => state.orders)

  useEffect(() => {
    dispatch(fetchOrdersThunk({ page: 1, limit: 20 }))
  }, [dispatch])

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1>Orders</h1>
          <p>History of placed orders.</p>
        </div>
      </header>

      {status === 'loading' ? <AppLoader label="Loading orders..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {items.length === 0 ? (
        <section className={styles.empty}>
          <h2>No orders yet</h2>
          <p>Your checkout history will appear here.</p>
        </section>
      ) : (
        <section className={styles.list}>
          {items.map((order) => (
            <article key={order.id} className={styles.card}>
              <div className={styles.topRow}>
                <div>
                  <h2>Order #{order.id.slice(0, 8)}</h2>
                  <p>{formatDate(order.createdAt)}</p>
                </div>
                <span className={styles.status}>{order.status}</span>
              </div>

              <div className={styles.place}>
                <strong>{order.placeTitle}</strong>
                <p>{order.addressText}</p>
              </div>

              <ul className={styles.items}>
                {order.items.map((item) => (
                  <li key={item.id}>
                    <span>{item.title} x {item.quantity}</span>
                    <strong>{formatCurrency(item.lineTotal, item.currency ?? order.currency)}</strong>
                  </li>
                ))}
              </ul>

              <div className={styles.totalRow}>
                <span>Total</span>
                <strong>{formatCurrency(order.total, order.currency)}</strong>
              </div>
            </article>
          ))}
        </section>
      )}
    </div>
  )
}
