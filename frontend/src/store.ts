import { create } from 'zustand'
import type { Project } from './types'
import { api } from './api'

export type Filter = 'all' | 'running' | 'idle' | 'stopped' | 'suspended' | 'error'

interface State {
  projects: Project[]
  selectedId: string | null
  history: Array<string | null>
  historyIndex: number
  filter: Filter
  query: string
  loading: boolean
  error: string | null
  load: () => Promise<void>
  select: (id: string | null) => void
  back: () => void
  forward: () => void
  canBack: boolean
  canForward: boolean
  setFilter: (f: Filter) => void
  setQuery: (q: string) => void
  add: (p: Project) => void
  remove: (id: string) => Promise<void>
  start: (id: string) => Promise<void>
  stop: (id: string) => Promise<void>
}

export const useProjects = create<State>((set, get) => ({
  projects: [],
  selectedId: null,
  history: [null],
  historyIndex: 0,
  canBack: false,
  canForward: false,
  filter: 'all',
  query: '',
  loading: false,
  error: null,
  setFilter(f) { set({ filter: f }) },
  setQuery(q) { set({ query: q }) },
  async load() {
    set({ loading: true, error: null })
    try {
      const list = await api.listProjects()
      set({ projects: list, loading: false })
    } catch (e: any) {
      set({ loading: false, error: e?.message ?? String(e) })
    }
  },
  select(id) {
    const { history, historyIndex } = get()
    const newHistory = [...history.slice(0, historyIndex + 1), id]
    const newIndex = newHistory.length - 1
    set({
      selectedId: id,
      history: newHistory,
      historyIndex: newIndex,
      canBack: newIndex > 0,
      canForward: false,
    })
  },
  back() {
    const { history, historyIndex } = get()
    if (historyIndex <= 0) return
    const newIndex = historyIndex - 1
    const id = history[newIndex]
    set({
      selectedId: id ?? null,
      historyIndex: newIndex,
      canBack: newIndex > 0,
      canForward: true,
    })
  },
  forward() {
    const { history, historyIndex } = get()
    if (historyIndex >= history.length - 1) return
    const newIndex = historyIndex + 1
    const id = history[newIndex]
    set({
      selectedId: id ?? null,
      historyIndex: newIndex,
      canBack: true,
      canForward: newIndex < history.length - 1,
    })
  },
  add(p) { set({ projects: [p, ...get().projects] }) },
  async remove(id) {
    await api.deleteProject(id)
    const { history, historyIndex } = get()
    // purge removed id from history
    const newHistory = history.map(h => (h === id ? null : h))
    set({
      projects: get().projects.filter(p => p.id !== id),
      selectedId: null,
      history: newHistory,
      historyIndex: 0,
      canBack: false,
      canForward: false,
    })
  },
  async start(id) {
    await api.startProject(id)
    await get().load()
  },
  async stop(id) {
    await api.stopProject(id)
    await get().load()
  },
}))
