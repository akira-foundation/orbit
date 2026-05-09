import type { AnalyzeResult, Project, ProjectStatus } from './types'

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          AnalyzePath(path: string): Promise<AnalyzeResult>
          AddProject(path: string): Promise<Project>
          ListProjects(): Promise<Project[] | null>
          GetProject(id: string): Promise<Project>
          DeleteProject(id: string): Promise<void>
          StartProject(id: string): Promise<void>
          StopProject(id: string): Promise<void>
          SelectProjectFolder(): Promise<string>
        }
      }
    }
  }
}

function bridge() {
  const app = window.go?.main?.App
  if (!app) throw new Error('Wails bindings not ready')
  return app
}

export const api = {
  analyzePath: (path: string) => bridge().AnalyzePath(path),
  addProject: (path: string) => bridge().AddProject(path),
  listProjects: async () => (await bridge().ListProjects()) ?? [],
  getProject: (id: string) => bridge().GetProject(id),
  deleteProject: (id: string) => bridge().DeleteProject(id),
  startProject: (id: string) => bridge().StartProject(id),
  stopProject: (id: string) => bridge().StopProject(id),
  selectFolder: () => bridge().SelectProjectFolder(),
}

export type { Project, ProjectStatus, AnalyzeResult }
