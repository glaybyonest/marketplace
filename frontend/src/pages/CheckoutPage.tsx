import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchCartThunk } from '@/store/slices/cartSlice'
import { checkoutThunk, fetchOrdersThunk } from '@/store/slices/ordersSlice'
import { fetchPlacesThunk } from '@/store/slices/placesSlice'
import { formatCurrency } from '@/utils/format'
import { resolveCartItemImage, resolveCartItemImageFallback, swapImageToFallback } from '@/utils/media'

import styles from '@/pages/CheckoutPage.module.scss'

type PaymentMethod = 'bank_card' | 'sbp' | 'cash_on_delivery'

const paymentOptions: Array<{ id: PaymentMethod; title: string; description: string; meta: string }> = [
  {
    id: 'bank_card',
    title: 'Банковская карта',
    description: 'Тестовая оплата без реального списания. Используется как заглушка для оформления.',
    meta: 'Заглушка',
  },
  {
    id: 'sbp',
    title: 'СБП',
    description: 'Имитация быстрой оплаты по СБП. Заказ создаётся сразу после подтверждения.',
    meta: 'Тестовый режим',
  },
  {
    id: 'cash_on_delivery',
    title: 'При получении',
    description: 'Оплата будет отмечена как отложенная. Товар оформится и попадёт в историю заказов.',
    meta: 'Без списания',
  },
]

