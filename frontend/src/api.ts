import {
  AnalyzePath,
  AddProject,
  ListProjects,
  GetProject,
  DeleteProject,
  StartProject,
  StopProject,
  SelectProjectFolder,
} from '../wailsjs/go/main/App'
import type { AnalyzeResult, Project } from './types'

// Cast through unknown because the wailsjs generated models use `any` for
// time.Time fields, while our hand-written types use `string`. The JSON wire
// format is identical (ISO-8601).
function cast<T>(p: Promise<unknown>): Promise<T> {
  return p as Promise<T>
}

export const api = {
  analyzePath:   (path: string): Promise<AnalyzeResult> => cast(AnalyzePath(path)),
  addProject:    (path: string): Promise<Project>       => cast(AddProject(path)),
  listProjects:  async (): Promise<Project[]>           => (await cast<Project[] | null>(ListProjects())) ?? [],
  getProject:    (id: string):   Promise<Project>       => cast(GetProject(id)),
  deleteProject: (id: string):   Promise<void>          => DeleteProject(id),
  startProject:  (id: string):   Promise<void>          => StartProject(id),
  stopProject:   (id: string):   Promise<void>          => StopProject(id),
  selectFolder:  ():             Promise<string>        => SelectProjectFolder(),
}

export type { Project, AnalyzeResult }
