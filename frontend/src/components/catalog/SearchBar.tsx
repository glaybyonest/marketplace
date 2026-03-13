import { useEffect, useRef, useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'

import { productService } from '@/services/productService'
import type { PopularSearch, SearchSuggestion } from '@/types/domain'

import styles from '@/components/catalog/SearchBar.module.scss'

interface SearchBarProps {
  initialValue?: string
  onSearch: (value: string) => void
}

const SUGGESTION_KIND_LABELS: Record<string, string> = {
  product: 'Product',
  brand: 'Brand',
  category: 'Category',
  query: 'Search',
}

export const SearchBar = ({ initialValue = '', onSearch }: SearchBarProps) => {
  const [value, setValue] = useState(initialValue)
  const [focused, setFocused] = useState(false)
  const [loading, setLoading] = useState(false)
  const [suggestions, setSuggestions] = useState<SearchSuggestion[]>([])
  const [popularSearches, setPopularSearches] = useState<PopularSearch[]>([])
  const inputRef = useRef<HTMLInputElement>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!focused) {
      return
    }

    let cancelled = false

    if (value.trim().length >= 2) {
      const timer = window.setTimeout(async () => {
        if (cancelled) {
          return
        }
        setLoading(true)
        try {
          const nextSuggestions = await productService.getSearchSuggestions(value.trim(), 8)
          if (!cancelled) {
            setSuggestions(nextSuggestions)
          }
        } finally {
          if (!cancelled) {
            setLoading(false)
          }
        }
      }, 200)

      return () => {
        cancelled = true
        window.clearTimeout(timer)
      }
    }

    if (popularSearches.length === 0) {
      const timer = window.setTimeout(() => {
        if (cancelled) {
          return
        }
        setLoading(true)
        productService
          .getPopularSearches(6)
          .then((items) => {
            if (!cancelled) {
              setPopularSearches(items)
            }
          })
          .finally(() => {
            if (!cancelled) {
              setLoading(false)
            }
          })
      }, 0)

      return () => {
        cancelled = true
        window.clearTimeout(timer)
      }
    }

    return () => {
      cancelled = true
    }
  }, [focused, popularSearches.length, value])

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (!wrapperRef.current?.contains(event.target as Node)) {
        setFocused(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const runSearch = (nextValue: string) => {
    const normalized = nextValue.trim()
    setValue(normalized)
    setFocused(false)
    onSearch(normalized)
  }

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    runSearch(value)
  }

  const handleClear = () => {
    setValue('')
    setSuggestions([])
    inputRef.current?.focus()
    onSearch('')
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      setFocused(false)
      handleClear()
    }
  }

  const showDropdown = focused && (value.trim().length >= 2 || popularSearches.length > 0 || loading)

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <div className={styles.inputWrapper} ref={wrapperRef}>
        <input
          ref={inputRef}
          className={styles.input}
          type="search"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onFocus={() => setFocused(true)}
          onKeyDown={handleKeyDown}
          placeholder="Search products, brands, and categories..."
          aria-label="Search products"
        />
        <svg
          className={styles.searchIcon}
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
        <button
          type="button"
          className={`${styles.clearButton} ${value ? styles.clearButtonVisible : ''}`}
          onClick={handleClear}
          aria-label="Clear search"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>

        {showDropdown ? (
          <div className={styles.dropdown}>
            {value.trim().length >= 2 ? (
              <>
                <div className={styles.dropdownTitle}>Suggestions</div>
                {loading ? <p className={styles.dropdownState}>Loading suggestions...</p> : null}
                {!loading && suggestions.length === 0 ? (
                  <p className={styles.dropdownState}>No suggestions yet. Press Enter to search anyway.</p>
                ) : null}
                {suggestions.map((item) => (
                  <button
                    key={`${item.kind}-${item.text}`}
                    type="button"
                    className={styles.dropdownItem}
                    onClick={() => runSearch(item.text)}
                  >
                    <span>{item.text}</span>
                    <small>{SUGGESTION_KIND_LABELS[item.kind] ?? 'Suggestion'}</small>
                  </button>
                ))}
              </>
            ) : (
              <>
                <div className={styles.dropdownTitle}>Popular searches</div>
                {loading ? <p className={styles.dropdownState}>Loading popular searches...</p> : null}
                {!loading ? (
                  <div className={styles.popularList}>
                    {popularSearches.map((item) => (
                      <button
                        key={item.query}
                        type="button"
                        className={styles.popularItem}
                        onClick={() => runSearch(item.query)}
                      >
                        <span>{item.query}</span>
                        {item.searchCount > 0 ? <small>{item.searchCount}</small> : null}
                      </button>
                    ))}
                  </div>
                ) : null}
              </>
            )}
          </div>
        ) : null}
      </div>
      <button className={styles.button} type="submit">
        <svg className={styles.buttonIcon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
        <span className={styles.buttonText}>Search</span>
      </button>
    </form>
  )
}
