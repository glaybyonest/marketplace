import { createAsyncThunk, createSlice } from '@reduxjs/toolkit'

import { ordersService } from '@/services/ordersService'
import { logoutThunk } from '@/store/slices/authSlice'
import type { Order } from '@/types/domain'
import { getErrorMessage } from '@/utils/error'

interface OrdersState {
  items: Order[]
  selectedOrder: Order | null
  page: number
  total: number
  totalPages: number
  status: 'idle' | 'loading' | 'succeeded' | 'failed'
  detailStatus: 'idle' | 'loading' | 'succeeded' | 'failed'
  checkoutStatus: 'idle' | 'loading' | 'succeeded' | 'failed'
  error: string | null
}

const initialState: OrdersState = {
  items: [],
  selectedOrder: null,
  page: 1,
  total: 0,
  totalPages: 1,
  status: 'idle',
  detailStatus: 'idle',
  checkoutStatus: 'idle',
  error: null,
}

export const checkoutThunk = createAsyncThunk(
  'orders/checkout',
  async (placeId: string, { rejectWithValue }) => {
    try {
      return await ordersService.checkout(placeId)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to place order'))
    }
  },
)

export const fetchOrdersThunk = createAsyncThunk(
  'orders/fetch',
  async (payload: { page?: number; limit?: number } | undefined, { rejectWithValue }) => {
    try {
      return await ordersService.list(payload?.page ?? 1, payload?.limit ?? 20)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to load orders'))
    }
  },
)

export const fetchOrderByIdThunk = createAsyncThunk(
  'orders/fetchById',
  async (id: string, { rejectWithValue }) => {
    try {
      return await ordersService.getById(id)
    } catch (error) {
      return rejectWithValue(getErrorMessage(error, 'Failed to load order'))
    }
  },
)

const ordersSlice = createSlice({
  name: 'orders',
  initialState,
  reducers: {
    clearOrders(state) {
      state.items = []
      state.selectedOrder = null
      state.page = 1
      state.total = 0
      state.totalPages = 1
      state.status = 'idle'
      state.detailStatus = 'idle'
      state.checkoutStatus = 'idle'
      state.error = null
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(checkoutThunk.pending, (state) => {
        state.checkoutStatus = 'loading'
        state.error = null
      })
      .addCase(checkoutThunk.fulfilled, (state, action) => {
        state.checkoutStatus = 'succeeded'
        state.selectedOrder = action.payload
        state.items = [action.payload, ...state.items.filter((item) => item.id !== action.payload.id)]
        state.total += 1
      })
      .addCase(checkoutThunk.rejected, (state, action) => {
        state.checkoutStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(fetchOrdersThunk.pending, (state) => {
        state.status = 'loading'
        state.error = null
      })
      .addCase(fetchOrdersThunk.fulfilled, (state, action) => {
        state.status = 'succeeded'
        state.items = action.payload.items
        state.page = action.payload.page
        state.total = action.payload.total
        state.totalPages = action.payload.totalPages
      })
      .addCase(fetchOrdersThunk.rejected, (state, action) => {
        state.status = 'failed'
        state.error = action.payload as string
      })
      .addCase(fetchOrderByIdThunk.pending, (state) => {
        state.detailStatus = 'loading'
        state.error = null
      })
      .addCase(fetchOrderByIdThunk.fulfilled, (state, action) => {
        state.detailStatus = 'succeeded'
        state.selectedOrder = action.payload
      })
      .addCase(fetchOrderByIdThunk.rejected, (state, action) => {
        state.detailStatus = 'failed'
        state.error = action.payload as string
      })
      .addCase(logoutThunk.fulfilled, () => initialState)
      .addCase(logoutThunk.rejected, () => initialState)
  },
})

export const { clearOrders } = ordersSlice.actions
export default ordersSlice.reducer
