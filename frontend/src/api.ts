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
  RuntimeLogsHistory,
  RuntimeMetrics,
  RuntimeMetricsAll,
  SystemConfig,
  SystemSaveConfig,
  SelectProjectFolder,
  OpenProject,
  RevealInFinder,
  SystemStatus,
  SystemSetup,
  SystemUninstall,
  SetLaunchAtLogin,
} from "../wailsjs/go/main/App";
import type {
  AnalyzeResult,
  MetricSample,
  Project,
  RuntimeSnapshot,
  RuntimeLogLine,
  SystemStatus as SystemStatusType,
  Config,
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
  runtimeLogsHistory: async (
    id: string,
    sinceTs: number,
    limit: number,
  ): Promise<RuntimeLogLine[]> =>
    (await cast<RuntimeLogLine[] | null>(
      RuntimeLogsHistory(id, sinceTs, limit),
    )) ?? [],
  runtimeMetrics: async (id: string, sinceTs: number): Promise<MetricSample[]> =>
    (await cast<MetricSample[] | null>(RuntimeMetrics(id, sinceTs))) ?? [],
  runtimeMetricsAll: async (
    sinceTs: number,
    ids: string[],
  ): Promise<Record<string, MetricSample[]>> =>
    (await cast<Record<string, MetricSample[]> | null>(
      RuntimeMetricsAll(sinceTs, ids),
    )) ?? {},
  selectFolder: (): Promise<string> => SelectProjectFolder(),
  openProject: (id: string): Promise<void> => OpenProject(id),
  revealInFinder: (path: string): Promise<void> => RevealInFinder(path),
  systemStatus: (): Promise<SystemStatusType> => cast(SystemStatus()),
  systemSetup: (): Promise<void> => SystemSetup(),
  systemUninstall: (): Promise<void> => SystemUninstall(),
  systemConfig: (): Promise<Config> => cast(SystemConfig()),
  systemSaveConfig: (cfg: Config): Promise<void> => SystemSaveConfig(cfg),
  setLaunchAtLogin: (enabled: boolean): Promise<void> =>
    SetLaunchAtLogin(enabled),
};

export type { Project, AnalyzeResult, Config };
