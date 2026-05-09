import { useState } from 'react'
import { api } from '../api'
import type { AnalyzeResult, Project } from '../types'
import { Button } from './Button'
import { FolderSearch, Sparkles, X } from 'lucide-react'

export function AddProjectDialog({ open, onClose, onCreated }: {
  open: boolean
  onClose: () => void
  onCreated: (p: Project) => void
}) {
  const [path, setPath] = useState('')
  const [analysis, setAnalysis] = useState<AnalyzeResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!open) return null

  const reset = () => { setPath(''); setAnalysis(null); setError(null); setBusy(false) }
  const close = () => { reset(); onClose() }

  const pickFolder = async () => {
    setError(null)
    try {
      const p = await api.selectFolder()
      if (!p) return
      setPath(p)
      await analyze(p)
    } catch (e: any) { setError(e?.message ?? String(e)) }
  }

  const analyze = async (target: string) => {
    setBusy(true); setError(null); setAnalysis(null)
    try {
      const a = await api.analyzePath(target)
      setAnalysis(a)
    } catch (e: any) { setError(e?.message ?? String(e)) }
    finally { setBusy(false) }
  }

  const create = async () => {
    if (!path) return
    setBusy(true); setError(null)
    try {
      const p = await api.addProject(path)
      onCreated(p); close()
    } catch (e: any) { setError(e?.message ?? String(e)); setBusy(false) }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={close}>
      <div onClick={e => e.stopPropagation()} className="w-full max-w-lg rounded-2xl bg-[var(--color-surface)] ring-1 ring-[var(--color-border)] shadow-2xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-[var(--color-border)]">
          <div>
            <h2 className="font-semibold tracking-tight">Add project</h2>
            <p className="text-xs text-[var(--color-muted)]">Select a local folder. Orbit will analyze it.</p>
          </div>
          <button onClick={close} className="text-[var(--color-muted)] hover:text-white"><X className="h-4 w-4" /></button>
        </div>

        <div className="p-5 space-y-4">
          <div className="flex gap-2">
            <input
              value={path}
              onChange={e => setPath(e.target.value)}
              placeholder="/path/to/project"
              className="flex-1 h-9 rounded-lg bg-[var(--color-bg)] ring-1 ring-[var(--color-border)] px-3 text-sm outline-none focus:ring-[var(--color-accent)]/60"
            />
            <Button onClick={pickFolder} variant="secondary"><FolderSearch className="h-4 w-4" />Browse</Button>
            <Button onClick={() => analyze(path)} disabled={!path || busy} variant="primary"><Sparkles className="h-4 w-4" />Analyze</Button>
          </div>

          {error && <div className="rounded-lg bg-rose-500/10 ring-1 ring-rose-500/30 text-rose-300 text-sm px-3 py-2">{error}</div>}

          {analysis && (
            <div className="rounded-xl bg-[var(--color-bg)]/40 ring-1 ring-[var(--color-border)] p-4 space-y-3">
              <Row label="Name" value={analysis.name} />
              <Row label="Framework" value={analysis.framework} />
              <Row label="Package manager" value={analysis.packageManager} />
              <Row label="Dev command" value={analysis.devCommand} mono />
              <Row label="Local domain" value={`https://${analysis.suggestedDomain}`} mono />
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-[var(--color-border)]">
          <Button onClick={close} variant="ghost">Cancel</Button>
          <Button onClick={create} disabled={!analysis || busy} variant="primary">Add to Orbit</Button>
        </div>
      </div>
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-[var(--color-muted)]">{label}</span>
      <span className={mono ? 'font-mono text-xs' : ''}>{value}</span>
    </div>
  )
}
