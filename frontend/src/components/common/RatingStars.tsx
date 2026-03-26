import { useId } from 'react'

import styles from '@/components/common/RatingStars.module.scss'

interface RatingStarsProps {
  rating: number
  max?: number
  size?: 'sm' | 'md' | 'lg'
}

const STAR_PATH =
  'M12 2.35l2.98 6.05 6.67.97-4.82 4.7 1.14 6.64L12 17.48 6.03 20.71l1.14-6.64-4.82-4.7 6.67-.97L12 2.35Z'

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

export const RatingStars = ({ rating, max = 5, size = 'md' }: RatingStarsProps) => {
  const gradientPrefix = useId()
  const normalizedRating = clamp(Number.isFinite(rating) ? rating : 0, 0, max)

  return (
    <div
      className={`${styles.stars} ${styles[`size${size.charAt(0).toUpperCase()}${size.slice(1)}`]}`}
      role="img"
      aria-label={`${normalizedRating.toFixed(1)} out of ${max} stars`}
    >
      {Array.from({ length: max }, (_, index) => {
        const fill = clamp(normalizedRating - index, 0, 1)
        const offset = `${Math.round(fill * 100)}%`
        const gradientId = `${gradientPrefix}-star-${index}`

        return (
          <svg key={gradientId} viewBox="0 0 24 24" aria-hidden="true" className={styles.star}>
            <defs>
              <linearGradient id={gradientId} x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset={offset} stopColor="currentColor" />
                <stop offset={offset} stopColor="currentColor" stopOpacity="0.18" />
                <stop offset="100%" stopColor="currentColor" stopOpacity="0.18" />
              </linearGradient>
            </defs>
            <path d={STAR_PATH} fill={`url(#${gradientId})`} stroke="currentColor" strokeWidth="1.35" strokeLinejoin="round" />
          </svg>
        )
      })}
    </div>
  )
}
