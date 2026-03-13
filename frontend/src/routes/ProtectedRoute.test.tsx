import { configureStore } from '@reduxjs/toolkit'
import { render, screen } from '@testing-library/react'
import { Provider } from 'react-redux'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import { ProtectedRoute } from '@/routes/ProtectedRoute'
import authReducer from '@/store/slices/authSlice'
import categoriesReducer from '@/store/slices/categoriesSlice'
import favoritesReducer from '@/store/slices/favoritesSlice'
import placesReducer from '@/store/slices/placesSlice'
import productsReducer from '@/store/slices/productsSlice'
import recommendationsReducer from '@/store/slices/recommendationsSlice'
import userReducer from '@/store/slices/userSlice'
import type { UserRole } from '@/types/domain'

const createTestStore = (isAuthenticated: boolean, role: UserRole = 'customer') => {
  const authState: ReturnType<typeof authReducer> = {
    token: isAuthenticated ? 'token' : null,
    refreshToken: isAuthenticated ? 'refresh' : null,
    user: isAuthenticated
      ? {
          id: '1',
          name: 'User',
          fullName: 'User',
          email: 'user@test.local',
          isEmailVerified: true,
          role,
        }
      : null,
    isAuthenticated,
    status: 'idle',
    error: null,
    errorCode: null,
    notice: null,
    requiresEmailVerification: false,
    sessionBootstrapped: true,
  }

  return configureStore({
    reducer: {
      auth: authReducer,
      user: userReducer,
      products: productsReducer,
      categories: categoriesReducer,
      favorites: favoritesReducer,
      places: placesReducer,
      recommendations: recommendationsReducer,
    },
    preloadedState: {
      auth: authState,
    },
  })
}

const renderRoute = (isAuthenticated: boolean, requiredRole?: Exclude<UserRole, 'guest'>, role: UserRole = 'customer') => {
  const store = createTestStore(isAuthenticated, role)
  return render(
    <Provider store={store}>
      <MemoryRouter initialEntries={['/secure']}>
        <Routes>
          <Route element={<ProtectedRoute requiredRole={requiredRole} />}>
            <Route path="/secure" element={<div>Secure page</div>} />
          </Route>
          <Route path="/login" element={<div>Login page</div>} />
          <Route path="/" element={<div>Home page</div>} />
        </Routes>
      </MemoryRouter>
    </Provider>,
  )
}

describe('ProtectedRoute', () => {
  it('redirects guest to login page', () => {
    renderRoute(false)
    expect(screen.getByText('Login page')).toBeInTheDocument()
  })

  it('allows access for authenticated user', () => {
    renderRoute(true)
    expect(screen.getByText('Secure page')).toBeInTheDocument()
  })

  it('redirects non-admin user away from admin route', () => {
    renderRoute(true, 'admin', 'customer')
    expect(screen.getByText('Home page')).toBeInTheDocument()
  })

  it('allows admin user to access admin route', () => {
    renderRoute(true, 'admin', 'admin')
    expect(screen.getByText('Secure page')).toBeInTheDocument()
  })
})
