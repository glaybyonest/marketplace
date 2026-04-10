import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'

import { AppLoader } from '@/components/common/AppLoader'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { ProductCardAIAssistant } from '@/components/seller/ProductCardAIAssistant'
import { SellerNav } from '@/components/seller/SellerNav'
import { categoryService } from '@/services/categoryService'
import type { ProductCardAIDraft, ProductCardAIMode } from '@/services/sellerAiService'
import { sellerService } from '@/services/sellerService'
import type { ProductFilters } from '@/types/api'
import type { Category, Product } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'
import { formatCurrency, formatUnitLabel } from '@/utils/format'
import {
  isGeneratedMediaSource,
  resolveProductImage,
  resolveProductImageFallback,
  swapImageToFallback,
} from '@/utils/media'

import styles from '@/pages/SellerPage.module.scss'

interface ProductFormState {
  name: string
  slug: string
  sku: string
  categoryId: string
  description: string
  price: string
  currency: string
  stockQty: string
  imageUrl: string
  imagesText: string
  brand: string
  unit: string
  specsText: string
  isActive: boolean
}

const SELLER_PRODUCTS_BATCH_SIZE = 100

const initialFilters: ProductFilters = { sort: 'new' }

const shouldSyncSellerCategoryIds = (filters: ProductFilters) => {
  const query = (filters.q ?? filters.query ?? '').trim()
  const categoryId = filters.category_id ?? filters.category
  const isActive = filters.is_active ?? filters.isActive

  return !query && !categoryId && typeof isActive !== 'boolean'
}

const createInitialFormState = (): ProductFormState => ({
  name: '',
  slug: '',
  sku: '',
  categoryId: '',
  description: '',
  price: '',
  currency: 'RUB',
  stockQty: '0',
  imageUrl: '',
  imagesText: '',
  brand: '',
  unit: '',
  specsText: '{}',
  isActive: true,
})

const productToFormState = (product: Product): ProductFormState => ({
  name: product.name ?? product.title,
  slug: product.slug ?? '',
  sku: product.sku ?? '',
  categoryId: product.categoryId,
  description: product.description,
  price: String(product.price),
  currency: product.currency ?? 'RUB',
  stockQty: String(product.stock ?? 0),
  imageUrl: isGeneratedMediaSource(product.imageUrl) ? '' : (product.imageUrl ?? ''),
  imagesText: product.images.filter((image) => !isGeneratedMediaSource(image)).join('\n'),
  brand: product.brand ?? '',
  unit: product.unit ?? '',
  specsText: JSON.stringify(product.specs ?? {}, null, 2),
  isActive: product.isActive ?? true,
})

