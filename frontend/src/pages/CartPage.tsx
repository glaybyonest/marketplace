import { useEffect } from 'react'
import { Link } from 'react-router-dom'

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

import styles from '@/pages/CartPage.module.scss'

export const CartPage = () => {
  const dispatch = useAppDispatch()
  const { items, total, currency, totalItems, status, mutationStatus, error } = useAppSelector((state) => state.cart)

  useEffect(() => {
    dispatch(fetchCartThunk())
  }, [dispatch])

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
          <h1>Cart</h1>
          <p>Review items before checkout.</p>
        </div>
        {items.length > 0 ? (
          <button type="button" className={styles.secondary} onClick={() => dispatch(clearCartThunk())}>
            Clear cart
          </button>
        ) : null}
      </header>

      {status === 'loading' ? <AppLoader label="Loading cart..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      {items.length === 0 ? (
        <section className={styles.empty}>
          <h2>Your cart is empty</h2>
          <p>Add products from the catalog to start checkout.</p>
          <Link to="/" className={styles.primaryLink}>
            Browse catalog
          </Link>
        </section>
      ) : (
        <div className={styles.layout}>
          <section className={styles.list}>
            {items.map((item) => (
              <article key={item.id} className={styles.item}>
                <div className={styles.itemMain}>
                  <Link to={`/products/${item.productId}`} className={styles.itemTitle}>
                    {item.title}
                  </Link>
                  <p className={styles.meta}>
                    {item.sku ? `SKU: ${item.sku}` : 'Cart item'}
                    {item.stock !== undefined ? ` · Stock: ${item.stock}` : ''}
                  </p>
                  <p className={styles.price}>{formatCurrency(item.price, item.currency ?? currency)}</p>
                </div>

                <div className={styles.controls}>
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

                  <div className={styles.itemActions}>
                    <strong>{formatCurrency(item.lineTotal, item.currency ?? currency)}</strong>
                    <button
                      type="button"
                      className={styles.textButton}
                      onClick={() => dispatch(removeCartItemThunk(item.productId))}
                      disabled={mutationStatus === 'loading'}
                    >
                      Remove
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </section>

          <aside className={styles.summary}>
            <h2>Summary</h2>
            <dl>
              <div>
                <dt>Items</dt>
                <dd>{totalItems}</dd>
              </div>
              <div>
                <dt>Total</dt>
                <dd>{formatCurrency(total, currency)}</dd>
              </div>
            </dl>
            <Link to="/checkout" className={styles.primaryLink}>
              Continue to checkout
            </Link>
          </aside>
        </div>
      )}
    </div>
  )
}
