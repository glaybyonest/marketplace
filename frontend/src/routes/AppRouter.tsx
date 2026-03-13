import { Navigate, Route, Routes } from 'react-router-dom'

import { ProtectedRoute } from '@/routes/ProtectedRoute'
import { AccountPage } from '@/pages/AccountPage'
import { CartPage } from '@/pages/CartPage'
import { CheckoutPage } from '@/pages/CheckoutPage'
import { FavoritesPage } from '@/pages/FavoritesPage'
import { HomePage } from '@/pages/HomePage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { OrdersPage } from '@/pages/OrdersPage'
import { PlacesPage } from '@/pages/PlacesPage'
import { ProductPage } from '@/pages/ProductPage'
import { RegisterPage } from '@/pages/RegisterPage'

export const AppRouter = () => (
  <Routes>
    <Route path="/" element={<HomePage />} />
    <Route path="/products/:id" element={<ProductPage />} />
    <Route path="/login" element={<LoginPage />} />
    <Route path="/register" element={<RegisterPage />} />

    <Route element={<ProtectedRoute />}>
      <Route path="/account" element={<AccountPage />} />
      <Route path="/cart" element={<CartPage />} />
      <Route path="/checkout" element={<CheckoutPage />} />
      <Route path="/favorites" element={<FavoritesPage />} />
      <Route path="/account/orders" element={<OrdersPage />} />
      <Route path="/account/places" element={<PlacesPage />} />
    </Route>

    <Route path="/seller/*" element={<Navigate to="/" replace />} />
    <Route path="/admin/*" element={<Navigate to="/" replace />} />

    <Route path="/logout" element={<Navigate to="/login" replace />} />
    <Route path="*" element={<NotFoundPage />} />
  </Routes>
)