export const SellerProductsPage = () => {
  const [categories, setCategories] = useState<Category[]>([])
  const [sellerCategoryIds, setSellerCategoryIds] = useState<string[]>([])
  const [items, setItems] = useState<Product[]>([])
  const [filters, setFilters] = useState<ProductFilters>(initialFilters)
  const [searchDraft, setSearchDraft] = useState('')
  const [categoryDraft, setCategoryDraft] = useState('')
  const [statusDraft, setStatusDraft] = useState<'all' | 'visible' | 'hidden'>('all')
  const [catalogRequested, setCatalogRequested] = useState(false)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [editingProduct, setEditingProduct] = useState<Product | null>(null)
  const [formState, setFormState] = useState<ProductFormState>(createInitialFormState())
  const [aiAssistantOpen, setAIAssistantOpen] = useState(false)
  const [aiAssistantMode, setAIAssistantMode] = useState<ProductCardAIMode>('generate')
  const [stockDrafts, setStockDrafts] = useState<Record<string, string>>({})
  const [total, setTotal] = useState(0)

  const categoryMap = useMemo(
    () => new Map(categories.map((category) => [category.id, category.name])),
    [categories],
  )
  const sellerFilterCategories = useMemo(() => {
    const allowedIds = new Set(sellerCategoryIds)
    return categories.filter((category) => allowedIds.has(category.id))
  }, [categories, sellerCategoryIds])

  const loadProducts = useCallback(async (nextFilters: ProductFilters) => {
    setLoading(true)
    setError(null)
    try {
      const baseFilters = {
        ...nextFilters,
        page: 1,
        limit: SELLER_PRODUCTS_BATCH_SIZE,
      }

      const firstResponse = await sellerService.getProducts(baseFilters)
      const allItems = [...firstResponse.items]

      for (let nextPage = 2; nextPage <= firstResponse.totalPages; nextPage += 1) {
        const pageResponse = await sellerService.getProducts({
          ...baseFilters,
          page: nextPage,
        })
        allItems.push(...pageResponse.items)
      }

      if (shouldSyncSellerCategoryIds(nextFilters)) {
        setSellerCategoryIds([
          ...new Set(allItems.map((product) => product.categoryId).filter(Boolean)),
        ])
      }

      setItems(allItems)
      setTotal(firstResponse.total)
      setStockDrafts(
        Object.fromEntries(allItems.map((product) => [product.id, String(product.stock ?? 0)])),
      )
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Не удалось загрузить товары магазина'))
    } finally {
      setLoading(false)
    }
  }, [])

  const loadCategories = useCallback(async () => {
    try {
      const data = await categoryService.getCategories()
      setCategories(data)
    } catch (loadError) {
      setError(getErrorMessage(loadError, 'Не удалось загрузить категории'))
    }
  }, [])

  useEffect(() => {
    const loadInitial = async () => {
      await Promise.all([loadCategories(), loadProducts(initialFilters)])
    }

    void loadInitial()
  }, [loadCategories, loadProducts])

  const resetForm = () => {
    setEditingProduct(null)
    setFormState(createInitialFormState())
    setAIAssistantOpen(false)
    setAIAssistantMode('generate')
  }

  const handleEdit = (product: Product) => {
    setEditingProduct(product)
    setFormState(productToFormState(product))
  }

  const buildPayloadFromForm = () => {
    const parsedPrice = Number(formState.price)
    const parsedStock = Number(formState.stockQty)
    const trimmedSpecs = formState.specsText.trim()
    const parsedSpecs = trimmedSpecs.length > 0 ? JSON.parse(trimmedSpecs) : {}

    return {
      name: formState.name,
      slug: formState.slug || undefined,
      description: formState.description,
      price: parsedPrice,
      categoryId: formState.categoryId,
      currency: formState.currency,
      sku: formState.sku,
      imageUrl: formState.imageUrl || undefined,
      images: formState.imagesText
        .split('\n')
        .map((item) => item.trim())
        .filter(Boolean),
      brand: formState.brand || undefined,
      unit: formState.unit || undefined,
      specs: parsedSpecs,
      stock: parsedStock,
      isActive: formState.isActive,
    }
  }

  const openAIAssistant = (mode: ProductCardAIMode) => {
    setAIAssistantMode(mode)
    setAIAssistantOpen(true)
  }

  const handleApplyAIDraft = (draft: ProductCardAIDraft) => {
    setFormState((current) => ({
      ...current,
      name: draft.name || current.name,
      slug: draft.slug || current.slug,
      description: draft.description || current.description,
      brand: draft.brand || current.brand,
      unit: draft.unit || current.unit,
      specsText:
        Object.keys(draft.specs).length > 0
          ? JSON.stringify(draft.specs, null, 2)
          : current.specsText,
    }))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      const payload = buildPayloadFromForm()

      if (editingProduct) {
        await sellerService.updateProduct(editingProduct.id, payload)
      } else {
        await sellerService.createProduct(payload)
      }

      await loadProducts(filters)
      resetForm()
    } catch (submitError) {
      setError(getErrorMessage(submitError, 'Не удалось сохранить товар'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleStockSave = async (product: Product) => {
    const nextStock = Number(stockDrafts[product.id] ?? product.stock ?? 0)
    setSubmitting(true)
    setError(null)
    try {
      await sellerService.updateProductStock(product.id, nextStock)
      await loadProducts(filters)
    } catch (stockError) {
      setError(getErrorMessage(stockError, 'Не удалось обновить остаток'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleToggleActive = async (product: Product) => {
    setSubmitting(true)
    setError(null)
    try {
      await sellerService.updateProduct(product.id, {
        name: product.name ?? product.title,
        slug: product.slug,
        description: product.description,
        price: product.price,
        categoryId: product.categoryId,
        currency: product.currency,
        sku: product.sku,
        imageUrl: product.imageUrl,
        images: product.images,
        brand: product.brand,
        unit: product.unit,
        specs: product.specs,
        stock: product.stock,
        isActive: !(product.isActive ?? true),
      })
      await loadProducts(filters)
    } catch (toggleError) {
      setError(getErrorMessage(toggleError, 'Не удалось обновить статус товара'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleArchive = async (product: Product) => {
    if (!window.confirm(`Скрыть товар «${product.title}» из витрины магазина?`)) {
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await sellerService.deleteProduct(product.id)
      await loadProducts(filters)
      if (editingProduct?.id === product.id) {
        resetForm()
      }
    } catch (deleteError) {
      setError(getErrorMessage(deleteError, 'Не удалось скрыть товар'))
    } finally {
      setSubmitting(false)
    }
  }

  const submitCatalogFilters = () => {
    const nextFilters: ProductFilters = {
      ...initialFilters,
      q: searchDraft.trim() || undefined,
      category_id: categoryDraft || undefined,
      is_active: statusDraft === 'all' ? undefined : statusDraft === 'visible',
    }

    setCatalogRequested(true)
    setFilters(nextFilters)
    void loadProducts(nextFilters)
  }

  const resetFilters = () => {
    setSearchDraft('')
    setCategoryDraft('')
    setStatusDraft('all')
    setCatalogRequested(false)
    setFilters({ ...initialFilters })
  }

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroTop}>
          <div className={styles.heroCopy}>
            <span className={styles.heroBadge}>Товары магазина</span>
            <h1>Управляйте ассортиментом продавца</h1>
            <p>
              Цена, остатки, изображения, характеристики и видимость карточек обновляются из одного
              раздела.
            </p>
          </div>
          <SellerNav />
        </div>

        <div className={styles.heroMeta}>
          <div className={styles.heroMetaCard}>
            <span>Всего товаров</span>
            <strong>{total}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>В продаже</span>
            <strong>{items.filter((item) => item.isActive).length}</strong>
          </div>
          <div className={styles.heroMetaCard}>
            <span>Требуют внимания</span>
            <strong>{items.filter((item) => (item.stock ?? 0) <= 10).length}</strong>
          </div>
        </div>
      </section>

      {loading ? <AppLoader label="Загружаем товары магазина..." /> : null}
      {error ? <ErrorMessage message={error} /> : null}

      <div className={styles.contentGrid}>
        <section className={styles.panel}>
          <div className={styles.panelHeader}>
            <div>
              <span className="badge-pill">
                {editingProduct ? 'Редактирование' : 'Новая карточка'}
              </span>
              <h2>{editingProduct ? 'Обновите карточку товара' : 'Добавьте новый товар'}</h2>
              <p>
                Новая карточка сразу попадёт в каталог вашего магазина и начнёт работать на витрине.
              </p>
            </div>
            <div className={styles.inlineActions}>
              <button
                type="button"
                className="action-secondary"
                onClick={() => openAIAssistant('generate')}
              >
                AI-черновик карточки
              </button>
            </div>
          </div>

          <ProductCardAIAssistant
            isOpen={aiAssistantOpen}
            defaultMode={aiAssistantMode}
            categoryName={categoryMap.get(formState.categoryId) ?? ''}
            formState={formState}
            busy={submitting}
            onApplyDraft={handleApplyAIDraft}
            onClose={() => setAIAssistantOpen(false)}
          />

          <form className={styles.form} onSubmit={handleSubmit}>
            <div className={styles.formGrid}>
              <label className={styles.field}>
                Название товара
                <input
                  value={formState.name}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, name: event.target.value }))
                  }
                  required
                />
              </label>
              <label className={styles.field}>
                Slug
                <input
                  value={formState.slug}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, slug: event.target.value }))
                  }
                />
              </label>
              <label className={styles.field}>
                Артикул
                <input
                  value={formState.sku}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, sku: event.target.value }))
                  }
                  required
                />
              </label>
              <label className={styles.field}>
                Категория
                <select
                  value={formState.categoryId}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, categoryId: event.target.value }))
                  }
                  required
                >
                  <option value="">Выберите категорию</option>
                  {categories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.field}>
                Цена
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={formState.price}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, price: event.target.value }))
                  }
                  required
                />
              </label>
              <label className={styles.field}>
                Валюта
                <input
                  value={formState.currency}
                  onChange={(event) =>
                    setFormState((current) => ({
                      ...current,
                      currency: event.target.value.toUpperCase(),
                    }))
                  }
                  maxLength={3}
                />
              </label>
              <label className={styles.field}>
                Остаток
                <input
                  type="number"
                  min="0"
                  value={formState.stockQty}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, stockQty: event.target.value }))
                  }
                  required
                />
              </label>
              <label className={styles.field}>
                Единица продажи
                <input
                  value={formState.unit}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, unit: event.target.value }))
                  }
                  placeholder="шт. / набор / кг"
                />
              </label>
              <label className={styles.field}>
                Бренд
                <input
                  value={formState.brand}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, brand: event.target.value }))
                  }
                />
              </label>
              <label className={styles.field}>
                URL обложки
                <input
                  value={formState.imageUrl}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, imageUrl: event.target.value }))
                  }
                  placeholder="Оставьте пустым для локальной обложки"
                />
              </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                <span className={styles.fieldLabelRow}>
                  <span>Описание</span>
                  <button
                    type="button"
                    className={styles.fieldAction}
                    onClick={() => openAIAssistant('improve')}
                  >
                    Улучшить текст
                  </button>
                </span>
                <textarea
                  value={formState.description}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, description: event.target.value }))
                  }
                />
              </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                Галерея изображений
                <textarea
                  value={formState.imagesText}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, imagesText: event.target.value }))
                  }
                  placeholder="По одному URL в строке"
                />
              </label>
              <label className={`${styles.field} ${styles.fullWidth}`}>
                JSON характеристик
                <textarea
                  value={formState.specsText}
                  onChange={(event) =>
                    setFormState((current) => ({ ...current, specsText: event.target.value }))
                  }
                />
              </label>
            </div>

            <p className={styles.helper}>
              Если вы не добавляете свои фото, маркетплейс покажет аккуратную локальную обложку для
              карточки.
            </p>

            <label className={styles.field}>
              <span>Видимость на витрине</span>
              <select
                value={formState.isActive ? 'visible' : 'hidden'}
                onChange={(event) =>
                  setFormState((current) => ({
                    ...current,
                    isActive: event.target.value === 'visible',
                  }))
                }
              >
                <option value="visible">Показывать в магазине</option>
                <option value="hidden">Скрыть с витрины</option>
              </select>
            </label>

            <div className={styles.inlineActions}>
              <button type="submit" className="action-primary" disabled={submitting}>
                {submitting ? 'Сохраняем...' : editingProduct ? 'Обновить товар' : 'Создать товар'}
              </button>
              <button
                type="button"
                className="action-secondary"
                onClick={resetForm}
                disabled={submitting}
              >
                Сбросить форму
              </button>
            </div>
          </form>
        </section>

        <section className={styles.panel}>
          <div className={`${styles.toolbar} ${styles.catalogToolbar}`}>
            <div className={styles.catalogToolbarHeading}>
              <span className="badge-pill">Каталог магазина</span>
              <h2>Текущие товары</h2>
              <p>
                {catalogRequested
                  ? `Найдено: ${total}`
                  : 'Выберите фильтры или нажмите «Применить», чтобы открыть товары.'}
              </p>
            </div>
            <form
              className={`${styles.toolbarFilters} ${styles.catalogToolbarFilters}`}
              onSubmit={(event) => {
                event.preventDefault()
                submitCatalogFilters()
              }}
            >
              <input
                value={searchDraft}
                onChange={(event) => setSearchDraft(event.target.value)}
                placeholder="Поиск по названию, артикулу или бренду"
              />
              <div className={styles.catalogToolbarControlGrid}>
                <select
                  value={categoryDraft}
                  onChange={(event) => setCategoryDraft(event.target.value)}
                >
                  <option value="">Все категории</option>
                  {sellerFilterCategories.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))}
                </select>
                <select
                  value={statusDraft}
                  onChange={(event) =>
                    setStatusDraft(event.target.value as 'all' | 'visible' | 'hidden')
                  }
                >
                  <option value="all">Все статусы</option>
                  <option value="visible">В продаже</option>
                  <option value="hidden">Скрытые</option>
                </select>
                <button type="submit" className={`action-secondary ${styles.catalogToolbarButton}`}>
                  Применить
                </button>
                <button
                  type="button"
                  className={`action-ghost ${styles.catalogToolbarButton}`}
                  onClick={resetFilters}
                >
                  Сбросить
                </button>
              </div>
            </form>
          </div>

          <div className={styles.list}>
            {!catalogRequested ? null : items.length === 0 && !loading ? (
              <div className="empty-state">
                <h2>Товары не найдены</h2>
                <p>Добавьте первую карточку или ослабьте фильтры каталога.</p>
              </div>
            ) : (
              items.map((product) => (
                <article
                  key={product.id}
                  className={`${styles.listCard} ${styles.productListCard}`}
                >
                  <img
                    className={`${styles.mediaThumb} ${styles.productThumb}`}
                    src={resolveProductImage(product)}
                    alt={product.title}
                    onError={(event) =>
                      swapImageToFallback(event.currentTarget, resolveProductImageFallback(product))
                    }
                  />
                  <div className={styles.productCardBody}>
                    <div className={styles.productCardTop}>
                      <div className={styles.productCardTitleBlock}>
                        <h3>{product.title}</h3>
                        <p className={styles.listMeta}>
                          {categoryMap.get(product.categoryId) ?? 'Категория'} • Артикул{' '}
                          {product.sku ?? '-'}
                        </p>
                      </div>
                      <div className={styles.productCardAside}>
                        <strong className={styles.productPrice}>
                          {formatCurrency(product.price, product.currency)}
                        </strong>
                        {product.unit ? (
                          <span className={styles.productUnit}>
                            {formatUnitLabel(product.unit)}
                          </span>
                        ) : null}
                        <div className={styles.badgeRow}>
                          <span className={product.isActive ? styles.badge : styles.badgeDanger}>
                            {product.isActive ? 'В продаже' : 'Скрыт'}
                          </span>
                          <span
                            className={
                              (product.stock ?? 0) <= 10 ? styles.badgeWarn : styles.badgeMuted
                            }
                          >
                            Остаток: {product.stock ?? 0}
                          </span>
                        </div>
                      </div>
                    </div>

                    {product.description ? (
                      <p className={styles.productDescription}>{product.description}</p>
                    ) : null}

                    <div className={styles.productCardFooter}>
                      <div className={styles.inlineActions}>
                        <button
                          type="button"
                          className={styles.inlineButton}
                          onClick={() => handleEdit(product)}
                        >
                          Изменить
                        </button>
                        <button
                          type="button"
                          className={styles.inlineButton}
                          onClick={() => handleToggleActive(product)}
                          disabled={submitting}
                        >
                          {product.isActive ? 'Скрыть' : 'Вернуть'}
                        </button>
                        <button
                          type="button"
                          className={`${styles.inlineButton} ${styles.inlineButtonDanger}`}
                          onClick={() => handleArchive(product)}
                          disabled={submitting}
                        >
                          Убрать с витрины
                        </button>
                      </div>

                      <form
                        className={styles.stockForm}
                        onSubmit={(event) => {
                          event.preventDefault()
                          void handleStockSave(product)
                        }}
                      >
                        <label className={styles.stockField}>
                          <span>Остаток</span>
                          <input
                            className={styles.stockInput}
                            type="number"
                            min="0"
                            value={stockDrafts[product.id] ?? String(product.stock ?? 0)}
                            onChange={(event) =>
                              setStockDrafts((current) => ({
                                ...current,
                                [product.id]: event.target.value,
                              }))
                            }
                          />
                        </label>
                        <button type="submit" className="action-secondary" disabled={submitting}>
                          Сохранить
                        </button>
                      </form>
                    </div>
                  </div>
                </article>
              ))
            )}
          </div>
        </section>
      </div>
    </div>
  )
}
