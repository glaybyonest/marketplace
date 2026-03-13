import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { addCartItemThunk } from '@/store/slices/cartSlice'
import { addFavoriteThunk, removeFavoriteThunk } from '@/store/slices/favoritesSlice'
import { fetchProductByIdThunk } from '@/store/slices/productsSlice'
import { fetchRecommendationsThunk } from '@/store/slices/recommendationsSlice'
import { formatCurrency } from '@/utils/format'

import styles from '@/pages/ProductPage.module.scss'

export const ProductPage = () => {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const dispatch = useAppDispatch()

  const auth = useAppSelector((state) => state.auth)
  const product = useAppSelector((state) => state.products.selectedProduct)
  const detailStatus = useAppSelector((state) => state.products.detailStatus)
  const productError = useAppSelector((state) => state.products.error)
  const favoriteItems = useAppSelector((state) => state.favorites.items)
  const favoriteMutationStatus = useAppSelector((state) => state.favorites.mutationStatus)
  const cartMutationStatus = useAppSelector((state) => state.cart.mutationStatus)
  const recommendations = useAppSelector((state) => state.recommendations.items)

  const [selectedImage, setSelectedImage] = useState<string>('')
  const [quantity, setQuantity] = useState(1)

  useEffect(() => {
    if (!id) {
      return
    }
    dispatch(fetchProductByIdThunk(id))
  }, [dispatch, id])

  useEffect(() => {
    if (auth.isAuthenticated) {
      dispatch(fetchRecommendationsThunk(8))
    }
  }, [auth.isAuthenticated, dispatch])

  const isFavorite = useMemo(
    () => favoriteItems.some((item) => item.id === id),
    [favoriteItems, id],
  )

  const gallery = product?.images.length
    ? product.images
    : ['https://placehold.co/500x400?text=No+Image']
  const activeImage = selectedImage && gallery.includes(selectedImage) ? selectedImage : gallery[0]
  const safeQuantity = Math.min(quantity, Math.max(product?.stock || 1, 1))

  const handleToggleFavorite = async () => {
    if (!product) {
      return
    }

    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    if (isFavorite) {
      await dispatch(removeFavoriteThunk(product.id))
    } else {
      await dispatch(addFavoriteThunk(product.id))
    }
  }

  const handleAddToCart = async () => {
    if (!product) {
      return
    }

    if (!auth.isAuthenticated) {
      navigate('/login')
      return
    }

    await dispatch(addCartItemThunk({ productId: product.id, quantity: safeQuantity }))
    navigate('/cart')
  }

  if (detailStatus === 'loading') {
    return <AppLoader label="Loading product..." />
  }

  if (productError) {
    return <ErrorMessage message={productError} />
  }

  if (!product) {
    return <ErrorMessage message="Product not found" />
  }

  return (
    <div className={styles.page}>
      <section className={styles.product}>
        <div className={styles.gallery}>
          <div className={styles.mainImageWrapper}>
            <img src={activeImage} alt={product.title} className={styles.mainImage} />
          </div>
          <div className={styles.thumbs}>
            {gallery.map((image, index) => (
              <button
                type="button"
                key={`${image}-${index}`}
                className={image === activeImage ? styles.thumbActive : styles.thumb}
                onClick={() => setSelectedImage(image)}
              >
                <img src={image} alt={`${product.title} preview ${index + 1}`} />
              </button>
            ))}
          </div>
        </div>

        <div className={styles.info}>
          <h1 className={styles.title}>{product.title}</h1>
          <div className={styles.priceBlock}>
            <p className={styles.price}>{formatCurrency(product.price, product.currency)}</p>
          </div>

          <p className={styles.description}>{product.description || 'No description'}</p>

          <div className={`${styles.stock} ${product.stock && product.stock > 0 ? styles.inStock : styles.outOfStock}`}>
            <span>{product.stock && product.stock > 0 ? `In stock: ${product.stock}` : 'Out of stock'}</span>
          </div>

          <div className={styles.quantity}>
            <span className={styles.quantityLabel}>Quantity</span>
            <div className={styles.quantityControls}>
              <button
                type="button"
                onClick={() => setQuantity((current) => Math.max(1, current - 1))}
                disabled={safeQuantity <= 1}
              >
                -
              </button>
              <span>{safeQuantity}</span>
              <button
                type="button"
                onClick={() => setQuantity((current) => Math.min(product.stock || 1, current + 1))}
                disabled={!product.stock || safeQuantity >= product.stock}
              >
                +
              </button>
            </div>
          </div>

          <div className={styles.actionGroup}>
            <button
              type="button"
              className={`${styles.cartButton} ${!product.stock ? styles.cartButtonDisabled : ''}`}
              onClick={handleAddToCart}
              disabled={!product.stock || cartMutationStatus === 'loading'}
            >
              {cartMutationStatus === 'loading' ? 'Adding...' : 'Add to cart'}
            </button>

            <button
              type="button"
              className={styles.favoriteButton}
              onClick={handleToggleFavorite}
              disabled={favoriteMutationStatus === 'loading'}
            >
              {isFavorite ? 'Remove from favorites' : 'Add to favorites'}
            </button>
          </div>

        </div>
      </section>

      {auth.isAuthenticated && recommendations.length > 0 ? (
        <section className={styles.reviews}>
          <div className={styles.reviewsHeader}>
            <h2>Recommended for you</h2>
          </div>
          <div className={styles.reviewList}>
            {recommendations
              .filter((item) => item.id !== product.id)
              .slice(0, 6)
              .map((item) => (
                <article key={item.id} className={styles.reviewItem}>
                  <h3>{item.title}</h3>
                  <p>{formatCurrency(item.price, item.currency)}</p>
                  <Link to={`/products/${item.id}`}>Open</Link>
                </article>
              ))}
          </div>
        </section>
      ) : null}
    </div>
  )
}
