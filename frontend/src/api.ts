import {
  AnalyzePath,
  AddProject,
  PathNeedsInstall,
  InstallProject,
  ListProjects,
  GetProject,
  DeleteProject,
  StartProject,
  StopProject,
  RestartProject,
  SelectProjectFolder,
  OpenProject,
  SetProjectSecure,
  ProjectEnv,
  SaveProjectEnv,
} from "../wailsjs/go/bindings/Projects";
import {
  ListGroups,
  CreateGroup,
  DeleteGroup,
  AddToGroup,
  RemoveFromGroup,
  StartGroup,
  StopGroup,
} from "../wailsjs/go/bindings/Groups";
import {
  ShareInfo as ShareInfoBinding,
  EnableLANShare,
  DisableLANShare,
} from "../wailsjs/go/bindings/Share";
import { ProjectRequests } from "../wailsjs/go/bindings/Requests";
import {
  RuntimeStatus,
  RuntimeLogs,
  RuntimeLogsHistory,
  RuntimeMetrics,
  RuntimeMetricsAll,
} from "../wailsjs/go/bindings/Runtime";
import {
  ListServices,
  ServiceStatus,
  StartService,
  StopService,
  EnableServiceForProject,
  DisableServiceForProject,
  ProjectServices,
  DetectedServices,
  UninstallService,
  ServiceSetup,
  ServicesConfig,
  SaveServicesConfig,
  ServicesDiskUsage,
  ClearServicesData,
} from "../wailsjs/go/bindings/Services";
import {
  MailList,
  MailGet,
  MailSetRead,
  MailDelete,
  MailDeleteAll,
  MailPart,
} from "../wailsjs/go/bindings/Mail";
import {
  ListNodeVersions,
  InstallNodeVersion,
  RemoveNodeVersion,
  ListPHPVersions,
  InstallPHPVersion,
  RemovePHPVersion,
  RuntimesConfig,
  SaveRuntimesConfig,
  DetectSystemNode,
  DetectSystemPHP,
} from "../wailsjs/go/bindings/Versions";
import {
  TerminalStart,
  TerminalWrite,
  TerminalResize,
  TerminalBuffer,
} from "../wailsjs/go/bindings/Terminal";
import {
  SystemConfig,
  SystemSaveConfig,
  OpenURL,
  GitStatus,
  Notify,
  RevealInFinder,
  SystemStatus,
  SystemSetup,
  SystemUninstall,
  TrustCA,
  UntrustCA,
  SetLaunchAtLogin,
} from "../wailsjs/go/bindings/System";
import type { mailpit } from "../wailsjs/go/models";
import type {
  AnalyzeResult,
  GitInfo,
  Group,
  RequestEntry,
  MetricSample,
  Project,
  ShareInfo,
  RuntimeSnapshot,
  RuntimeLogLine,
  SystemStatus as SystemStatusType,
  Config,
  ServiceInfo,
  ServiceSnapshot,
  ServiceSetup as ServiceSetupType,
  ServicesConfig as ServicesConfigType,
  NodeVersionInfo,
  PHPVersionInfo,
  RuntimesConfig as RuntimesConfigType,
  SystemRuntimeStatus,
} from "./types";

function cast<T>(p: Promise<unknown>): Promise<T> {
  return p as Promise<T>;
}

