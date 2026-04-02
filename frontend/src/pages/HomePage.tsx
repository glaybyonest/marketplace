import { startTransition, useEffect, useMemo, useRef, useState } from 'react'
import type { CSSProperties } from 'react'
import { useSearchParams } from 'react-router-dom'

import type { FilterValues } from '@/components/catalog/FilterPanel'
import { FilterPanel } from '@/components/catalog/FilterPanel'
import { ProductGrid } from '@/components/catalog/ProductGrid'
import { CategoryIcon } from '@/components/common/CategoryIcon'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchCategoriesThunk } from '@/store/slices/categoriesSlice'
import { fetchProductsThunk } from '@/store/slices/productsSlice'
import { fetchRecommendationsThunk } from '@/store/slices/recommendationsSlice'
import { defaultCategoryVisual, resolveCategoryVisual } from '@/utils/categoryIcons'
import { filtersToSearchParams, searchParamsToFilters } from '@/utils/query'

import styles from '@/pages/HomePage.module.scss'

const CATALOG_PAGE_SIZE = 20

export const HomePage = () => {
  const dispatch = useAppDispatch()
  const [searchParams, setSearchParams] = useSearchParams()
  const [filtersOpen, setFiltersOpen] = useState(false)
  const filtersControlRef = useRef<HTMLDivElement | null>(null)
  const catalogSectionRef = useRef<HTMLDivElement | null>(null)

  const auth = useAppSelector((state) => state.auth)
  const categories = useAppSelector((state) => state.categories.items)
  const productsState = useAppSelector((state) => state.products)
  const recommendations = useAppSelector((state) => state.recommendations.items)

  const filters = useMemo(() => searchParamsToFilters(searchParams), [searchParams])
  const hasActiveFilters = Boolean(
    filters.q || filters.category_id || filters.min_price || filters.max_price || filters.in_stock || (filters.sort && filters.sort !== 'new'),
  )

  useEffect(() => {
    dispatch(fetchCategoriesThunk())
  }, [dispatch])

  useEffect(() => {
    dispatch(fetchProductsThunk({ ...filters, sort: filters.sort ?? 'new', limit: CATALOG_PAGE_SIZE }))
  }, [dispatch, filters])

  useEffect(() => {
    if (auth.isAuthenticated) {
      dispatch(fetchRecommendationsThunk(8))
    }
  }, [auth.isAuthenticated, dispatch])

  useEffect(() => {
    if (!filtersOpen) {
      return
    }

    const handlePointerDown = (event: MouseEvent) => {
      if (filtersControlRef.current && !filtersControlRef.current.contains(event.target as Node)) {
        setFiltersOpen(false)
      }
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setFiltersOpen(false)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [filtersOpen])

  const applyFilters = (values: FilterValues) => {
    const nextParams = filtersToSearchParams({
      ...filters,
      ...values,
      page: 1,
      limit: CATALOG_PAGE_SIZE,
    })
    startTransition(() => setSearchParams(nextParams))
  }

  const handleCategoryClick = (categoryId?: string) => {
    const nextParams = filtersToSearchParams({
      ...filters,
      category_id: categoryId,
      page: 1,
      limit: CATALOG_PAGE_SIZE,
    })
    startTransition(() => setSearchParams(nextParams))
  }

  const handleReset = () => {
    startTransition(() => setSearchParams(new URLSearchParams()))
  }

  const scrollToCatalogSection = () => {
    window.requestAnimationFrame(() => {
      catalogSectionRef.current?.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
      })
    })
  }

  const handleOpenAllProducts = () => {
    handleReset()
    scrollToCatalogSection()
  }

  const handlePageChange = (page: number) => {
    startTransition(() => setSearchParams(filtersToSearchParams({ ...filters, page, limit: CATALOG_PAGE_SIZE })))
    scrollToCatalogSection()
  }

  const currentPage = filters.page ?? 1
  const nextPage = Math.min(productsState.totalPages, currentPage + 1)
  const prevPage = Math.max(1, currentPage - 1)

  const renderPaginationButtons = () => {
    const buttons = []
    const totalPages = productsState.totalPages
    const maxVisible = 5
    let start = Math.max(1, currentPage - Math.floor(maxVisible / 2))
    const end = Math.min(totalPages, start + maxVisible - 1)

    if (end - start < maxVisible - 1) {
      start = Math.max(1, end - maxVisible + 1)
    }

    for (let i = start; i <= end; i++) {
      buttons.push(
        <button
          key={i}
          type="button"
          className={`${styles.paginationBtn} ${i === currentPage ? styles.paginationBtnActive : ''}`}
          onClick={() => handlePageChange(i)}
        >
          {i}
        </button>,
      )
    }

    return buttons
  }

  const filterValues: FilterValues = {
    category_id: filters.category_id,
    min_price: filters.min_price,
    max_price: filters.max_price,
    in_stock: filters.in_stock,
    sort: (filters.sort as FilterValues['sort']) ?? 'new',
  }

  const shelves = useMemo(() => {
    const items = productsState.items
    const fallbackRecommended = items.slice(4, 8)

    return {
      fast: items.slice(0, 4),
      recommended: (recommendations.length > 0 ? recommendations : fallbackRecommended).slice(0, 4),
    }
  }, [productsState.items, recommendations])

  const selectedCategory = categories.find((category) => category.id === filters.category_id)

  return (
    <div className={styles.page}>
      {!hasActiveFilters ? (
        <>
          <section className={styles.shelfSection}>
            <div className={styles.sectionHeading}>
              <div>
                <span className={styles.sectionKicker}>Под рукой</span>
                <h2>Категории каталога</h2>
              </div>
              <button type="button" className="action-secondary" onClick={handleOpenAllProducts}>
                Весь каталог
              </button>
            </div>

            <div className={styles.categoriesGrid}>
              <button
                type="button"
                className={`${styles.categoryCard} ${!filters.category_id ? styles.categoryCardActive : ''}`}
                style={{ '--category-accent': defaultCategoryVisual.accent } as CSSProperties}
                onClick={() => handleCategoryClick(undefined)}
              >
                <CategoryIcon iconKey={defaultCategoryVisual.iconKey} accent={defaultCategoryVisual.accent} className={styles.categoryIcon} />
                <span className={styles.categoryName}>Все товары</span>
              </button>
              {categories.map((category) => {
                const visual = resolveCategoryVisual({ slug: category.slug, name: category.name })
                return (
                  <button
                    key={category.id}
                    type="button"
                    className={`${styles.categoryCard} ${filters.category_id === category.id ? styles.categoryCardActive : ''}`}
                    style={{ '--category-accent': visual.accent } as CSSProperties}
                    onClick={() => handleCategoryClick(category.id)}
                  >
                    <CategoryIcon iconKey={visual.iconKey} accent={visual.accent} className={styles.categoryIcon} />
                    <span className={styles.categoryName}>{category.name}</span>
                  </button>
                )
              })}
            </div>
          </section>

          {shelves.fast.length > 0 || shelves.recommended.length > 0 ? (
            <section className={styles.shelfSplit}>
              <div className={styles.shelfPanel}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span className={styles.sectionKicker}>Популярное</span>
                    <h2>Хиты недели</h2>
                  </div>
                </div>
                <ProductGrid products={shelves.fast} />
              </div>

              <div className={styles.shelfPanel}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span className={styles.sectionKicker}>Персонально</span>
                    <h2>Рекомендуем вам</h2>
                  </div>
                </div>
                <ProductGrid products={shelves.recommended} />
              </div>
            </section>
          ) : null}

        </>
      ) : null}

      <div className={styles.heading} ref={catalogSectionRef}>
        <div>
          <span className={styles.sectionKicker}>Каталог</span>
          <h2>{selectedCategory?.name ?? (filters.q ? `Результаты по запросу «${filters.q}»` : 'Все товары')}</h2>
        </div>
        <div className={styles.filtersControl} ref={filtersControlRef}>
          <button
            type="button"
            className={`action-secondary ${styles.filterButton} ${filtersOpen ? styles.filterButtonActive : ''}`}
            onClick={() => setFiltersOpen((value) => !value)}
            aria-expanded={filtersOpen}
            aria-controls="catalog-filters"
          >
            {filtersOpen ? 'Скрыть фильтры' : 'Фильтры'}
          </button>
          <div className={styles.filtersDropdown}>
            <FilterPanel
              categories={categories}
              values={filterValues}
              onChange={applyFilters}
              onReset={handleReset}
              open={filtersOpen}
              onClose={() => setFiltersOpen(false)}
              panelId="catalog-filters"
            />
          </div>
        </div>
      </div>

      <div className={styles.layout}>
        <section className={styles.catalog}>
          {productsState.status === 'loading' ? (
            <div className={styles.loading}>
              <div className={styles.loader} />
            </div>
          ) : null}
          {productsState.error ? <ErrorMessage message={productsState.error} /> : null}
          <ProductGrid products={productsState.items} />

          {productsState.totalPages > 1 ? (
            <div className={styles.pagination}>
              <button
                type="button"
                className={styles.paginationBtn}
                disabled={currentPage <= 1}
                onClick={() => handlePageChange(prevPage)}
              >
                Назад
              </button>
              {renderPaginationButtons()}
              <span className={styles.paginationInfo}>
                {currentPage} / {productsState.totalPages}
              </span>
              <button
                type="button"
                className={styles.paginationBtn}
                disabled={currentPage >= productsState.totalPages}
                onClick={() => handlePageChange(nextPage)}
              >
                Дальше
              </button>
            </div>
          ) : null}
        </section>
      </div>
    </div>
  )
}
