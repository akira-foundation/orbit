import {
  AnalyzePath,
  AddProject,
  ListProjects,
  GetProject,
  DeleteProject,
  StartProject,
  StopProject,
  RestartProject,
  RuntimeStatus,
  RuntimeLogs,
  SelectProjectFolder,
  OpenProject,
  SystemStatus,
  SystemSetup,
  SystemUninstall,
} from "../wailsjs/go/main/App";
import type {
  AnalyzeResult,
  Project,
  RuntimeSnapshot,
  RuntimeLogLine,
  SystemStatus as SystemStatusType,
} from "./types";

function cast<T>(p: Promise<unknown>): Promise<T> {
  return p as Promise<T>;
}

export const api = {
  analyzePath: (path: string): Promise<AnalyzeResult> =>
    cast(AnalyzePath(path)),
  addProject: (path: string): Promise<Project> => cast(AddProject(path)),
  listProjects: async (): Promise<Project[]> =>
    (await cast<Project[] | null>(ListProjects())) ?? [],
  getProject: (id: string): Promise<Project> => cast(GetProject(id)),
  deleteProject: (id: string): Promise<void> => DeleteProject(id),
  startProject: (id: string): Promise<void> => StartProject(id),
  stopProject: (id: string): Promise<void> => StopProject(id),
  restartProject: (id: string): Promise<void> => RestartProject(id),
  runtimeStatus: (id: string): Promise<RuntimeSnapshot> =>
    cast(RuntimeStatus(id)),
  runtimeLogs: async (id: string): Promise<RuntimeLogLine[]> =>
    (await cast<RuntimeLogLine[] | null>(RuntimeLogs(id))) ?? [],
  selectFolder: (): Promise<string> => SelectProjectFolder(),
  openProject: (id: string): Promise<void> => OpenProject(id),
  systemStatus: (): Promise<SystemStatusType> => cast(SystemStatus()),
  systemSetup: (): Promise<void> => SystemSetup(),
  systemUninstall: (): Promise<void> => SystemUninstall(),
};

export type { Project, AnalyzeResult };
