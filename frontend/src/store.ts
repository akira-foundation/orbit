import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { Group, Project } from './types'
import { api } from './api'

export type Filter = 'all' | 'running' | 'idle' | 'stopped' | 'suspended' | 'error'
export type View = 'projects' | 'metrics' | 'project-metrics' | 'project-logs' | 'services' | 'runtimes' | 'mail' | 'database' | 'storage' | 'settings'

interface State {
  projects: Project[]
  selectedId: string | null
  view: View
  history: Array<string | null>
  historyIndex: number
  filter: Filter
  query: string
  loading: boolean
  error: string | null
  load: () => Promise<void>
  select: (id: string | null) => void
  setView: (v: View) => void
  back: () => void
  forward: () => void
  canBack: boolean
  canForward: boolean
  setFilter: (f: Filter) => void
  setQuery: (q: string) => void
  showProjectMetrics: (id: string) => void
  showProjectLogs: (id: string) => void
  add: (p: Project) => void
  remove: (id: string) => Promise<void>
  start: (id: string) => Promise<void>
  stop: (id: string) => Promise<void>
  patchStatus: (id: string, status: Project['status']) => void
  groups: Group[]
  loadGroups: () => Promise<void>
  createGroup: (name: string) => Promise<void>
  deleteGroup: (id: string) => Promise<void>
  startGroup: (id: string) => Promise<void>
  stopGroup: (id: string) => Promise<void>
  addToGroup: (groupId: string, projectId: string) => Promise<void>
  removeFromGroup: (groupId: string, projectId: string) => Promise<void>
}

export const useProjects = create<State>()(persist((set, get) => ({
  projects: [],
  selectedId: null,
  view: 'projects',
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
  setView(v) { set({ view: v, selectedId: v === 'projects' ? get().selectedId : null }) },
  showProjectMetrics(id) { set({ view: 'project-metrics', selectedId: id }) },
  showProjectLogs(id) { set({ view: 'project-logs', selectedId: id }) },
  async load() {
    set({ loading: true, error: null })
    try {
      const list = await api.listProjects()
      const { selectedId, view } = get()
      const stillExists = selectedId ? list.some(p => p.id === selectedId) : true
      if (!stillExists) {
        const projectScoped = view === 'project-metrics' || view === 'project-logs'
        set({ selectedId: null, view: projectScoped ? 'projects' : view })
      }
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
      view: 'projects',
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
  patchStatus(id, status) {
    set({
      projects: get().projects.map(p => p.id === id ? { ...p, status } : p),
    })
  },
  groups: [],
  async loadGroups() {
    set({ groups: await api.listGroups() })
  },
  async createGroup(name) {
    await api.createGroup(name)
    await get().loadGroups()
  },
  async deleteGroup(id) {
    await api.deleteGroup(id)
    await get().loadGroups()
  },
  async startGroup(id) {
    await api.startGroup(id)
    await get().load()
  },
  async stopGroup(id) {
    await api.stopGroup(id)
    await get().load()
  },
  async addToGroup(groupId, projectId) {
    await api.addToGroup(groupId, projectId)
    await get().loadGroups()
  },
  async removeFromGroup(groupId, projectId) {
    await api.removeFromGroup(groupId, projectId)
    await get().loadGroups()
  },
}), {
  name: 'orbit.nav',
  partialize: (s) => ({ view: s.view, selectedId: s.selectedId, filter: s.filter }),
}))
