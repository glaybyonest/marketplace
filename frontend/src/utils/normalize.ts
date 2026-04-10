import type {
  AdminStats,
  Cart,
  CartItem,
  Category,
  Conversation,
  ConversationMessage,
  ConversationReadState,
  ConversationRole,
  Order,
  Place,
  Product,
  ProductSpecs,
  Review,
  SellerDashboard,
  SellerMetrics,
  SellerOrderSummary,
  SellerProfile,
  SellerStatus,
  SessionInfo,
  UnreadCount,
  User,
  UserRole,
} from '@/types/domain'

const asRecord = (value: unknown): Record<string, unknown> =>
  (value as Record<string, unknown>) ?? {}

const pickString = (value: unknown, fallback = '') => (typeof value === 'string' ? value : fallback)

const pickNumber = (value: unknown, fallback = 0) =>
  typeof value === 'number' && Number.isFinite(value) ? value : fallback

const asStringArray = (value: unknown): string[] =>
  Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string' && item.length > 0)
    : []

const normalizeSpecs = (value: unknown): ProductSpecs => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }

  const result: ProductSpecs = {}
  for (const [key, item] of Object.entries(value as Record<string, unknown>)) {
    if (
      typeof item === 'string' ||
      typeof item === 'number' ||
      typeof item === 'boolean' ||
      item === null
    ) {
      result[key] = item
    }
  }
  return result
}

const ensureRole = (value: unknown, hasIdentity = false): UserRole => {
  if (value === 'customer' || value === 'seller' || value === 'admin') {
    return value
  }
  return hasIdentity ? 'customer' : 'guest'
}

const createId = () => {
  if (
    typeof globalThis.crypto !== 'undefined' &&
    typeof globalThis.crypto.randomUUID === 'function'
  ) {
    return globalThis.crypto.randomUUID()
  }

  return Math.random().toString(36).slice(2, 10)
}

export const normalizeUser = (input: unknown): User => {
  const source = asRecord(input)
  const email = pickString(source.email)
  const fullName = pickString(
    source.full_name,
    pickString(source.fullName, pickString(source.name)),
  )
  const emailVerifiedAt = pickString(source.email_verified_at, pickString(source.emailVerifiedAt))
  const isEmailVerified =
    source.is_email_verified === undefined
      ? emailVerifiedAt.length > 0
      : Boolean(source.is_email_verified)

  return {
    id: pickString(source.id, pickString(source._id, createId())),
    email,
    fullName,
    isActive: source.is_active === undefined ? true : Boolean(source.is_active),
    emailVerifiedAt: emailVerifiedAt || undefined,
    isEmailVerified,
    name: fullName || 'User',
    role: ensureRole(source.role, Boolean(email)),
    isBlocked: Boolean(source.isBlocked),
    phone: pickString(source.phone),
    avatarUrl: pickString(source.avatarUrl, pickString(source.avatar)),
  }
}

export const normalizeCategory = (input: unknown): Category => {
  const source = asRecord(input)
  const category: Category = {
    id: pickString(source.id, pickString(source._id, createId())),
    name: pickString(source.name, 'Uncategorized'),
    slug: pickString(source.slug),
    parentId: pickString(source.parent_id, pickString(source.parentId)),
  }

  if (Array.isArray(source.children)) {
    category.children = source.children.map(normalizeCategory)
  }

  return category
}

