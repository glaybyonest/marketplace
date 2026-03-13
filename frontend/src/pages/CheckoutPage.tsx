import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchCartThunk } from '@/store/slices/cartSlice'
import { checkoutThunk } from '@/store/slices/ordersSlice'
import { fetchPlacesThunk } from '@/store/slices/placesSlice'
import { formatCurrency } from '@/utils/format'

import styles from '@/pages/CheckoutPage.module.scss'

export const CheckoutPage = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const cart = useAppSelector((state) => state.cart)
  const places = useAppSelector((state) => state.places)
  const orders = useAppSelector((state) => state.orders)

  const [selectedPlaceId, setSelectedPlaceId] = useState('')

  useEffect(() => {
    dispatch(fetchCartThunk())
    dispatch(fetchPlacesThunk())
  }, [dispatch])

  const activePlaceId =
    places.items.some((place) => place.id === selectedPlaceId) ? selectedPlaceId : (places.items[0]?.id ?? '')

  const handleCheckout = async () => {
    if (!activePlaceId) {
      return
    }

    const result = await dispatch(checkoutThunk(activePlaceId))
    if (checkoutThunk.fulfilled.match(result)) {
      navigate('/account/orders')
    }
  }

  const isLoading = cart.status === 'loading' || places.status === 'loading'

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1>Checkout</h1>
          <p>Select a saved place and confirm your order.</p>
        </div>
      </header>

      {isLoading ? <AppLoader label="Preparing checkout..." /> : null}
      {cart.error ? <ErrorMessage message={cart.error} /> : null}
      {places.error ? <ErrorMessage message={places.error} /> : null}
      {orders.error ? <ErrorMessage message={orders.error} /> : null}

      {cart.items.length === 0 ? (
        <section className={styles.empty}>
          <h2>Cart is empty</h2>
          <p>You need at least one item before checkout.</p>
          <Link to="/cart" className={styles.primaryLink}>
            Open cart
          </Link>
        </section>
      ) : places.items.length === 0 ? (
        <section className={styles.empty}>
          <h2>No saved places</h2>
          <p>Create at least one place before placing an order.</p>
          <Link to="/account/places" className={styles.primaryLink}>
            Manage places
          </Link>
        </section>
      ) : (
        <div className={styles.layout}>
          <section className={styles.places}>
            <h2>Delivery place</h2>
            <div className={styles.placeList}>
              {places.items.map((place) => (
                <label key={place.id} className={`${styles.placeCard} ${activePlaceId === place.id ? styles.placeCardActive : ''}`}>
                  <input
                    type="radio"
                    name="place"
                    value={place.id}
                    checked={activePlaceId === place.id}
                    onChange={() => setSelectedPlaceId(place.id)}
                  />
                  <div>
                    <strong>{place.title}</strong>
                    <p>{place.addressText}</p>
                  </div>
                </label>
              ))}
            </div>
          </section>

          <aside className={styles.summary}>
            <h2>Order summary</h2>
            <ul className={styles.summaryList}>
              {cart.items.map((item) => (
                <li key={item.id}>
                  <span>{item.title} x {item.quantity}</span>
                  <strong>{formatCurrency(item.lineTotal, item.currency ?? cart.currency)}</strong>
                </li>
              ))}
            </ul>

            <dl>
              <div>
                <dt>Items</dt>
                <dd>{cart.totalItems}</dd>
              </div>
              <div>
                <dt>Total</dt>
                <dd>{formatCurrency(cart.total, cart.currency)}</dd>
              </div>
            </dl>

            <button type="button" className={styles.primaryButton} onClick={handleCheckout} disabled={!activePlaceId || orders.checkoutStatus === 'loading'}>
              {orders.checkoutStatus === 'loading' ? 'Placing order...' : 'Place order'}
            </button>
          </aside>
        </div>
      )}
    </div>
  )
}
