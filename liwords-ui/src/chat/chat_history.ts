import { ChatEntityObj, ChatEntityType } from "../store/constants";

// The chat with a channel's latest messages, fetched again, merged in. The
// history is oldest first, as the server sends it. Nothing is cleared, so the
// notices posted into the chat (game events, the word check, errors) stay
// where they are. A fetched message the chat lacks goes before the first
// entry newer than it. A message of the channel that the chat has between
// the oldest and the newest fetched ones, but the history does not, was
// deleted meanwhile and goes too. At most max entries stay, the newest.
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
  if (!changed) {
    return chat;
  }
  return merged.length > max ? merged.slice(merged.length - max) : merged;
};
