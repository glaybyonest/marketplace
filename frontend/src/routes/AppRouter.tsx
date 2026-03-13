import { Navigate, Route, Routes } from 'react-router-dom'

import { ProtectedRoute } from '@/routes/ProtectedRoute'
import { AccountPage } from '@/pages/AccountPage'
import { AdminCategoriesPage } from '@/pages/AdminCategoriesPage'
import { AdminDashboardPage } from '@/pages/AdminDashboardPage'
import { AdminProductsPage } from '@/pages/AdminProductsPage'
import { CartPage } from '@/pages/CartPage'
import { CheckoutPage } from '@/pages/CheckoutPage'
import { FavoritesPage } from '@/pages/FavoritesPage'
import { ForgotPasswordPage } from '@/pages/ForgotPasswordPage'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { OrdersPage } from '@/pages/OrdersPage'
import { PlacesPage } from '@/pages/PlacesPage'
import { ProductPage } from '@/pages/ProductPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { ResetPasswordPage } from '@/pages/ResetPasswordPage'
import { SessionsPage } from '@/pages/SessionsPage'
import { VerifyEmailPage } from '@/pages/VerifyEmailPage'

export const AppRouter = () => (
  <Routes>
    <Route path="/" element={<HomePage />} />
    <Route path="/products/:id" element={<ProductPage />} />
    <Route path="/login" element={<LoginPage />} />
    <Route path="/register" element={<RegisterPage />} />
    <Route path="/verify-email" element={<VerifyEmailPage />} />
    <Route path="/forgot-password" element={<ForgotPasswordPage />} />
    <Route path="/reset-password" element={<ResetPasswordPage />} />

    <Route element={<ProtectedRoute />}>
      <Route path="/account" element={<AccountPage />} />
      <Route path="/cart" element={<CartPage />} />
      <Route path="/checkout" element={<CheckoutPage />} />
      <Route path="/favorites" element={<FavoritesPage />} />
      <Route path="/account/orders" element={<OrdersPage />} />
      <Route path="/account/places" element={<PlacesPage />} />
      <Route path="/account/sessions" element={<SessionsPage />} />
    </Route>

    <Route element={<ProtectedRoute requiredRole="admin" />}>
      <Route path="/admin" element={<AdminDashboardPage />} />
      <Route path="/admin/categories" element={<AdminCategoriesPage />} />
      <Route path="/admin/products" element={<AdminProductsPage />} />
    </Route>

    <Route path="/seller/*" element={<Navigate to="/" replace />} />

    <Route path="/logout" element={<Navigate to="/login" replace />} />
    <Route path="*" element={<NotFoundPage />} />
  </Routes>
)
