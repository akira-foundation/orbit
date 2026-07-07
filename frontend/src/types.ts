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

export interface ProjectProcess {
  id: string
  projectId: string
  role: 'backend' | 'frontend'
  kind: 'command' | 'php-fpm'
  workDir: string
  command: string
  port: number
  phpVersion: string
  order: number
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
  nodeVersion?: string
  status: ProjectStatus
  secure: boolean
  createdAt: string
  updatedAt: string
  scripts?: ProjectScript[]
  processes?: ProjectProcess[]
}

export type NotificationKind = 'crash' | 'cert' | 'info'

export interface AppNotification {
  id: string
  kind: NotificationKind
  title: string
  body: string
  ts: number
  read: boolean
}

export interface Group {
  id: string
  name: string
  order: number
  createdAt: string
  projectIds: string[]
}

export interface QueryResult {
  columns: string[]
  rows: string[][]
}

export interface S3Object {
  key: string
  size: number
  lastModified: string
}

export interface ShareInfo {
  enabled: boolean
  url: string
  qr: string
}

export interface RequestEntry {
  ts: number
  method: string
  path: string
  status: number
  durationMs: number
  bytes: number
  ws: boolean
}

export interface GitInfo {
  repo: boolean
  branch: string
  dirty: number
  ahead: number
  behind: number
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
  tlsDaemonOk: boolean
  caTrustedOk: boolean
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
  nodeVersion?: string
  scripts: Record<string, string>
  suggestedSlug: string
  suggestedDomain: string
  processes?: ProjectProcess[]
  ambiguousLayout?: boolean
  ambiguousReason?: string
}

export interface ServiceInfo {
  engine: string
  displayName: string
  description: string
  family: string
  version: string
  status: "stopped" | "starting" | "running" | "error" | "external"
  webUrl: string
  installed: boolean
  refs: number
}

export interface NodeVersionInfo {
  id: string
  version: string
  installed: boolean
  diskBytes: number
  path: string
}

export interface PHPVersionInfo {
  id: string
  version: string
  installed: boolean
  diskBytes: number
  path: string
}

export interface ServiceSnapshot {
  engine: string
  status: string
  pid: number
  refs: number
}

export interface ServiceSetupField {
  label: string
  value: string
}

export interface ServiceSetupSnippet {
  label: string
  language: string
  code: string
}

export interface ServiceSetup {
  fields: ServiceSetupField[]
  snippets: ServiceSetupSnippet[]
}

export interface ServicesConfig {
  autoManage: boolean
  idleStopMinutes: number
  defaults: Record<string, boolean>
}

export interface RuntimesConfig {
  preferSystemNode: boolean
  preferSystemPhp: boolean
}

export interface SystemRuntimeStatus {
  available: boolean
  version: string
}
