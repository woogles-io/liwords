// Shared components and utility functions for tournament director tools

import { AutoComplete, Form, message, Select } from "antd";
import React, { useMemo } from "react";
import { TournamentPerson } from "../../../gen/api/proto/ipc/tournament_pb";
import { Division } from "../../../store/reducers/tournament_reducer";
import { useTournamentStoreContext } from "../../../store/store";

// Helper function to convert string to lowercase kebab-case
export const lowerAndJoin = (v: string): string => {
  const l = v.toLowerCase();
  return l.split(" ").join("-");
};

// Helper function to show error messages
export const showError = (msg: string) => {
  message.error({
    content: "Error: " + msg,
    duration: 5,
  });
};

// Helper function to extract username from full player ID (uuid:username)
export const username = (fullID: string) => {
  const parts = fullID.split(":");
  return parts[1];
};

// Helper function to get UUID from username in a division
export const userUUID = (username: string, divobj: Division) => {
  if (!divobj) {
    return "";
  }
  const p = divobj.players.find((p) => {
    const parts = p.id.split(":");
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return "";
  }
  return p.id.split(":")[0];
};

// Helper function to get full player ID from username
export const fullPlayerID = (username: string, divobj: Division) => {
  if (!divobj) {
    return "";
  }
  const p = divobj.players.find((p) => {
    const parts = p.id.split(":");
    const pusername = parts[1].toLowerCase();

    if (username.toLowerCase() === pusername) {
      return true;
    }
    return false;
  });
  if (!p) {
    return "";
  }
  return p.id;
};

// Component to select a division
export const DivisionSelector = (props: {
  onChange?: (value: string) => void;
  value?: string;
  exclude?: Array<string>;
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  return (
    <Select onChange={props.onChange} value={props.value}>
      {Object.keys(tournamentContext.divisions)
        .filter((d) => !props.exclude?.includes(d))
        .map((d) => (
          <Select.Option value={d} key={`div-${d}`}>
            {d}
          </Select.Option>
        ))}
    </Select>
  );
};

// Form item for division selection
export const DivisionFormItem = (props: {
  onChange?: (value: string) => void;
  value?: string;
}) => {
  return (
    <Form.Item
      name="division"
      label="Division Name"
      rules={[
        {
          required: true,
          message: "Please input division name",
        },
      ]}
    >
      <DivisionSelector onChange={props.onChange} value={props.value} />
    </Form.Item>
  );
};

// Form item for player selection with autocomplete
export const PlayersFormItem = (props: {
  name: string;
  label: string;
  division: string;
  required?: boolean;
}) => {
  const { tournamentContext } = useTournamentStoreContext();
  const thisDivPlayers = tournamentContext.divisions[props.division]?.players;
  const alphabetizedOptions = useMemo(() => {
    if (!thisDivPlayers) {
      return null;
    }
    const playersCopy = [...thisDivPlayers];
    const players = playersCopy.sort(
      (a: TournamentPerson, b: TournamentPerson) => {
        const usera = username(a.id);
        const userb = username(b.id);
        if (usera < userb) {
          return -1;
        } else if (usera > userb) {
          return 1;
        }
        return 0;
      },
    );
    return players.map((u) => ({ value: username(u.id) }));
  }, [thisDivPlayers]);

  return (
    <Form.Item
      name={props.name}
      label={props.label}
      required={props.required ?? false}
    >
      {alphabetizedOptions ? (
        <AutoComplete
          options={alphabetizedOptions}
          filterOption={(inputValue, option) =>
            option?.value.toUpperCase().indexOf(inputValue.toUpperCase()) !== -1
          }
        />
      ) : null}
    </Form.Item>
  );
};
