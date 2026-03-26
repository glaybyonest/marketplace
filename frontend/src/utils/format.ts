export const formatCurrency = (value: number, currency = 'RUB') =>
  new Intl.NumberFormat('ru-RU', {
    style: 'currency',
    currency,
    maximumFractionDigits: 0,
  }).format(value)

export const formatDate = (isoDate: string) =>
  new Intl.DateTimeFormat('ru-RU', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(isoDate))

export const formatReviewCount = (count: number) => {
  const absCount = Math.abs(count)
  const lastTwo = absCount % 100
  const last = absCount % 10

  if (lastTwo >= 11 && lastTwo <= 14) {
    return `${count} отзывов`
  }
  if (last === 1) {
    return `${count} отзыв`
  }
  if (last >= 2 && last <= 4) {
    return `${count} отзыва`
  }
  return `${count} отзывов`
}

const UNIT_LABELS: Record<string, string> = {
  piece: 'шт.',
  pieces: 'шт.',
  item: 'шт.',
  items: 'шт.',
  unit: 'шт.',
  units: 'шт.',
  pc: 'шт.',
  pcs: 'шт.',
}

export const formatUnitLabel = (value?: string | null) => {
  const normalized = value?.trim()
  if (!normalized) {
    return ''
  }

  return UNIT_LABELS[normalized.toLowerCase()] ?? normalized
}
