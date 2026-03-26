import { useEffect, useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import {
  clearCartThunk,
  fetchCartThunk,
  removeCartItemThunk,
  updateCartItemThunk,
} from '@/store/slices/cartSlice'
import { formatCurrency } from '@/utils/format'
import { resolveCartItemImage, resolveCartItemImageFallback, swapImageToFallback } from '@/utils/media'
import { getProductPath } from '@/utils/productRef'

import styles from '@/pages/CartPage.module.scss'

export const CartPage = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const { items, total, currency, totalItems, status, mutationStatus, error } = useAppSelector((state) => state.cart)

  useEffect(() => {
    dispatch(fetchCartThunk())
  }, [dispatch])

  const regularTotal = useMemo(
    () =>
      items.reduce((sum, item) => {
        const oldPrice = Math.round(item.price * 1.12)
        return sum + oldPrice * item.quantity
      }, 0),
    [items],
  )

  const benefit = Math.max(0, regularTotal - total)

  const handleDecrease = (productId: string, quantity: number) => {
    if (quantity <= 1) {
      dispatch(removeCartItemThunk(productId))
      return
    }

    dispatch(updateCartItemThunk({ productId, quantity: quantity - 1 }))
  }

  const handleIncrease = (productId: string, quantity: number, stock?: number) => {
    if (stock !== undefined && quantity >= stock) {
      return
    }

    dispatch(updateCartItemThunk({ productId, quantity: quantity + 1 }))
  }

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="badge-pill">Корзина</span>
          <h1>Проверьте состав заказа</h1>
        </div>
        <div className={styles.headerActions}>
          <Link to="/" className="action-secondary">
            Продолжить покупки
          </Link>
          {items.length > 0 ? (
            <button type="button" className="action-ghost" onClick={() => dispatch(clearCartThunk())}>
              Очистить корзину
            </button>
          ) : null}
        </div>
      </header>

      {status === 'loading' ? <AppLoader label="Загружаем корзину..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {items.length === 0 ? (
        <section className={`${styles.empty} empty-state`}>
          <h2>Корзина пока пустая</h2>
          <p>Добавьте товары из каталога, чтобы перейти к оформлению заказа и выбору адреса.</p>
          <Link to="/" className="action-primary">
            Перейти в каталог
          </Link>
        </section>
      ) : (
        <div className={styles.layout}>
          <section className={styles.list}>
            {items.map((item) => (
              <article key={item.id} className={styles.item}>
                <Link to={getProductPath(item)} className={styles.imageWrap}>
                  <img
                    src={resolveCartItemImage(item)}
                    alt={item.title}
                    className={styles.image}
                    onError={(event) => swapImageToFallback(event.currentTarget, resolveCartItemImageFallback(item))}
                  />
                </Link>

                <div className={styles.itemMain}>
                  <div className={styles.itemTop}>
                    <div>
                      <Link to={getProductPath(item)} className={styles.itemTitle}>
                        {item.title}
                      </Link>
                      <p className={styles.meta}>
                        {item.sku ? `Артикул ${item.sku}` : 'Товар из каталога'}
                        {item.sellerName ? ` | ${item.sellerName}` : ''}
                        {item.stock !== undefined ? ` | Доступно ${item.stock}` : ''}
                      </p>
                    </div>
                    <button
                      type="button"
                      className={styles.removeButton}
                      onClick={() => dispatch(removeCartItemThunk(item.productId))}
                      disabled={mutationStatus === 'loading'}
                    >
                      Удалить
                    </button>
                  </div>

                  <div className={styles.itemBottom}>
                    <div className={styles.priceBlock}>
                      <span className={styles.oldPrice}>{formatCurrency(Math.round(item.price * 1.12), item.currency ?? currency)}</span>
                      <strong>{formatCurrency(item.price, item.currency ?? currency)}</strong>
                    </div>

                    <div className={styles.quantity}>
                      <button
                        type="button"
                        onClick={() => handleDecrease(item.productId, item.quantity)}
                        disabled={mutationStatus === 'loading'}
                      >
                        -
                      </button>
                      <span>{item.quantity}</span>
                      <button
                        type="button"
                        onClick={() => handleIncrease(item.productId, item.quantity, item.stock)}
                        disabled={mutationStatus === 'loading' || (item.stock !== undefined && item.quantity >= item.stock)}
                      >
                        +
                      </button>
                    </div>

                    <div className={styles.lineTotal}>
                      <span>Итого</span>
                      <strong>{formatCurrency(item.lineTotal, item.currency ?? currency)}</strong>
                    </div>
                  </div>
                </div>
              </article>
            ))}
          </section>

          <aside className={`${styles.summary} summary-card`}>
            <div className={styles.summaryBlock}>
              <span className="badge-pill">Ваш заказ</span>
              <h2>{totalItems} позиций</h2>
            </div>

            <dl className={styles.summaryRows}>
              <div>
                <dt>Товары</dt>
                <dd>{formatCurrency(regularTotal, currency)}</dd>
              </div>
              <div>
                <dt>Скидка витрины</dt>
                <dd className={styles.benefit}>{formatCurrency(benefit, currency)}</dd>
              </div>
              <div>
                <dt>Доставка</dt>
                <dd>Рассчитывается на следующем шаге</dd>
              </div>
            </dl>

            <div className={styles.totalBlock}>
              <span>К оплате</span>
              <strong>{formatCurrency(total, currency)}</strong>
            </div>

            <button type="button" className="action-primary" onClick={() => navigate('/checkout')}>
              Перейти к оформлению
            </button>
          </aside>
        </div>
      )}
    </div>
  )
}