export const CheckoutPage = () => {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const cart = useAppSelector((state) => state.cart)
  const places = useAppSelector((state) => state.places)
  const orders = useAppSelector((state) => state.orders)

  const [selectedPlaceId, setSelectedPlaceId] = useState('')
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>('bank_card')

  useEffect(() => {
    dispatch(fetchCartThunk())
    dispatch(fetchPlacesThunk())
  }, [dispatch])

  const activePlaceId =
    places.items.some((place) => place.id === selectedPlaceId) ? selectedPlaceId : (places.items[0]?.id ?? '')

  const activePlace = places.items.find((place) => place.id === activePlaceId)
  const activePayment = paymentOptions.find((item) => item.id === paymentMethod) ?? paymentOptions[0]

  const handleCheckout = async () => {
    if (!activePlaceId) {
      return
    }

    const result = await dispatch(checkoutThunk(activePlaceId))
    if (!checkoutThunk.fulfilled.match(result)) {
      return
    }

    await dispatch(fetchOrdersThunk({ page: 1, limit: 20 }))
    navigate('/account/orders', {
      state: {
        message: `Заказ #${result.payload.id.slice(0, 8)} оформлен.`,
      },
    })
  }

  const isLoading = cart.status === 'loading' || places.status === 'loading'

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <span className="badge-pill">Оформление</span>
          <h1>Подтвердите адрес, состав и оплату</h1>
        </div>
      </header>

      {isLoading ? <AppLoader label="Подготавливаем оформление..." /> : null}
      {cart.error ? <ErrorMessage message={cart.error} /> : null}
      {places.error ? <ErrorMessage message={places.error} /> : null}
      {orders.error ? <ErrorMessage message={orders.error} /> : null}

      {cart.items.length === 0 ? (
        <section className={`${styles.empty} empty-state`}>
          <h2>Нечего оформлять</h2>
          <p>В корзине пока нет товаров. Сначала добавьте позиции в каталоге.</p>
          <Link to="/cart" className="action-primary">
            Открыть корзину
          </Link>
        </section>
      ) : places.items.length === 0 ? (
        <section className={`${styles.empty} empty-state`}>
          <h2>Нет сохранённых адресов</h2>
          <p>Добавьте хотя бы одно место доставки в личном кабинете, чтобы продолжить оформление.</p>
          <Link to="/account/places" className="action-primary">
            Управлять адресами
          </Link>
        </section>
      ) : (
        <div className={styles.layout}>
          <section className={styles.mainColumn}>
            <section className={`${styles.sectionCard} page-card`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className="badge-pill">Шаг 1</span>
                  <h2>Куда доставить заказ</h2>
                </div>
                <Link to="/account/places" className="action-ghost">
                  Изменить адреса
                </Link>
              </div>

              <div className={styles.placeList}>
                {places.items.map((place) => (
                  <label
                    key={place.id}
                    className={`${styles.placeCard} ${activePlaceId === place.id ? styles.placeCardActive : ''}`}
                  >
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

            <section className={`${styles.sectionCard} page-card`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className="badge-pill">Шаг 2</span>
                  <h2>Состав заказа</h2>
                </div>
                <Link to="/cart" className="action-ghost">
                  Вернуться в корзину
                </Link>
              </div>

              <ul className={styles.orderList}>
                {cart.items.map((item) => (
                  <li key={item.id} className={styles.orderItem}>
                    <img
                      src={resolveCartItemImage(item)}
                      alt={item.title}
                      onError={(event) => swapImageToFallback(event.currentTarget, resolveCartItemImageFallback(item))}
                    />
                    <div className={styles.orderCopy}>
                      <strong>{item.title}</strong>
                      <span>
                        {item.sku ? `Артикул ${item.sku}` : 'Товар из каталога'}
                        {item.sellerName ? ` • ${item.sellerName}` : ''}
                      </span>
                    </div>
                    <span className={styles.orderQty}>x{item.quantity}</span>
                    <strong>{formatCurrency(item.lineTotal, item.currency ?? cart.currency)}</strong>
                  </li>
                ))}
              </ul>
            </section>

            <section className={`${styles.sectionCard} page-card`}>
              <div className={styles.sectionHeading}>
                <div>
                  <span className="badge-pill">Шаг 3</span>
                  <h2>Способ оплаты</h2>
                </div>
              </div>

              <div className={styles.paymentList}>
                {paymentOptions.map((option) => (
                  <label
                    key={option.id}
                    className={`${styles.paymentCard} ${paymentMethod === option.id ? styles.paymentCardActive : ''}`}
                  >
                    <input
                      type="radio"
                      name="payment"
                      value={option.id}
                      checked={paymentMethod === option.id}
                      onChange={() => setPaymentMethod(option.id)}
                    />
                    <div className={styles.paymentCopy}>
                      <strong>{option.title}</strong>
                      <p>{option.description}</p>
                    </div>
                    <span className={styles.paymentMeta}>{option.meta}</span>
                  </label>
                ))}
              </div>
            </section>
          </section>

          <aside className={`${styles.summary} summary-card`}>
            <div className={styles.summaryTop}>
              <span className="badge-pill">Итог</span>
              <h2>Заказ готов к оформлению</h2>
            </div>

            <div className={styles.deliveryInfo}>
              <strong>{activePlace?.title || 'Адрес не выбран'}</strong>
              <p>{activePlace?.addressText || 'Выберите место доставки слева.'}</p>
            </div>

            <dl className={styles.summaryRows}>
              <div>
                <dt>Товаров</dt>
                <dd>{cart.totalItems}</dd>
              </div>
              <div>
                <dt>Сумма заказа</dt>
                <dd>{formatCurrency(cart.total, cart.currency)}</dd>
              </div>
              <div>
                <dt>Оплата</dt>
                <dd>{activePayment.title}</dd>
              </div>
              <div>
                <dt>Получение</dt>
                <dd>По выбранному адресу</dd>
              </div>
            </dl>

            <button
              type="button"
              className="action-primary"
              onClick={handleCheckout}
              disabled={!activePlaceId || orders.checkoutStatus === 'loading'}
            >
              {orders.checkoutStatus === 'loading' ? 'Оформляем заказ...' : 'Оплатить и оформить'}
            </button>

            <p className={styles.helper}>После оформления корзина очистится, а новый заказ появится в истории покупок.</p>
          </aside>
        </div>
      )}
    </div>
  )
}
