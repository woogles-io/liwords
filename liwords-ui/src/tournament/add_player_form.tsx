import { Button, Form, Input, InputNumber, Select } from 'antd';
import React, { useCallback } from 'react';
import axios from 'axios';

import { debounce } from '../utils/debounce';
import { toAPIUrl } from '../api/api';
import { AutoComplete } from 'antd';
import { useMountedState } from '../utils/mounted';
import { Store } from 'antd/lib/form/interface';

type user = {
  username: string;
  uuid: string;
};

type SearchResponse = {
  users: Array<user>;
};

// Players are added to a division
type playerToAdd = {
  userID: string;
  username: string;
  // some number to identify the user. Not necessarily their wolges
  // rating, it could be another rating system.
  rating: number;
};

export type playersToAdd = { [division: string]: Array<playerToAdd> };

type Props = {
  divisions: Array<string>;
  addPlayers: (p: playersToAdd) => void;
};

export const AddPlayerForm = (props: Props) => {
  const { useState } = useMountedState();

  const [usernameOptions, setUsernameOptions] = useState<Array<user>>([]);
  const [playersToAdd, setPlayersToAdd] = useState<playersToAdd>({});
  const [usersAdded, setUsersAdded] = useState<Set<string>>(new Set());
  const onUsernameSearch = useCallback((searchText: string) => {
    axios
      .post<SearchResponse>(
        toAPIUrl('user_service.AutocompleteService', 'GetCompletion'),
        {
          prefix: searchText,
        }
      )
      .then((res) => {
        setUsernameOptions(res.data.users);
      });
  }, []);

  const searchUsernameDebounced = debounce(onUsernameSearch, 300);

  const onFormSubmit = (val: Store) => {
    const playersCopy = { ...playersToAdd };
    const user = val.player.split(':');

    if (usersAdded.has(user[0])) {
      window.alert('You have already added this player');
      return;
    }

    const newUsersAdded = new Set(usersAdded);
    newUsersAdded.add(user[0]);

    let division: Array<playerToAdd>;
    if (playersCopy[val.division]) {
      division = playersCopy[val.division];
    } else {
      division = [];
    }
    playersCopy[val.division] = [
      ...division,
      {
        userID: user[0],
        username: user[1],
        rating: val.rating,
      },
    ];
    setPlayersToAdd(playersCopy);
    setUsersAdded(newUsersAdded);
  };

  return (
    <>
      <Form
        labelCol={{ span: 6 }}
        wrapperCol={{ span: 24 }}
        layout="horizontal"
        onFinish={onFormSubmit}
      >
        <Form.Item label="Player" name="player" rules={[{ required: true }]}>
          <AutoComplete
            placeholder="Add a player"
            onSearch={searchUsernameDebounced}
            style={{ width: 200 }} // fix me sory
          >
            {usernameOptions.map((user) => (
              <AutoComplete.Option
                key={user.uuid}
                value={`${user.uuid}:${user.username}`}
              >
                {user.username}
              </AutoComplete.Option>
            ))}
          </AutoComplete>
        </Form.Item>

        <Form.Item label="Rating" name="rating">
          <InputNumber min={0} max={10000} />
        </Form.Item>

        <Form.Item
          label="Division"
          name="division"
          rules={[{ required: true }]}
        >
          <Select>
            {props.divisions.map((d) => (
              <Select.Option key={d} value={d}>
                {d}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit">
            Add this player
          </Button>
        </Form.Item>
      </Form>
      <div>
        Users to add (You must Submit To Server below to save these changes):
        {props.divisions.map((d) => (
          <div key={d}>
            <h4>Division {d}</h4>
            {playersToAdd[d]?.map((p) => (
              <div key={p.userID}>
                {p.username} ({p.rating})
              </div>
            ))}
          </div>
        ))}
        <Button onClick={() => props.addPlayers(playersToAdd)}>
          Submit to Server
        </Button>
      </div>
    </>
  );
};
