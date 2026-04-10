export type UserRole = 'guest' | 'customer' | 'seller' | 'admin'

export interface User {
  id: string
  email: string
  fullName?: string
  isActive?: boolean
  emailVerifiedAt?: string
  isEmailVerified?: boolean
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

export type ProductSpecValue = string | number | boolean | null
export type ProductSpecs = Record<string, ProductSpecValue>

export interface SearchSuggestion {
  text: string
  kind: string
}

export interface PopularSearch {
  query: string
  searchCount: number
}

export interface Product {
  id: string
  title: string
  name?: string
  slug?: string
  sku?: string
  sellerName?: string
  sellerSlug?: string
  description: string
  price: number
  currency?: string
  rating: number
  imageUrl?: string
  images: string[]
  brand?: string
  unit?: string
  specs?: ProductSpecs
  categoryId: string
  categoryName?: string
  sellerId?: string
  stock?: number
  reviewsCount?: number
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

export interface SessionInfo {
  id: string
  userId: string
  userAgent?: string
  ip?: string
  createdAt?: string
  lastSeenAt?: string
  expiresAt?: string
  isCurrent?: boolean
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
  sellerName?: string
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

export type SellerStatus = 'pending' | 'active' | 'paused'

export interface SellerProfile {
  userId: string
  storeName: string
  storeSlug: string
  legalName?: string
  description?: string
  logoUrl?: string
  bannerUrl?: string
  supportEmail?: string
  supportPhone?: string
  city?: string
  status: SellerStatus
  createdAt?: string
  updatedAt?: string
}

export interface SellerMetrics {
  productsTotal: number
  activeProducts: number
  hiddenProducts: number
  lowStockProducts: number
  ordersTotal: number
  unitsSold: number
  grossRevenue: number
}

export interface SellerOrderSummary {
  orderId: string
  status: OrderStatus
  currency: string
  createdAt: string
  placeTitle: string
  itemsCount: number
  grossRevenue: number
}

export interface SellerDashboard {
  profile: SellerProfile
  metrics: SellerMetrics
  recentProducts: Product[]
  lowStock: Product[]
  recentOrders: SellerOrderSummary[]
}

export type ConversationRole = 'buyer' | 'seller' | 'all'

export interface Conversation {
  id: string
  productId: string
  productName: string
  productImageUrl?: string
  sellerId: string
  sellerName: string
  sellerStoreName?: string
  buyerId: string
  buyerName: string
  orderId?: string
  lastMessageAt: string
  lastMessagePreview?: string
  unreadCount: number
  currentUserRole?: ConversationRole
}

export interface ConversationMessage {
  id: string
  conversationId: string
  senderId: string
  body: string
  createdAt: string
  editedAt?: string
}

export interface ConversationReadState {
  conversationId: string
  userId: string
  lastReadMessageId?: string
  lastReadAt?: string
}

export interface UnreadCount {
  total: number
}