export const api = {
  analyzePath: (path: string): Promise<AnalyzeResult> =>
    cast(AnalyzePath(path)),
  addProject: (path: string): Promise<Project> => cast(AddProject(path)),
  pathNeedsInstall: (path: string): Promise<boolean> => cast(PathNeedsInstall(path)),
  installProject: (id: string): Promise<void> => InstallProject(id),
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
  openURL: (url: string): Promise<void> => OpenURL(url),
  gitStatus: (id: string): Promise<GitInfo> => cast(GitStatus(id)),
  projectRequests: async (id: string): Promise<RequestEntry[]> =>
    (await cast<RequestEntry[] | null>(ProjectRequests(id))) ?? [],
  shareInfo: (id: string): Promise<ShareInfo> => cast(ShareInfoBinding(id)),
  enableLANShare: (id: string): Promise<ShareInfo> => cast(EnableLANShare(id)),
  disableLANShare: (id: string): Promise<void> => DisableLANShare(id),
  notify: (title: string, body: string): Promise<void> => Notify(title, body),
  revealInFinder: (path: string): Promise<void> => RevealInFinder(path),
  systemStatus: (): Promise<SystemStatusType> => cast(SystemStatus()),
  systemSetup: (): Promise<void> => SystemSetup(),
  systemUninstall: (): Promise<void> => SystemUninstall(),
  trustCA: (): Promise<void> => TrustCA(),
  untrustCA: (): Promise<void> => UntrustCA(),
  setProjectSecure: (id: string, secure: boolean): Promise<void> =>
    SetProjectSecure(id, secure),
  projectEnv: async (id: string): Promise<Record<string, string>> =>
    (await cast<Record<string, string> | null>(ProjectEnv(id))) ?? {},
  saveProjectEnv: (id: string, vars: Record<string, string>): Promise<void> =>
    SaveProjectEnv(id, vars),
  listGroups: async (): Promise<Group[]> =>
    (await cast<Group[] | null>(ListGroups())) ?? [],
  createGroup: (name: string): Promise<Group> => cast(CreateGroup(name)),
  deleteGroup: (id: string): Promise<void> => DeleteGroup(id),
  addToGroup: (groupId: string, projectId: string): Promise<void> =>
    AddToGroup(groupId, projectId),
  removeFromGroup: (groupId: string, projectId: string): Promise<void> =>
    RemoveFromGroup(groupId, projectId),
  startGroup: (id: string): Promise<void> => StartGroup(id),
  stopGroup: (id: string): Promise<void> => StopGroup(id),
  systemConfig: (): Promise<Config> => cast(SystemConfig()),
  systemSaveConfig: (cfg: Config): Promise<void> => SystemSaveConfig(cfg),
  setLaunchAtLogin: (enabled: boolean): Promise<void> =>
    SetLaunchAtLogin(enabled),
  listServices: async (): Promise<ServiceInfo[]> =>
    (await cast<ServiceInfo[] | null>(ListServices())) ?? [],
  serviceStatus: (engine: string): Promise<ServiceSnapshot> =>
    cast(ServiceStatus(engine)),
  startService: (engine: string): Promise<void> => StartService(engine),
  stopService: (engine: string): Promise<void> => StopService(engine),
  enableServiceForProject: (projectId: string, engine: string): Promise<void> =>
    EnableServiceForProject(projectId, engine),
  disableServiceForProject: (projectId: string, engine: string): Promise<void> =>
    DisableServiceForProject(projectId, engine),
  projectServices: async (projectId: string): Promise<string[]> =>
    (await cast<string[] | null>(ProjectServices(projectId))) ?? [],
  detectedServices: async (projectId: string): Promise<string[]> =>
    (await cast<string[] | null>(DetectedServices(projectId))) ?? [],
  uninstallService: (engine: string): Promise<void> => UninstallService(engine),
  serviceSetup: (engine: string): Promise<ServiceSetupType> =>
    cast(ServiceSetup(engine)),
  servicesConfig: (): Promise<ServicesConfigType> => cast(ServicesConfig()),
  saveServicesConfig: (cfg: ServicesConfigType): Promise<void> =>
    SaveServicesConfig(cfg as never),
  servicesDiskUsage: (): Promise<number> => cast(ServicesDiskUsage()),
  clearServicesData: (): Promise<void> => ClearServicesData(),
  listNodeVersions: async (): Promise<NodeVersionInfo[]> =>
    (await cast<NodeVersionInfo[] | null>(ListNodeVersions())) ?? [],
  installNodeVersion: (version: string): Promise<void> =>
    InstallNodeVersion(version),
  removeNodeVersion: (version: string): Promise<void> =>
    RemoveNodeVersion(version),
  listPHPVersions: async (): Promise<PHPVersionInfo[]> =>
    (await cast<PHPVersionInfo[] | null>(ListPHPVersions())) ?? [],
  installPHPVersion: (version: string): Promise<void> =>
    InstallPHPVersion(version),
  removePHPVersion: (version: string): Promise<void> =>
    RemovePHPVersion(version),
  runtimesConfig: (): Promise<RuntimesConfigType> => cast(RuntimesConfig()),
  saveRuntimesConfig: (cfg: RuntimesConfigType): Promise<void> =>
    SaveRuntimesConfig(cfg as never),
  detectSystemNode: (): Promise<SystemRuntimeStatus> =>
    cast(DetectSystemNode()),
  detectSystemPHP: (): Promise<SystemRuntimeStatus> => cast(DetectSystemPHP()),
  terminalStart: (projectId: string): Promise<void> =>
    TerminalStart(projectId),
  terminalWrite: (projectId: string, data: string): Promise<void> =>
    TerminalWrite(projectId, data),
  terminalResize: (
    projectId: string,
    cols: number,
    rows: number,
  ): Promise<void> => TerminalResize(projectId, cols, rows),
  terminalBuffer: (projectId: string): Promise<string> =>
    TerminalBuffer(projectId),
  mailList: (start: number, limit: number): Promise<mailpit.ListResult> =>
    MailList(start, limit),
  mailGet: (id: string): Promise<mailpit.Message> => MailGet(id),
  mailSetRead: (ids: string[], read: boolean): Promise<void> =>
    MailSetRead(ids, read),
  mailDelete: (ids: string[]): Promise<void> => MailDelete(ids),
  mailDeleteAll: (): Promise<void> => MailDeleteAll(),
  mailPart: (id: string, partId: string): Promise<mailpit.PartData> =>
    MailPart(id, partId),
};

export type { Project, AnalyzeResult, Config };
