import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import type { FilterValues } from '@/components/catalog/FilterPanel'
import { FilterPanel } from '@/components/catalog/FilterPanel'
import { ProductGrid } from '@/components/catalog/ProductGrid'
import { SearchBar } from '@/components/catalog/SearchBar'
import { ErrorMessage } from '@/components/common/ErrorMessage'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { fetchCategoriesThunk } from '@/store/slices/categoriesSlice'
import { fetchProductsThunk } from '@/store/slices/productsSlice'
import { filtersToSearchParams, searchParamsToFilters } from '@/utils/query'

import styles from '@/pages/HomePage.module.scss'

const CATEGORY_ICONS: Record<string, string> = {
  stroymaterialy: 'SM',
  cement: 'CM',
  pilomaterialy: 'WD',
  oak: 'OK',
  aspen: 'AS',
  birch: 'BR',
  krepezh: 'FX',
  nails: 'NL',
}

const getCategoryIcon = (slug?: string): string => {
  if (!slug) return 'CT'
  return CATEGORY_ICONS[slug] || 'CT'
}

export const HomePage = () => {
  const dispatch = useAppDispatch()
  const [searchParams, setSearchParams] = useSearchParams()
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false)

  const categories = useAppSelector((state) => state.categories.items)
  const productsState = useAppSelector((state) => state.products)

  const filters = useMemo(() => searchParamsToFilters(searchParams), [searchParams])

  useEffect(() => {
    dispatch(fetchCategoriesThunk())
  }, [dispatch])

  useEffect(() => {
    dispatch(fetchProductsThunk({ limit: 12, sort: 'new', ...filters }))
  }, [dispatch, filters])

  const applyFilters = (values: FilterValues) => {
    const nextParams = filtersToSearchParams({
      ...filters,
      ...values,
      page: 1,
      limit: 12,
    })
    setSearchParams(nextParams)
  }

  const handleSearch = (query: string) => {
    const nextParams = filtersToSearchParams({
      ...filters,
      q: query || undefined,
      page: 1,
      limit: 12,
    })
    setSearchParams(nextParams)
  }

  const handleCategoryClick = (categoryId?: string) => {
    const nextParams = filtersToSearchParams({
      ...filters,
      category_id: categoryId,
      page: 1,
      limit: 12,
    })
    setSearchParams(nextParams)
  }

  const handleReset = () => {
    setSearchParams(new URLSearchParams())
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
          onClick={() => setSearchParams(filtersToSearchParams({ ...filters, page: i, limit: 12 }))}
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

  return (
    <div className={styles.page}>
      <section className={styles.hero}>
        <div className={styles.heroContent}>
          <span className={styles.eyebrow}>Marketplace</span>
          <h1 className={styles.heroTitle}>Find the right product faster</h1>
          <p className={styles.heroSub}>Search with suggestions, popular queries, price filters, stock status, and category shortcuts.</p>
          <div className={styles.searchWrapper}>
            <SearchBar key={`hero-${filters.q ?? ''}`} initialValue={filters.q ?? ''} onSearch={handleSearch} />
          </div>
        </div>
      </section>

      {categories.length > 0 && (
        <section className={styles.categoriesSection}>
          <h2 className={styles.sectionTitle}>Categories</h2>
          <div className={styles.categoriesGrid}>
            <div
              className={`${styles.categoryCard} ${!filters.category_id ? styles.categoryCardActive : ''}`}
              onClick={() => handleCategoryClick(undefined)}
            >
              <span className={styles.categoryIcon}>ALL</span>
              <span className={styles.categoryName}>All products</span>
            </div>
            {categories.map((category) => (
              <div
                key={category.id}
                className={`${styles.categoryCard} ${filters.category_id === category.id ? styles.categoryCardActive : ''}`}
                onClick={() => handleCategoryClick(category.id)}
              >
                <span className={styles.categoryIcon}>{getCategoryIcon(category.slug)}</span>
                <span className={styles.categoryName}>{category.name}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      <div className={styles.heading}>
        <div>
          <p className={styles.eyebrow}>Catalog</p>
          <h1>Products</h1>
          <p className={styles.sub}>Use backend-compatible filters and sorting.</p>
        </div>
        <button
          type="button"
          className={styles.mobileFilterButton}
          onClick={() => setMobileFiltersOpen(true)}
        >
          Filters
        </button>
      </div>

      <div className={styles.searchRow}>
        <SearchBar key={`row-${filters.q ?? ''}`} initialValue={filters.q ?? ''} onSearch={handleSearch} />
      </div>

      <div className={styles.layout}>
        <aside>
          <FilterPanel
            categories={categories}
            values={filterValues}
            onChange={applyFilters}
            onReset={handleReset}
            mobileOpen={mobileFiltersOpen}
            onCloseMobile={() => setMobileFiltersOpen(false)}
          />
        </aside>

        <section className={styles.catalog}>
          {productsState.status === 'loading' ? (
            <div className={styles.loading}>
              <div className={styles.loader} />
            </div>
          ) : null}
          {productsState.error ? <ErrorMessage message={productsState.error} /> : null}
          <ProductGrid products={productsState.items} />

          {productsState.totalPages > 1 && (
            <div className={styles.pagination}>
              <button
                type="button"
                className={styles.paginationBtn}
                disabled={currentPage <= 1}
                onClick={() => setSearchParams(filtersToSearchParams({ ...filters, page: prevPage, limit: 12 }))}
              >
                Prev
              </button>
              {renderPaginationButtons()}
              <span className={styles.paginationInfo}>
                {currentPage} / {productsState.totalPages}
              </span>
              <button
                type="button"
                className={styles.paginationBtn}
                disabled={currentPage >= productsState.totalPages}
                onClick={() => setSearchParams(filtersToSearchParams({ ...filters, page: nextPage, limit: 12 }))}
              >
                Next
              </button>
            </div>
          )}
        </section>
      </div>
    </div>
  )
}
