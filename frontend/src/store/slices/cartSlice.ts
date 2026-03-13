import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'

import { cartService } from '@/services/cartService'
import { logoutThunk } from '@/store/slices/authSlice'
import { checkoutThunk } from '@/store/slices/ordersSlice'
import type { Cart } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'

interface CartState extends Cart {
  status: 'idle' | 'loading' | 'succeeded' | 'failed'
  mutationStatus: 'idle' | 'loading' | 'succeeded' | 'failed'
  error: string | null
}

const emptyCart = (): Cart => ({
  items: [],
  total: 0,
  currency: 'RUB',
  totalItems: 0,
})

const initialState: CartState = {
  ...emptyCart(),
  status: 'idle',
  mutationStatus: 'idle',
  error: null,
}

const applyCart = (state: CartState, payload: Cart) => {
  state.items = payload.items
  state.total = payload.total
  state.currency = payload.currency
  state.totalItems = payload.totalItems
}

export const fetchCartThunk = createAsyncThunk('cart/fetch', async (_, { rejectWithValue }) => {
  try {
    return await cartService.getCart()
  } catch (error) {
    return rejectWithValue(getErrorMessage(error, 'Failed to load cart'))
  }
})

export const addCartItemThunk = createAsyncThunk(
  'cart/addItem',
  async (payload: { productId: string; quantity: number }, { rejectWithValue }) => {
    try {
      return await cartService.addItem(payload.productId, payload.quantity)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to add item to cart'))
    }
  },
)

export const updateCartItemThunk = createAsyncThunk(
  'cart/updateItem',
  async (payload: { productId: string; quantity: number }, { rejectWithValue }) => {
    try {
      return await cartService.updateItem(payload.productId, payload.quantity)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to update cart item'))
    }
  },
)

export const removeCartItemThunk = createAsyncThunk(
  'cart/removeItem',
  async (productId: string, { rejectWithValue }) => {
    try {
      return await cartService.removeItem(productId)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to remove cart item'))
    }
  },
)

export const clearCartThunk = createAsyncThunk('cart/clear', async (_, { rejectWithValue }) => {
  try {
    return await cartService.clear()
  } catch (error) {
    return rejectWithValue(getErrorMessage(error, 'Failed to clear cart'))
  }
})

const cartSlice = createSlice({
  name: 'cart',
  initialState,
  reducers: {
    clearCartState: () => initialState,
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchCartThunk.pending, (state) => {
        state.status = 'loading'
        state.error = null
      })
      .addCase(fetchCartThunk.fulfilled, (state, action) => {
        state.status = 'succeeded'
        applyCart(state, action.payload)
      })
      .addCase(fetchCartThunk.rejected, (state, action) => {
        state.status = 'failed'
        state.error = action.payload as string
      })
      .addCase(addCartItemThunk.pending, (state) => {
        state.mutationStatus = 'loading'
        state.error = null
      })
      .addCase(addCartItemThunk.fulfilled, (state, action) => {
        state.mutationStatus = 'succeeded'
        applyCart(state, action.payload)
      })
      .addCase(addCartItemThunk.rejected, (state, action) => {
        state.mutationStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(updateCartItemThunk.pending, (state) => {
        state.mutationStatus = 'loading'
        state.error = null
      })
      .addCase(updateCartItemThunk.fulfilled, (state, action) => {
        state.mutationStatus = 'succeeded'
        applyCart(state, action.payload)
      })
      .addCase(updateCartItemThunk.rejected, (state, action) => {
        state.mutationStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(removeCartItemThunk.pending, (state) => {
        state.mutationStatus = 'loading'
        state.error = null
      })
      .addCase(removeCartItemThunk.fulfilled, (state, action) => {
        state.mutationStatus = 'succeeded'
        applyCart(state, action.payload)
      })
      .addCase(removeCartItemThunk.rejected, (state, action) => {
        state.mutationStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(clearCartThunk.pending, (state) => {
        state.mutationStatus = 'loading'
        state.error = null
      })
      .addCase(clearCartThunk.fulfilled, (state, action) => {
        state.mutationStatus = 'succeeded'
        applyCart(state, action.payload)
      })
      .addCase(clearCartThunk.rejected, (state, action) => {
        state.mutationStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(checkoutThunk.fulfilled, (state) => {
        applyCart(state, emptyCart())
        state.mutationStatus = 'idle'
      })
      .addCase(logoutThunk.fulfilled, () => initialState)
      .addCase(logoutThunk.rejected, () => initialState)
  },
})

export const { clearCartState } = cartSlice.actions
export default cartSlice.reducer
