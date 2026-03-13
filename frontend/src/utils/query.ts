import type { ProductFilters } from '@/types/api'

const numberFields = new Set(['page', 'limit', 'min_price', 'max_price'])
const booleanFields = new Set(['in_stock'])

export const searchParamsToFilters = (searchParams: URLSearchParams): ProductFilters => {
  const filters: ProductFilters = {}

  searchParams.forEach((value, key) => {
    if (!value) {
      return
    }

    if (numberFields.has(key)) {
      const parsed = Number(value)
      if (!Number.isNaN(parsed)) {
        ;(filters as Record<string, number>)[key] = parsed
      }
      return
    }

    if (booleanFields.has(key)) {
      ;(filters as Record<string, boolean>)[key] = value === 'true'
      return
    }

    ;(filters as Record<string, string>)[key] = value
  })

  return filters
}

export const filtersToSearchParams = (filters: ProductFilters): URLSearchParams => {
  const params = new URLSearchParams()

  Object.entries(filters).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') {
      return
    }

    params.set(key, String(value))
  })

  return params
}
