import {
  CopilotProviders,
  CopilotConfig,
  SaveCopilotConfig,
  CopilotExplain,
  CopilotAsk,
  CopilotWhySlow,
} from "../wailsjs/go/bindings/Copilot";

export interface CopilotProvider {
  id: string;
  displayName: string;
  version: string;
}

export interface CopilotConfigType {
  enabled: boolean;
  provider: string;
}

export interface CopilotAnswer {
  provider: string;
  text: string;
  durationMs: number;
}

function cast<T>(p: Promise<unknown>): Promise<T> {
  return p as Promise<T>;
}

export const copilotApi = {
  providers: async (): Promise<CopilotProvider[]> =>
    (await cast<CopilotProvider[] | null>(CopilotProviders())) ?? [],
  config: (): Promise<CopilotConfigType> => cast(CopilotConfig()),
  saveConfig: (cfg: CopilotConfigType): Promise<void> =>
    SaveCopilotConfig(cfg as never),
  explain: (projectId: string): Promise<CopilotAnswer> =>
    cast(CopilotExplain(projectId)),
  ask: (projectId: string, question: string): Promise<CopilotAnswer> =>
    cast(CopilotAsk(projectId, question)),
  whySlow: (projectId: string): Promise<CopilotAnswer> =>
    cast(CopilotWhySlow(projectId)),
};
