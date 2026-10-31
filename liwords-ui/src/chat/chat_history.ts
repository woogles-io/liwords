import { ChatEntityObj, ChatEntityType } from "../store/constants";

// The most messages the server sends for a channel: GetChatsForChannel
// (socialize.go) reads the newest this many.
export const FETCHED_CHATS = 100;

// The notice at a hole the chat cannot fill. More was said while the page
// was away than the server sends again, so messages between the ones above
// it and the ones below it may be missing.
const gapNotice = (before: ChatEntityObj): ChatEntityObj => ({
  entityType: ChatEntityType.ServerMsg,
  sender: "",
  message: "Some messages sent while you were away may be missing here.",
  id: `catch-up-gap-${before.id}`,
  channel: "server",
});

// The chat with a channel's latest messages, fetched again, merged in. The
// history is oldest first, as the server sends it. Nothing is cleared, so the
// notices posted into the chat (game events, the word check, errors) stay
// where they are. A fetched message the chat lacks goes before the first
// entry newer than it. A message of the channel that the chat has between
// the oldest and the newest fetched ones, but the history does not, was
// deleted meanwhile and goes too. A full history whose oldest message is new
// to the chat, though the chat has older ones of the channel, may start
// after a hole, which a notice marks. At most max entries stay, the newest.
export const mergeChatHistory = (
  chat: Array<ChatEntityObj>,
  history: Array<ChatEntityObj>,
  channel: string,
  max: number,
): Array<ChatEntityObj> => {
  if (!history.length) {
    return chat;
  }
  const ids = new Set(history.map((m) => m.id));
  const oldest = history[0].timestamp;
  const newest = history[history.length - 1].timestamp;
  const merged = chat.filter(
    (e) =>
      e.entityType !== ChatEntityType.UserChat ||
      e.channel !== channel ||
      ids.has(e.id) ||
      e.timestamp === undefined ||
      oldest === undefined ||
      newest === undefined ||
      e.timestamp <= oldest ||
      e.timestamp >= newest,
  );
  let changed = merged.length !== chat.length;
  const first = history[0];
  const hole =
    history.length >= FETCHED_CHATS &&
    oldest !== undefined &&
    !chat.some((e) => e.id === first.id) &&
    chat.some(
      (e) =>
        e.entityType === ChatEntityType.UserChat &&
        e.channel === channel &&
        e.timestamp !== undefined &&
        e.timestamp < oldest,
    );
  const have = new Set(merged.map((e) => e.id));
  for (const m of history) {
    if (have.has(m.id)) {
      continue;
    }
    const at = merged.findIndex(
      (e) =>
        e.timestamp !== undefined &&
        m.timestamp !== undefined &&
        e.timestamp > m.timestamp,
    );
    merged.splice(at < 0 ? merged.length : at, 0, m);
    changed = true;
  }
  if (hole) {
    merged.splice(
      merged.findIndex((e) => e.id === first.id),
      0,
      gapNotice(first),
    );
  }
  if (!changed) {
    return chat;
  }
  return merged.length > max ? merged.slice(merged.length - max) : merged;
};
