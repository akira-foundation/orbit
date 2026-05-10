export type ProjectStatus =
  | 'stopped'
  | 'starting'
  | 'running'
  | 'idle'
  | 'suspended'
  | 'error'

export interface ProjectScript {
  id: string
  projectId: string
  name: string
  command: string
  createdAt: string
}

import type { config } from "../wailsjs/go/models";
export type Config = config.Config;

export interface Project {
  id: string
  name: string
  path: string
  slug: string
  localDomain: string
  detectedFramework: string
  packageManager: string
  devCommand: string
  devPort: number
  status: ProjectStatus
  createdAt: string
  updatedAt: string
  scripts?: ProjectScript[]
}

export interface RuntimeLogLine {
  timestamp: string
  stream: 'stdout' | 'stderr' | 'system'
  text: string
}

export interface RuntimeSnapshot {
  projectId: string
  status: ProjectStatus
  pid: number
  port: number
  startedAt: string
  readyAt?: string
  uptimeMs: number
  startupMs?: number
  attempts?: number
  conns: number
  lastActivity: string
  error?: string
}

export interface RuntimeLogEvent {
  projectId: string
  line: RuntimeLogLine
}

export interface RuntimeStatusEvent {
  projectId: string
  snapshot: RuntimeSnapshot
}

export interface MetricSample {
  ts: number
  projectId: string
  status: ProjectStatus
  port: number
  conns: number
  uptimeMs: number
  attempts: number
  reqCount: number
  errCount: number
  httpReqs: number
  wsReqs: number
  bytesIn: number
  bytesOut: number
  p50Ms: number
  p95Ms: number
  p99Ms: number
  memKb: number
  cpuPct: number
  crashes: number
  autostops: number
  wakeMs: number
}

export interface SystemStatus {
  os: string
  setup: boolean
  loopbackOk: boolean
  dnsmasqOk: boolean
  resolverOk: boolean
  daemonOk: boolean
  herdConflict: boolean
  launchAtLogin: boolean
  message: string
}

export interface AnalyzeResult {
  name: string
  path: string
  packageManager: string
  framework: string
  devCommand: string
  devPort: number
  scripts: Record<string, string>
  suggestedSlug: string
  suggestedDomain: string
}
