import { describe, expect, it } from 'vitest'

import { filtersToSearchParams, searchParamsToFilters } from '@/utils/query'

describe('query utils', () => {
  it('converts search params to backend-compatible filters', () => {
    const params = new URLSearchParams('q=laptop&category_id=cat-1&min_price=100&max_price=400&in_stock=true&sort=price_desc&page=2&limit=24')
    const filters = searchParamsToFilters(params)

    expect(filters).toEqual({
      q: 'laptop',
      category_id: 'cat-1',
      min_price: 100,
      max_price: 400,
      in_stock: true,
      sort: 'price_desc',
      page: 2,
      limit: 24,
    })
  })

  it('converts filters back to URL params', () => {
    const params = filtersToSearchParams({
      q: 'phone',
      category_id: 'cat-2',
      min_price: 50,
      max_price: 500,
      in_stock: true,
      sort: 'new',
      page: 1,
      limit: 12,
    })

    expect(params.get('q')).toBe('phone')
    expect(params.get('category_id')).toBe('cat-2')
    expect(params.get('min_price')).toBe('50')
    expect(params.get('max_price')).toBe('500')
    expect(params.get('in_stock')).toBe('true')
    expect(params.get('sort')).toBe('new')
    expect(params.get('page')).toBe('1')
    expect(params.get('limit')).toBe('12')
  })
})
