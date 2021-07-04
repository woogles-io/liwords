import React, { useCallback, useEffect } from 'react';
import axios, { AxiosError } from 'axios';
import { toAPIUrl } from '../api/api';
import { useMountedState } from '../utils/mounted';
import iconx from '../assets/icon-x.png';
import { notification, Modal } from 'antd';

type user = {
  username: string;
  uuid: string;
};

type Props = {};

const errorCatcher = (e: AxiosError) => {
  console.log('ERROR');
  if (e.response) {
    notification.warning({
      message: 'Fetch Error',
      description: e.response.data.msg,
      duration: 4,
    });
  }
};

export const BlockedPlayers = React.memo((props: Props) => {
  const { useState } = useMountedState();
  const [blockedUsers, setBlockedUsers] = useState<Array<user>>([]);
  const [confirmModalUser, setConfirmModalUser] = useState<user | undefined>(
    undefined
  );

  const refreshBlocks = useCallback(() => {
    axios
      .post(
        toAPIUrl('user_service.SocializeService', 'GetBlocks'),
        {},
        { withCredentials: true }
      )
      .then((res) => {
        setBlockedUsers(res.data.users);
      })
      .catch(errorCatcher);
  }, []);

  const unblock = useCallback(
    (user) => {
      axios
        .post(
          toAPIUrl('user_service.SocializeService', `RemoveBlock`),
          {
            uuid: user.uuid,
          },
          { withCredentials: true }
        )
        .then(() => {
          notification.info({
            message: 'Success',
            description: user.username + ' was unblocked.',
          });
          setBlockedUsers(
            blockedUsers.filter((blockedUser) => blockedUser.uuid !== user.uuid)
          );
        })
        .catch(errorCatcher);
    },
    [blockedUsers]
  );

  useEffect(refreshBlocks, [refreshBlocks]);

  const confirmModal =
    confirmModalUser !== undefined ? (
      <Modal
        className="confirm-unblock-modal"
        title="Unblock player"
        visible={true}
        okText="Yes, unblock"
        onCancel={() => {
          setConfirmModalUser(undefined);
        }}
        onOk={() => {
          unblock(confirmModalUser);
          setConfirmModalUser(undefined);
        }}
      >
        Are you sure you want to unblock{' '}
        <span className="blocked-player">{confirmModalUser.username}</span>?
        <br />
        <br />
        If you choose to unblock them, you’ll be able to see each other
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
          This is the list of players you’ve blocked on Woogles.io. Blocked
          players can’t see that you’re online, and you can’t see that they’re
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
