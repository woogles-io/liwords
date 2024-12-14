import React from "react";
import { UsernameWithContext } from "../shared/usernameWithContext";
import { moderateUser } from "../mod/moderate";
import { PresenceEntity } from "../store/constants";

type Props = {
  players: Array<PresenceEntity>;
  sendMessage?: (uuid: string, username: string) => void;
  channel: string;
};

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
    />
  );
  const currentChannelPresences = props.players.filter(
    (p) => p.channel === props.channel,
  );
  const knownUsers = currentChannelPresences.filter((p) => !p.anon);
  const presences = knownUsers.length
    ? knownUsers.map<React.ReactNode>((u, i) => (
        <React.Fragment key={u.username}>
          {i > 0 && ", "}
          {profileLink(u)}
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