export const normalizeProduct = (input: unknown): Product => {
  const source = asRecord(input)
  const imageUrl = pickString(
    source.image_url,
    pickString(source.imageUrl, pickString(source.image)),
  )
  const gallery =
    asStringArray(source.images).length > 0
      ? asStringArray(source.images)
      : asStringArray(source.gallery)
  const images = Array.from(new Set([imageUrl, ...gallery].filter((image) => image.length > 0)))
  const specs = normalizeSpecs(source.specs)

  return {
    id: pickString(source.id, pickString(source._id, createId())),
    title: pickString(source.title, pickString(source.name, 'Product')),
    name: pickString(source.name, pickString(source.title, 'Product')),
    slug: pickString(source.slug),
    sku: pickString(source.sku),
    sellerName: pickString(source.seller_name, pickString(source.sellerName)),
    sellerSlug: pickString(source.seller_slug, pickString(source.sellerSlug)),
    description: pickString(source.description),
    price: pickNumber(source.price),
    currency: pickString(source.currency, 'RUB'),
    rating: pickNumber(source.rating, 0),
    imageUrl: imageUrl || images[0],
    images,
    brand: pickString(source.brand),
    unit: pickString(source.unit),
    specs,
    categoryId: pickString(
      source.category_id,
      pickString(source.categoryId, pickString(source.category, '')),
    ),
    categoryName: pickString(source.categoryName, pickString(source.category_name)),
    sellerId: pickString(source.seller_id, pickString(source.sellerId)),
    stock: pickNumber(source.stock_qty, pickNumber(source.stock, 0)),
    reviewsCount: pickNumber(source.reviews_count, pickNumber(source.reviewsCount, 0)),
    isPublished:
      source.isPublished === undefined
        ? Boolean(source.is_active ?? true)
        : Boolean(source.isPublished),
    isActive: source.is_active === undefined ? true : Boolean(source.is_active),
  }
}

export const normalizePlace = (input: unknown): Place => {
  const source = asRecord(input)
  return {
    id: pickString(source.id, createId()),
    title: pickString(source.title),
    addressText: pickString(source.address_text, pickString(source.addressText)),
    lat: typeof source.lat === 'number' ? source.lat : undefined,
    lon: typeof source.lon === 'number' ? source.lon : undefined,
    createdAt: pickString(source.created_at, pickString(source.createdAt)),
    updatedAt: pickString(source.updated_at, pickString(source.updatedAt)),
  }
}

export const normalizeSession = (input: unknown): SessionInfo => {
  const source = asRecord(input)
  return {
    id: pickString(source.id, createId()),
    userId: pickString(source.user_id, pickString(source.userId)),
    userAgent: pickString(source.user_agent, pickString(source.userAgent)) || undefined,
    ip: pickString(source.ip) || undefined,
    createdAt: pickString(source.created_at, pickString(source.createdAt)) || undefined,
    lastSeenAt: pickString(source.last_seen_at, pickString(source.lastSeenAt)) || undefined,
    expiresAt: pickString(source.expires_at, pickString(source.expiresAt)) || undefined,
    isCurrent: Boolean(source.is_current ?? source.isCurrent),
  }
}

export const normalizeReview = (input: unknown): Review => {
  const source = asRecord(input)
  return {
    id: pickString(source.id, pickString(source._id, createId())),
    productId: pickString(source.product_id, pickString(source.productId)),
    userId: pickString(source.user_id, pickString(source.userId)),
    userName: pickString(
      source.user_name,
      pickString(
        source.userName,
        pickString(source.author_name, pickString(source.authorName, 'Buyer')),
      ),
    ),
    rating: pickNumber(source.rating),
    comment: pickString(source.comment, pickString(source.text)),
    createdAt: pickString(
      source.created_at,
      pickString(source.createdAt, new Date().toISOString()),
    ),
  }
}

export const normalizeCartItem = (input: unknown): CartItem => {
  const source = asRecord(input)
  const productId = pickString(
    source.product_id,
    pickString(source.productId, pickString(source.id, createId())),
  )
  const lineTotal = pickNumber(
    source.line_total,
    pickNumber(
      source.lineTotal,
      pickNumber(source.total, pickNumber(source.price) * pickNumber(source.quantity, 1)),
    ),
  )
  return {
    id: pickString(source.id, pickString(source._id, productId)),
    productId,
    title: pickString(
      source.title,
      pickString(
        source.product_name,
        pickString(source.name, pickString(source.productTitle, 'Product')),
      ),
    ),
    slug: pickString(source.slug),
    sku: pickString(source.sku),
    sellerName: pickString(
      source.seller_store_name,
      pickString(source.seller_name, pickString(source.sellerName)),
    ),
    imageUrl: pickString(source.image_url, pickString(source.imageUrl, pickString(source.image))),
    price: pickNumber(source.unit_price, pickNumber(source.price)),
    quantity: pickNumber(source.quantity, 1),
    lineTotal,
    currency: pickString(source.currency, 'RUB'),
    stock: pickNumber(source.stock_qty, pickNumber(source.stock, 0)),
    isActive: source.is_active === undefined ? true : Boolean(source.is_active),
  }
}

