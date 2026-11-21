import { useCallback, useRef, useState } from "react";
import { Client } from "@connectrpc/connect";
import { SocializeService } from "../gen/api/proto/user_service/user_service_pb";
import { ChatEntityObj, chatMessageToChatEntity } from "../store/constants";
import { FETCHED_CHATS } from "./chat_history";

// Message IDs are UUIDv7s, which sort by time, as text too.
export const isMessageID = (id: string | undefined): id is string =>
  !!id &&
  /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id);

// What the chat has loaded of the channel shown: a stretch of its history,
// which reaches the newest message unless hasNewer.
export type HistoryWindow = {
  channel: string;
  hasOlder: boolean;
  hasNewer: boolean;
  // The oldest and newest messages loaded; "" if none.
  oldest: string;
  newest: string;
  // The newest message the user had read when the channel was opened; "" if
  // none. The first message after it from someone else is dividerBefore,
  // where a "new messages" line goes.
  lastReadAtOpen: string;
  dividerBefore: string;
};

const emptyWindow = (channel: string): HistoryWindow => ({
  channel,
  hasOlder: false,
  hasNewer: false,
  oldest: "",
  newest: "",
  lastReadAtOpen: "",
  dividerBefore: "",
});

type Page = {
  messages: Array<ChatEntityObj>;
  hasOlder: boolean;
  hasNewer: boolean;
};

// useChatHistory loads a channel's history a page at a time: opened around
// the user's read position, then older or newer pages as the user scrolls,
// or the newest again. Pages replace (open, latest) or join (older, newer)
// what the chat shows, through replace and insert.
export const useChatHistory = (
  client: Client<typeof SocializeService>,
  userID: string,
  replace: (entities: Array<ChatEntityObj>) => void,
  insert: (entities: Array<ChatEntityObj>, channel: string) => void,
) => {
  const [win, setWin] = useState<HistoryWindow>(emptyWindow(""));
  const winRef = useRef(win);
  winRef.current = win;
  const loading = useRef(false);

  const fetchPage = useCallback(
    async (req: {
      channel: string;
      beforeId?: string;
      afterId?: string;
      aroundLastRead?: boolean;
    }): Promise<Page & { lastReadId: string }> => {
      const resp = await client.getChatsForChannel({
        ...req,
        limit: FETCHED_CHATS,
      });
      return {
        messages: (resp.messages ?? []).map(chatMessageToChatEntity),
        hasOlder: !!resp.hasOlder,
        hasNewer: !!resp.hasNewer,
        lastReadId: resp.lastReadId ?? "",
      };
    },
    [client],
  );

  const bounds = (msgs: Array<ChatEntityObj>) => ({
    oldest: msgs.length ? (msgs[0].id ?? "") : "",
    newest: msgs.length ? (msgs[msgs.length - 1].id ?? "") : "",
  });

  // Opens channel around the user's read position (the newest messages if
  // there is none), and returns the window.
  const open = useCallback(
    async (channel: string): Promise<HistoryWindow> => {
      const page = await fetchPage({ channel, aroundLastRead: true });
      const readAt = page.lastReadId;
      const firstUnread = readAt
        ? page.messages.find(
            (m) => m.id !== undefined && m.id > readAt && m.senderId !== userID,
          )
        : undefined;
      const w: HistoryWindow = {
        channel,
        hasOlder: page.hasOlder,
        hasNewer: page.hasNewer,
        ...bounds(page.messages),
        lastReadAtOpen: readAt,
        dividerBefore: firstUnread?.id ?? "",
      };
      replace(page.messages);
      setWin(w);
      return w;
    },
    [fetchPage, replace, userID],
  );

  // Loads the page before the oldest message loaded. Returns whether it did.
  const loadOlder = useCallback(async (): Promise<boolean> => {
    const w = winRef.current;
    if (!w.hasOlder || !w.oldest || loading.current) {
      return false;
    }
    loading.current = true;
    try {
      const page = await fetchPage({ channel: w.channel, beforeId: w.oldest });
      if (winRef.current.channel !== w.channel) {
        return false;
      }
      insert(page.messages, w.channel);
      setWin((cur) => ({
        ...cur,
        hasOlder: page.hasOlder,
        oldest: bounds(page.messages).oldest || cur.oldest,
      }));
      return page.messages.length > 0;
    } finally {
      loading.current = false;
    }
  }, [fetchPage, insert]);

  // Loads the page after the newest message loaded.
  const loadNewer = useCallback(async (): Promise<boolean> => {
    const w = winRef.current;
    if (!w.hasNewer || !w.newest || loading.current) {
      return false;
    }
    loading.current = true;
    try {
      const page = await fetchPage({ channel: w.channel, afterId: w.newest });
      if (winRef.current.channel !== w.channel) {
        return false;
      }
      insert(page.messages, w.channel);
      setWin((cur) => ({
        ...cur,
        hasNewer: page.hasNewer,
        newest: bounds(page.messages).newest || cur.newest,
      }));
      return page.messages.length > 0;
    } finally {
      loading.current = false;
    }
  }, [fetchPage, insert]);

  // Loads the newest messages again, in place of what is loaded.
  const jumpToLatest = useCallback(async () => {
    const channel = winRef.current.channel;
    if (!channel) {
      return;
    }
    const page = await fetchPage({ channel });
    if (winRef.current.channel !== channel) {
      return;
    }
    replace(page.messages);
    setWin((cur) => ({
      ...cur,
      hasOlder: page.hasOlder,
      hasNewer: false,
      ...bounds(page.messages),
      dividerBefore: "",
    }));
  }, [fetchPage, replace]);

  // A live message of the channel shown moves the newest loaded forward,
  // unless the chat is showing older history.
  const noteLive = useCallback((id: string) => {
    setWin((cur) =>
      cur.hasNewer || !isMessageID(id) || id <= cur.newest
        ? cur
        : { ...cur, newest: id },
    );
  }, []);

  const reset = useCallback((channel: string) => {
    setWin(emptyWindow(channel));
  }, []);

  return { win, open, loadOlder, loadNewer, jumpToLatest, noteLive, reset };
};
