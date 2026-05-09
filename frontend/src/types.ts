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