export const normalizeCart = (input: unknown): Cart => {
  const source = asRecord(input)
  const rawItems = Array.isArray(source.items) ? source.items : []
  const items = rawItems.map(normalizeCartItem)
  const total =
    pickNumber(source.total_amount, pickNumber(source.total, NaN)) ||
    items.reduce((acc, item) => acc + item.lineTotal, 0)
  const totalItems =
    pickNumber(source.total_items, NaN) || items.reduce((acc, item) => acc + item.quantity, 0)
  return {
    items,
    total,
    currency: pickString(source.currency, 'RUB'),
    totalItems,
  }
}

export const normalizeOrder = (input: unknown): Order => {
  const source = asRecord(input)
  const rawItems = Array.isArray(source.items) ? source.items : []
  return {
    id: pickString(source.id, pickString(source._id, createId())),
    items: rawItems.map(normalizeCartItem),
    total: pickNumber(source.total_amount, pickNumber(source.total)),
    currency: pickString(source.currency, 'RUB'),
    status: pickString(source.status, 'pending') as Order['status'],
    createdAt: pickString(
      source.created_at,
      pickString(source.createdAt, new Date().toISOString()),
    ),
    updatedAt: pickString(source.updated_at, pickString(source.updatedAt)),
    placeId: pickString(source.place_id, pickString(source.placeId)),
    placeTitle: pickString(source.place_title, pickString(source.placeTitle)),
    addressText: pickString(source.address_text, pickString(source.addressText)),
    lat: typeof source.lat === 'number' ? source.lat : undefined,
    lon: typeof source.lon === 'number' ? source.lon : undefined,
  }
}

export const normalizeAdminStats = (input: unknown): AdminStats => {
  const source = asRecord(input)
  return {
    totalOrders: pickNumber(source.totalOrders),
    totalUsers: pickNumber(source.totalUsers),
    totalProducts: pickNumber(source.totalProducts),
    activeSellers: pickNumber(source.activeSellers),
  }
}

const ensureSellerStatus = (value: unknown): SellerStatus => {
  if (value === 'pending' || value === 'paused') {
    return value
  }
  return 'active'
}

const ensureConversationRole = (value: unknown): ConversationRole | undefined => {
  if (value === 'buyer' || value === 'seller' || value === 'all') {
    return value
  }
  return undefined
}

export const normalizeSellerProfile = (input: unknown): SellerProfile => {
  const source = asRecord(input)
  return {
    userId: pickString(source.user_id, pickString(source.userId)),
    storeName: pickString(source.store_name, pickString(source.storeName)),
    storeSlug: pickString(source.store_slug, pickString(source.storeSlug)),
    legalName: pickString(source.legal_name, pickString(source.legalName)) || undefined,
    description: pickString(source.description) || undefined,
    logoUrl: pickString(source.logo_url, pickString(source.logoUrl)) || undefined,
    bannerUrl: pickString(source.banner_url, pickString(source.bannerUrl)) || undefined,
    supportEmail: pickString(source.support_email, pickString(source.supportEmail)) || undefined,
    supportPhone: pickString(source.support_phone, pickString(source.supportPhone)) || undefined,
    city: pickString(source.city) || undefined,
    status: ensureSellerStatus(source.status),
    createdAt: pickString(source.created_at, pickString(source.createdAt)) || undefined,
    updatedAt: pickString(source.updated_at, pickString(source.updatedAt)) || undefined,
  }
}

export const normalizeSellerMetrics = (input: unknown): SellerMetrics => {
  const source = asRecord(input)
  return {
    productsTotal: pickNumber(source.products_total, pickNumber(source.productsTotal)),
    activeProducts: pickNumber(source.active_products, pickNumber(source.activeProducts)),
    hiddenProducts: pickNumber(source.hidden_products, pickNumber(source.hiddenProducts)),
    lowStockProducts: pickNumber(source.low_stock_products, pickNumber(source.lowStockProducts)),
    ordersTotal: pickNumber(source.orders_total, pickNumber(source.ordersTotal)),
    unitsSold: pickNumber(source.units_sold, pickNumber(source.unitsSold)),
    grossRevenue: pickNumber(source.gross_revenue, pickNumber(source.grossRevenue)),
  }
}

