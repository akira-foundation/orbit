import { Logo } from './components/Logo'
import { Dashboard } from './pages/Dashboard'
import { ProjectDetail } from './pages/ProjectDetail'
import { useProjects } from './store'

export function App() {
  const selectedId = useProjects(s => s.selectedId)

  return (
    <div className="min-h-full flex flex-col">
      <header className="sticky top-0 z-10 backdrop-blur bg-[var(--color-bg)]/80 border-b border-[var(--color-border)]">
        <div className="max-w-6xl mx-auto px-6 h-14 flex items-center justify-between">
          <Logo />
          <div className="text-xs text-[var(--color-muted)]">v0.1 · local-first</div>
        </div>
      </header>

      <main className="flex-1 max-w-6xl w-full mx-auto px-6 py-8 scrollbar-thin">
        {selectedId
          ? <ProjectDetail id={selectedId} />
          : <Dashboard />}
      </main>
    </div>
  )
}
