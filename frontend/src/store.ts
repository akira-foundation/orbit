import { create } from 'zustand'
import type { Project } from './types'
import { api } from './api'

export type Filter = 'all' | 'running' | 'idle' | 'stopped' | 'suspended' | 'error'

interface State {
  projects: Project[]
  selectedId: string | null
  filter: Filter
  query: string
  loading: boolean
  error: string | null
  load: () => Promise<void>
  select: (id: string | null) => void
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
  select(id) { set({ selectedId: id }) },
  add(p) { set({ projects: [p, ...get().projects] }) },
  async remove(id) {
    await api.deleteProject(id)
    set({ projects: get().projects.filter(p => p.id !== id), selectedId: null })
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
