import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { ErrorBoundary } from '@/components/common/ErrorBoundary'
import { AppLayout } from '@/components/layout/AppLayout'
import { isCookieAuthMode } from '@/config/auth'
import { AUTH_UNAUTHORIZED_EVENT } from '@/services/apiClient'
import { AppRouter } from '@/routes/AppRouter'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { forceLogout, setUser } from '@/store/slices/authSlice'
import { clearFavorites, fetchFavoritesThunk } from '@/store/slices/favoritesSlice'
import { fetchProfileThunk } from '@/store/slices/userSlice'

const App = () => {
  const navigate = useNavigate()
  const dispatch = useAppDispatch()
  const auth = useAppSelector((state) => state.auth)

  useEffect(() => {
    if (auth.sessionBootstrapped) {
      return
    }
    if (!isCookieAuthMode && !auth.token) {
      return
    }

    dispatch(fetchProfileThunk()).then((result) => {
      if (fetchProfileThunk.fulfilled.match(result)) {
        dispatch(setUser(result.payload))
      } else {
        dispatch(forceLogout())
      }
    })
  }, [auth.sessionBootstrapped, auth.token, dispatch])

  useEffect(() => {
    if (!auth.isAuthenticated) {
      return
    }

    dispatch(fetchFavoritesThunk({ page: 1, limit: 50 }))
  }, [auth.isAuthenticated, dispatch])

  useEffect(() => {
    const handleUnauthorized = () => {
      dispatch(forceLogout())
      dispatch(clearFavorites())
      navigate('/login')
    }

    window.addEventListener(AUTH_UNAUTHORIZED_EVENT, handleUnauthorized)
    return () => {
      window.removeEventListener(AUTH_UNAUTHORIZED_EVENT, handleUnauthorized)
    }
  }, [dispatch, navigate])

  return (
    <ErrorBoundary>
      <AppLayout>
        <AppRouter />
      </AppLayout>
    </ErrorBoundary>
  )
}

export default App
