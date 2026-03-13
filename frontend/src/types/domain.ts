export type UserRole = 'guest' | 'customer' | 'seller' | 'admin'

export interface User {
  id: string
  email: string
  fullName?: string
  isActive?: boolean
  // legacy fields kept for compatibility with non-active screens
  name?: string
  role?: UserRole
  isBlocked?: boolean
  phone?: string
  avatarUrl?: string
}

export interface Category {
  id: string
  name: string
  slug?: string
  parentId?: string
  children?: Category[]
}

export interface Product {
  id: string
  title: string
  name?: string
  slug?: string
  description: string
  price: number
  currency?: string
  rating: number
  images: string[]
  categoryId: string
  categoryName?: string
  sellerId?: string
  stock?: number
  isPublished?: boolean
  isActive?: boolean
  isFavorite?: boolean
}

export interface Place {
  id: string
  title: string
  addressText: string
  lat?: number
  lon?: number
  createdAt?: string
  updatedAt?: string
}

export interface Review {
  id: string
  productId: string
  userId: string
  userName: string
  rating: number
  comment: string
  createdAt: string
}

export interface CartItem {
  id: string
  productId: string
  title: string
  slug?: string
  sku?: string
  imageUrl?: string
  price: number
  quantity: number
  lineTotal: number
  currency?: string
  stock?: number
  isActive?: boolean
}

export interface Cart {
  items: CartItem[]
  total: number
  currency: string
  totalItems: number
}

export type OrderStatus = 'pending' | 'cancelled'

export interface Order {
  id: string
  items: CartItem[]
  total: number
  currency: string
  status: OrderStatus
  createdAt: string
  updatedAt?: string
  placeId: string
  placeTitle: string
  addressText: string
  lat?: number
  lon?: number
}

export interface AdminStats {
  totalOrders: number
  totalUsers: number
  totalProducts: number
  activeSellers: number
}
