import React, { useCallback, useEffect, useState } from "react";
import iconx from "../assets/icon-x.png";
import { App } from "antd";
import { Modal } from "../utils/focus_modal";
import { flashError, useClient } from "../utils/hooks/connect";
import { SocializeService } from "../gen/api/proto/user_service/user_service_pb";

type user = {
  username: string;
  uuid: string;
};

export const BlockedPlayers = React.memo(() => {
  const [blockedUsers, setBlockedUsers] = useState<Array<user>>([]);
  const [confirmModalUser, setConfirmModalUser] = useState<user | undefined>(
    undefined,
  );
  const { notification } = App.useApp();

  const socializeClient = useClient(SocializeService);

  const refreshBlocks = useCallback(() => {
    (async () => {
      try {
        const res = await socializeClient.getBlocks({});
        setBlockedUsers(res.users);
      } catch (e) {
        flashError(e);
      }
    })();
  }, [socializeClient]);

  const unblock = useCallback(
    async (user: user) => {
      try {
        await socializeClient.removeBlock({ uuid: user.uuid });
        notification.info({
          message: "Success",
          description: `${user.username} was unblocked.`,
        });
        setBlockedUsers(
          blockedUsers.filter((blockedUser) => blockedUser.uuid !== user.uuid),
        );
      } catch (e) {
        flashError(e);
      }
    },
    [blockedUsers, socializeClient, notification],
  );

  useEffect(refreshBlocks, [refreshBlocks]);

  const confirmModal =
    confirmModalUser !== undefined ? (
      <Modal
        className="confirm-unblock-modal"
        title="Unblock player"
        open={true}
        okText="Yes, unblock"
        onCancel={() => {
          setConfirmModalUser(undefined);
        }}
        onOk={() => {
          unblock(confirmModalUser);
          setConfirmModalUser(undefined);
        }}
      >
        Are you sure you want to unblock{" "}
        <span className="blocked-player">{confirmModalUser.username}</span>?
        <br />
        <br />
        If you choose to unblock them, you'll be able to see each other
        throughout Woogles.io again.
      </Modal>
    ) : null;

  const playerList =
    blockedUsers.length > 0 ? (
      blockedUsers.map((user) => {
        return (
          <div className="blocked-player" key={user.uuid}>
            {user.username}
            <img
              src={iconx}
              className="iconx"
              alt="Unblock"
              onClick={() => setConfirmModalUser(user)}
            />
          </div>
        );
      })
    ) : (
      <div className="none-blocked">You have no blocked players</div>
    );

  return (
    <>
      <div className="blocked-players">
        <h3>Blocked players list</h3>
        <div className="header">
          This is the list of players you've blocked on Woogles.io. Blocked
          players can't see that you're online, and you can't see that they're
          online.
          <br />
          <br />
          You will, however, still be able to see each others' profiles and past
          games.
        </div>
        <div className="player-list">{playerList}</div>
      </div>
      {confirmModal}
    </>
  );
});
