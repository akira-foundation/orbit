import { useState } from "react"
import { api } from "../api"
import type { AnalyzeResult, Project } from "../types"
import { FolderSearch } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogBody,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "./ui/dialog"
import { Button } from "./ui/button"
import { Input } from "./ui/input"

export function AddProjectDialog({
  open,
  onClose,
  onCreated,
}: {
  open: boolean
  onClose: () => void
  onCreated: (p: Project) => void
}) {
  const [path, setPath] = useState("")
  const [analysis, setAnalysis] = useState<AnalyzeResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reset = () => {
    setPath("")
    setAnalysis(null)
    setError(null)
    setBusy(false)
  }

  const close = () => {
    reset()
    onClose()
  }

  const analyze = async (target: string) => {
    const t = target.trim()
    if (!t) return
    setBusy(true)
    setError(null)
    setAnalysis(null)
    try {
      setAnalysis(await api.analyzePath(t))
    } catch (e: any) {
      setError(e?.message ?? String(e))
    } finally {
      setBusy(false)
    }
  }

  const pickFolder = async () => {
    setError(null)
    try {
      const p = await api.selectFolder()
      if (!p) return
      setPath(p)
      await analyze(p)
    } catch (e: any) {
      setError(e?.message ?? String(e))
    }
  }

  const create = async () => {
    if (!path) return
    setBusy(true)
    setError(null)
    try {
      onCreated(await api.addProject(path))
      close()
    } catch (e: any) {
      setError(e?.message ?? String(e))
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && close()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add project</DialogTitle>
          <DialogDescription>Select a local folder. Orbit will analyze it.</DialogDescription>
        </DialogHeader>

        <DialogBody className="space-y-4">
          {/* Path input */}
          <div className="flex gap-2">
            <Input
              value={path}
              onChange={(e) => {
                setPath(e.target.value)
                setAnalysis(null)
                setError(null)
              }}
              onBlur={(e) => analyze(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && analyze(path)}
              placeholder="/path/to/project"
              className="flex-1"
              disabled={busy}
            />
            <Button variant="outline" onClick={pickFolder} disabled={busy}>
              <FolderSearch />
              Browse
            </Button>
          </div>

          {/* Analyzing indicator */}
          {busy && (
            <p className="text-xs text-[var(--orbit-muted)]">Analyzing…</p>
          )}

          {/* Error */}
          {error && (
            <div className="rounded-md bg-destructive/15 border border-destructive/40 text-destructive text-sm px-3 py-2">
              {error}
            </div>
          )}

          {/* Analysis result */}
          {analysis && (
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-4 space-y-2">
              <Row label="Name"            value={analysis.name} />
              <Row label="Framework"       value={analysis.framework} />
              <Row label="Package manager" value={analysis.packageManager} />
              <Row label="Dev command"     value={analysis.devCommand} mono />
              <Row label="Local domain"    value={`https://${analysis.suggestedDomain}`} mono />
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="ghost" onClick={close}>Cancel</Button>
          <Button onClick={create} disabled={!analysis || busy}>Add to Orbit</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono text-xs" : ""}>{value}</span>
    </div>
  )
}
