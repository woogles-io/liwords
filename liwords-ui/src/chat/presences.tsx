import React from "react";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { moderateUser } from "../mod/moderate";
import { PresenceEntity } from "../store/constants";

type Props = {
  players: Array<PresenceEntity>;
  sendMessage?: (uuid: string, username: string) => void;
  channel: string;
  playerInfoMap?: Map<string, string>;
  onInfoTextClick?: (userId: string) => void;
  // When each username was first seen in this channel (useFirstSeen). Kept by
  // the always-mounted chat, since this list mounts only when it is shown.
  firstSeen: Map<string, number>;
};

// How long a newcomer stays highlighted.
export const ARRIVAL_HIGHLIGHT_MS = 2 * 60 * 1000;

// Records when each username was first seen in the list, and forgets anyone
// who has left (so a return counts as a new arrival). Those present when the
// list is first shown get time 0: they did not just arrive.
export const updateFirstSeen = (
  prev: Map<string, number>,
  usernames: Array<string>,
  now: number,
  initial: boolean,
): Map<string, number> => {
  const next = new Map<string, number>();
  for (const name of usernames) {
    next.set(name, prev.get(name) ?? (initial ? 0 : now));
  }
  return next;
};

// Tracks first-seen times for the logged-in players of a channel, starting
// over when the channel changes.
export const useFirstSeen = (
  players: Array<PresenceEntity>,
  channel: string,
): Map<string, number> => {
  const tracked = React.useRef<{
    channel: string;
    seen: Map<string, number>;
  } | null>(null);
  const usernames = players
    .filter((p) => p.channel === channel && !p.anon)
    .map((p) => p.username);
  // The channel's presences arrive after the chat opens, so the first
  // non-empty list counts as those already there, not as arrivals.
  const initial =
    tracked.current?.channel !== channel || tracked.current.seen.size === 0;
  const seen = updateFirstSeen(
    initial ? new Map() : tracked.current!.seen,
    usernames,
    Date.now(),
    initial,
  );
  tracked.current = { channel, seen };
  return seen;
};

export const sortByUsername = (players: Array<PresenceEntity>) =>
  [...players].sort((a, b) =>
    a.username.localeCompare(b.username, undefined, { sensitivity: "base" }),
  );

export const Presences = React.memo((props: Props) => {
  const profileLink = (player: PresenceEntity) => (
    <UsernameWithContext
      username={player.username}
      key={player.uuid}
      userID={player.uuid}
      sendMessage={props.sendMessage}
      omitSendMessage={!props.sendMessage}
      showModTools
      moderate={moderateUser}
      infoText={props.playerInfoMap?.get(player.uuid)}
      handleInfoText={
        props.onInfoTextClick
          ? () => props.onInfoTextClick!(player.uuid)
          : undefined
      }
    />
  );
  const currentChannelPresences = props.players.filter(
    (p) => p.channel === props.channel,
  );
  // Alphabetical, so a host can see who is missing; anyone who arrived in the
  // last couple of minutes is highlighted, so arrivals still stand out.
  const knownUsers = sortByUsername(
    currentChannelPresences.filter((p) => !p.anon),
  );
  const now = Date.now();
  const seen = props.firstSeen;
  const isRecent = (username: string) =>
    now - (seen.get(username) ?? 0) < ARRIVAL_HIGHLIGHT_MS;
  // Re-render when the oldest highlight should fade.
  const [, setTick] = React.useState(0);
  const nextFade = Math.min(
    ...knownUsers
      .map((u) => (seen.get(u.username) ?? 0) + ARRIVAL_HIGHLIGHT_MS - now)
      .filter((ms) => ms > 0),
  );
  React.useEffect(() => {
    if (!Number.isFinite(nextFade)) return;
    const t = setTimeout(() => setTick((n) => n + 1), nextFade);
    return () => clearTimeout(t);
  }, [nextFade]);
  const presences = knownUsers.length
    ? knownUsers.map<React.ReactNode>((u, i) => (
        <React.Fragment key={u.username}>
          {i > 0 && ", "}
          {isRecent(u.username) ? (
            <span className="recent-arrival">{profileLink(u)}</span>
          ) : (
            profileLink(u)
          )}
        </React.Fragment>
      ))
    : null;
  const anonCount = currentChannelPresences.length - knownUsers.length;
  if (!knownUsers.length) {
    return <span className="anonymous">No logged in players.</span>;
  }
  return (
    <>
      {presences}
      <span className="anonymous">
        {anonCount === 1 ? " and 1 anonymous viewer" : null}
        {anonCount > 1 ? ` and ${anonCount} anonymous viewers` : null}
      </span>
    </>
  );
});
