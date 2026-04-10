import { MessagesInbox } from '@/components/messages/MessagesInbox'
import { SellerNav } from '@/components/seller/SellerNav'

export const SellerMessagesPage = () => (
  <MessagesInbox role="seller" nav={<SellerNav variant="surface" />} />
)
