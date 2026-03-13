import { configureStore } from '@reduxjs/toolkit'

import authReducer from '@/store/slices/authSlice'
import cartReducer from '@/store/slices/cartSlice'
import categoriesReducer from '@/store/slices/categoriesSlice'
import favoritesReducer from '@/store/slices/favoritesSlice'
import ordersReducer from '@/store/slices/ordersSlice'
import placesReducer from '@/store/slices/placesSlice'
import productsReducer from '@/store/slices/productsSlice'
import recommendationsReducer from '@/store/slices/recommendationsSlice'
import userReducer from '@/store/slices/userSlice'

export const store = configureStore({
  reducer: {
    auth: authReducer,
    user: userReducer,
    cart: cartReducer,
    orders: ordersReducer,
    products: productsReducer,
    categories: categoriesReducer,
    favorites: favoritesReducer,
    places: placesReducer,
    recommendations: recommendationsReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
