import { useState } from "react"
import { api } from "../api"
import type { AnalyzeResult, Project } from "../types"
import { FolderSearch, PackageOpen, TriangleAlert } from "lucide-react"
import { Switch } from "./ui/switch"
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
  const [needsInstall, setNeedsInstall] = useState(false)
  const [installAfterAdd, setInstallAfterAdd] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState<string | null>(null)

  const reset = () => {
    setPath("")
    setAnalysis(null)
    setNeedsInstall(false)
    setInstallAfterAdd(true)
    setError(null)
    setConflict(null)
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
    setConflict(null)
    setAnalysis(null)
    try {
      const a = await api.analyzePath(t)
      setAnalysis(a)
      setNeedsInstall(await api.pathNeedsInstall(t))
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

  const finish = (proj: Project) => {
    onCreated(proj)
    if (needsInstall && installAfterAdd) {
      api.installProject(proj.id).catch((e) => {
        console.warn("install kicked off failed:", e)
      })
    }
    close()
  }

  const create = async () => {
    if (!path) return
    setBusy(true)
    setError(null)
    try {
      const res = await api.addProject(path)
      if (res.conflict) {
        setConflict(res.conflictName)
        setBusy(false)
        return
      }
      if (res.project) finish(res.project)
    } catch (e: any) {
      setError(e?.message ?? String(e))
      setBusy(false)
    }
  }

  const overwrite = async () => {
    if (!path) return
    setBusy(true)
    setError(null)
    try {
      const res = await api.addProjectOverwrite(path)
      if (res.project) finish(res.project)
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
                setConflict(null)
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

          {conflict && (
            <div className="flex items-start gap-3 rounded-xl border border-amber-400/20 bg-amber-400/[0.04] p-3.5">
              <TriangleAlert className="size-4 mt-0.5 text-amber-300 shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-[13px] font-medium text-amber-100">
                  Already registered
                </p>
                <p className="text-[11px] text-amber-200/70 mt-0.5">
                  A project named <span className="font-medium">{conflict}</span> already
                  exists. Overwrite replaces its registration with this folder;
                  its group membership and enabled services are reset.
                </p>
              </div>
            </div>
          )}

          {analysis?.ambiguousLayout && (
            <div className="flex items-start gap-3 rounded-xl border border-destructive/30 bg-destructive/[0.06] p-3.5">
              <TriangleAlert className="size-4 mt-0.5 text-destructive shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-[13px] font-medium text-destructive">Ambiguous layout</p>
                <p className="text-[11px] text-destructive/80 mt-0.5">
                  {analysis.ambiguousReason || "Could not determine a single backend to run. Point Orbit at a specific subfolder instead."}
                </p>
              </div>
            </div>
          )}

          {analysis && needsInstall && (
            <div className="flex items-start gap-3 rounded-xl border border-amber-400/20 bg-amber-400/[0.04] p-3.5">
              <PackageOpen className="size-4 mt-0.5 text-amber-300 shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-[13px] font-medium text-amber-100">
                  Install dependencies
                </p>
                <p className="text-[11px] text-amber-200/70 mt-0.5">
                  <code className="font-mono">node_modules</code> is missing. Orbit can
                  run{" "}
                  <code className="font-mono">
                    {installCmdFor(analysis.packageManager)}
                  </code>{" "}
                  right after adding the project.
                </p>
              </div>
              <Switch
                checked={installAfterAdd}
                onCheckedChange={setInstallAfterAdd}
                className="mt-0.5 shrink-0"
              />
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="ghost" onClick={close}>Cancel</Button>
          {conflict ? (
            <Button
              onClick={overwrite}
              disabled={busy}
              className="bg-amber-500/90 hover:bg-amber-500 text-black"
            >
              Overwrite
            </Button>
          ) : (
            <Button onClick={create} disabled={!analysis || busy || analysis.ambiguousLayout}>
              Add to Orbit
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function installCmdFor(pm: string): string {
  switch (pm) {
    case "yarn": return "yarn install"
    case "pnpm": return "pnpm install"
    case "bun":  return "bun install"
    default:     return "npm install"
  }
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className={mono ? "font-mono text-xs" : ""}>{value}</span>
    </div>
  )
}
