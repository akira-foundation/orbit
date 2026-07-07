import { create } from 'zustand'
import type { AppNotification, NotificationKind } from '../types'

const MAX_ITEMS = 50

interface NotificationsState {
  items: AppNotification[]
  push: (n: { kind: NotificationKind; title: string; body: string }) => void
  markAllRead: () => void
  remove: (id: string) => void
  clear: () => void
}

export const useNotificationStore = create<NotificationsState>((set) => ({
  items: [],
  push({ kind, title, body }) {
    const item: AppNotification = {
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      kind,
      title,
      body,
      ts: Date.now(),
      read: false,
    }
    set((s) => ({ items: [item, ...s.items].slice(0, MAX_ITEMS) }))
  },
  markAllRead() {
    set((s) => ({ items: s.items.map((i) => ({ ...i, read: true })) }))
  },
  remove(id) {
    set((s) => ({ items: s.items.filter((i) => i.id !== id) }))
  },
  clear() {
    set({ items: [] })
  },
}))
