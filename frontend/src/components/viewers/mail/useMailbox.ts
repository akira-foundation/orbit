import { useCallback, useEffect, useState } from "react";
import { api } from "../../../api";
import { useWailsEvent } from "../../../hooks/useWailsEvent";
import type { mailpit } from "../../../../wailsjs/go/models";

type Summary = mailpit.MessageSummary;
type Message = mailpit.Message;

export function useMailbox() {
  const [messages, setMessages] = useState<Summary[]>([]);
  const [selected, setSelected] = useState<Message | null>(null);
  const [unread, setUnread] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.mailList(0, 50);
      setMessages(res.messages ?? []);
      setUnread(res.unread ?? 0);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const select = useCallback(async (id: string) => {
    const msg = await api.mailGet(id);
    setSelected(msg);
    setMessages((prev) =>
      prev.map((m) =>
        m.ID === id ? Object.assign(Object.create(Object.getPrototypeOf(m)), m, { Read: true }) : m,
      ),
    );
    setUnread((u) => Math.max(0, u - 1));
  }, []);

  const remove = useCallback(
    async (id: string) => {
      await api.mailDelete([id]);
      setSelected((s) => (s?.ID === id ? null : s));
      await refresh();
    },
    [refresh],
  );

  useWailsEvent<Summary>("mail:new", (m) => {
    setMessages((prev) => [m, ...prev]);
    setUnread((u) => u + 1);
  });
  useWailsEvent("mail:delete", () => {
    void refresh();
  });
  useWailsEvent("mail:truncate", () => {
    setMessages([]);
    setSelected(null);
    setUnread(0);
  });

  return { messages, selected, unread, loading, error, select, remove, refresh };
}