export const normalizeSellerOrderSummary = (input: unknown): SellerOrderSummary => {
  const source = asRecord(input)
  return {
    orderId: pickString(source.order_id, pickString(source.orderId)),
    status: pickString(source.status, 'pending') as SellerOrderSummary['status'],
    currency: pickString(source.currency, 'RUB'),
    createdAt: pickString(
      source.created_at,
      pickString(source.createdAt, new Date().toISOString()),
    ),
    placeTitle: pickString(source.place_title, pickString(source.placeTitle)),
    itemsCount: pickNumber(source.items_count, pickNumber(source.itemsCount)),
    grossRevenue: pickNumber(source.gross_revenue, pickNumber(source.grossRevenue)),
  }
}

export const normalizeSellerDashboard = (input: unknown): SellerDashboard => {
  const source = asRecord(input)
  const recentProducts = Array.isArray(source.recent_products)
    ? source.recent_products.map(normalizeProduct)
    : Array.isArray(source.recentProducts)
      ? source.recentProducts.map(normalizeProduct)
      : []
  const lowStock = Array.isArray(source.low_stock)
    ? source.low_stock.map(normalizeProduct)
    : Array.isArray(source.lowStock)
      ? source.lowStock.map(normalizeProduct)
      : []
  const recentOrders = Array.isArray(source.recent_orders)
    ? source.recent_orders.map(normalizeSellerOrderSummary)
    : Array.isArray(source.recentOrders)
      ? source.recentOrders.map(normalizeSellerOrderSummary)
      : []

  return {
    profile: normalizeSellerProfile(source.profile ?? {}),
    metrics: normalizeSellerMetrics(source.metrics ?? {}),
    recentProducts,
    lowStock,
    recentOrders,
  }
}

export const normalizeConversation = (input: unknown): Conversation => {
  const source = asRecord(input)
  return {
    id: pickString(source.id, createId()),
    productId: pickString(source.product_id, pickString(source.productId)),
    productName: pickString(source.product_name, pickString(source.productName, 'Товар')),
    productImageUrl:
      pickString(source.product_image_url, pickString(source.productImageUrl)) || undefined,
    sellerId: pickString(source.seller_id, pickString(source.sellerId)),
    sellerName: pickString(source.seller_name, pickString(source.sellerName, 'Продавец')),
    sellerStoreName:
      pickString(source.seller_store_name, pickString(source.sellerStoreName)) || undefined,
    buyerId: pickString(source.buyer_id, pickString(source.buyerId)),
    buyerName: pickString(source.buyer_name, pickString(source.buyerName, 'Покупатель')),
    orderId: pickString(source.order_id, pickString(source.orderId)) || undefined,
    lastMessageAt: pickString(
      source.last_message_at,
      pickString(source.lastMessageAt, new Date().toISOString()),
    ),
    lastMessagePreview:
      pickString(source.last_message_preview, pickString(source.lastMessagePreview)) || undefined,
    unreadCount: pickNumber(source.unread_count, pickNumber(source.unreadCount)),
    currentUserRole:
      ensureConversationRole(source.current_user_role) ??
      ensureConversationRole(source.currentUserRole),
  }
}

export const normalizeConversationMessage = (input: unknown): ConversationMessage => {
  const source = asRecord(input)
  return {
    id: pickString(source.id, createId()),
    conversationId: pickString(source.conversation_id, pickString(source.conversationId)),
    senderId: pickString(source.sender_id, pickString(source.senderId)),
    body: pickString(source.body),
    createdAt: pickString(
      source.created_at,
      pickString(source.createdAt, new Date().toISOString()),
    ),
    editedAt: pickString(source.edited_at, pickString(source.editedAt)) || undefined,
  }
}

export const normalizeConversationReadState = (input: unknown): ConversationReadState => {
  const source = asRecord(input)
  return {
    conversationId: pickString(source.conversation_id, pickString(source.conversationId)),
    userId: pickString(source.user_id, pickString(source.userId)),
    lastReadMessageId:
      pickString(source.last_read_message_id, pickString(source.lastReadMessageId)) || undefined,
    lastReadAt: pickString(source.last_read_at, pickString(source.lastReadAt)) || undefined,
  }
}

export const normalizeUnreadCount = (input: unknown): UnreadCount => {
  const source = asRecord(input)
  return {
    total: pickNumber(source.total),
  }
}
