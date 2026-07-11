import { useEffect, useState } from "react";
import {
  ParkedFolders,
  AddParkedFolder,
  RemoveParkedFolder,
  ParkedProjectIDs,
} from "../wailsjs/go/bindings/Parked";

function cast<T>(p: Promise<unknown>): Promise<T> {
  return p as Promise<T>;
}

export const parkedApi = {
  folders: async (): Promise<string[]> =>
    (await cast<string[] | null>(ParkedFolders())) ?? [],
  addFolder: (path: string): Promise<void> => AddParkedFolder(path),
  removeFolder: (path: string): Promise<void> => RemoveParkedFolder(path),
  projectIds: async (): Promise<string[]> =>
    (await cast<string[] | null>(ParkedProjectIDs())) ?? [],
};

export function useParkedIds(): Set<string> {
  const [ids, setIds] = useState<Set<string>>(new Set());
  useEffect(() => {
    parkedApi.projectIds().then((xs) => setIds(new Set(xs)));
  }, []);
  return ids;
}
