import type { CSSProperties } from 'react'
import { useMemo } from 'react'

import type { Product } from '@/types/domain'
import { formatCurrency } from '@/utils/format'
import { resolveProductImage, resolveProductImageFallback, swapImageToFallback } from '@/utils/media'

import styles from '@/components/catalog/ProductInfographic.module.scss'

interface ProductInfographicProps {
  products: Product[]
  total?: number
}

const buildBreakdown = (values: string[], limit: number) => {
  const counts = new Map<string, number>()

  for (const value of values) {
    const key = value.trim()
    if (!key) {
      continue
    }
    counts.set(key, (counts.get(key) ?? 0) + 1)
  }

  const total = values.length || 1

  return Array.from(counts.entries())
    .sort((left, right) => right[1] - left[1])
    .slice(0, limit)
    .map(([label, value]) => ({
      label,
      value,
      share: Math.max(14, Math.round((value / total) * 100)),
    }))
}

export const ProductInfographic = ({ products, total }: ProductInfographicProps) => {
  const metrics = useMemo(() => {
    const visibleCount = products.length
    const pricedItems = products.filter((product) => Number.isFinite(product.price))
    const availableCount = products.filter((product) => (product.stock ?? 0) > 0).length
    const totalStock = products.reduce((sum, product) => sum + Math.max(product.stock ?? 0, 0), 0)
    const averagePrice =
      pricedItems.length > 0
        ? Math.round(pricedItems.reduce((sum, product) => sum + product.price, 0) / pricedItems.length)
        : 0
    const maxPrice = pricedItems.reduce((max, product) => Math.max(max, product.price), 0)
    const minPrice =
      pricedItems.length > 0
        ? pricedItems.reduce((min, product) => Math.min(min, product.price), pricedItems[0]?.price ?? 0)
        : 0
    const coverage = total && total > visibleCount ? Math.round((visibleCount / total) * 100) : 100

    return {
      visibleCount,
      availableCount,
      totalStock,
      averagePrice,
      maxPrice,
      minPrice,
      coverage,
      categories: buildBreakdown(
        products.map((product) => product.categoryName || 'Каталог'),
        4,
      ),
      brands: buildBreakdown(
        products.map((product) => product.brand || product.sellerName || 'Marketplace'),
        6,
      ),
      showcase: products.slice(0, 5),
    }
  }, [products, total])

  if (metrics.visibleCount === 0) {
    return null
  }

  return (
    <section className={styles.section}>
      <div className={styles.hero}>
        <div className={styles.copy}>
          <span className={styles.eyebrow}>Инфографика витрины</span>
          <h2>Товары с реальными фото и быстрой сводкой по ассортименту</h2>
          <p>
            Блок собирается из текущей выборки каталога, поэтому карточки, категории и ценовой срез
            обновляются вместе с фильтрами.
          </p>

          <div className={styles.metricsGrid}>
            <article className={styles.metricCard}>
              <span>Карточек на экране</span>
              <strong>{metrics.visibleCount}</strong>
              <small>{metrics.coverage}% от текущей выдачи</small>
            </article>
            <article className={styles.metricCard}>
              <span>Средняя цена</span>
              <strong>{formatCurrency(metrics.averagePrice, products[0]?.currency || 'RUB')}</strong>
              <small>Диапазон до {formatCurrency(metrics.maxPrice, products[0]?.currency || 'RUB')}</small>
            </article>
            <article className={styles.metricCard}>
              <span>В наличии</span>
              <strong>{metrics.availableCount}</strong>
              <small>{metrics.totalStock} единиц на складе</small>
            </article>
          </div>
        </div>

        <div className={styles.mosaic}>
          {metrics.showcase.map((product, index) => {
            const primaryImage = resolveProductImage(product, index)
            const fallbackImage = resolveProductImageFallback(product, index)

            return (
              <article
                key={product.id}
                className={`${styles.shot} ${
                  index === 0
                    ? styles.shotHero
                    : index === 1
                      ? styles.shotTall
                      : index === 2
                        ? styles.shotWide
                        : styles.shotCompact
                }`}
              >
                <img
                  src={primaryImage}
                  alt={product.title}
                  loading={index === 0 ? 'eager' : 'lazy'}
                  onError={(event) => swapImageToFallback(event.currentTarget, fallbackImage)}
                />
                <div className={styles.shotOverlay}>
                  <span>{product.categoryName || 'Каталог'}</span>
                  <strong>{product.title}</strong>
                  <small>{formatCurrency(product.price, product.currency)}</small>
                </div>
              </article>
            )
          })}
        </div>
      </div>

      <div className={styles.dataGrid}>
        <article className={styles.chartCard}>
          <div className={styles.cardHeader}>
            <div>
              <span>Категории</span>
              <h3>Где сейчас больше всего карточек</h3>
            </div>
            <strong>{metrics.categories.length}</strong>
          </div>

          <div className={styles.barList}>
            {metrics.categories.map((item) => (
              <div key={item.label} className={styles.barRow}>
                <div className={styles.barMeta}>
                  <span>{item.label}</span>
                  <strong>{item.value}</strong>
                </div>
                <div className={styles.barTrack}>
                  <div className={styles.barFill} style={{ width: `${item.share}%` }} />
                </div>
              </div>
            ))}
          </div>
        </article>

        <article className={styles.chartCard}>
          <div className={styles.cardHeader}>
            <div>
              <span>Бренды и магазины</span>
              <h3>Кто чаще встречается в подборке</h3>
            </div>
            <strong>{metrics.brands.length}</strong>
          </div>

          <div className={styles.brandCloud}>
            {metrics.brands.map((item, index) => (
              <span
                key={item.label}
                className={styles.brandChip}
                style={{ '--brand-delay': `${index * 60}ms` } as CSSProperties}
              >
                {item.label}
                <strong>{item.value}</strong>
              </span>
            ))}
          </div>

          <div className={styles.priceRange}>
            <div>
              <span>Минимум</span>
              <strong>{formatCurrency(metrics.minPrice, products[0]?.currency || 'RUB')}</strong>
            </div>
            <div>
              <span>Потолок</span>
              <strong>{formatCurrency(metrics.maxPrice, products[0]?.currency || 'RUB')}</strong>
            </div>
          </div>
        </article>
      </div>
    </section>
  )
}
