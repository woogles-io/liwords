import React, { useState, useCallback } from "react";
import { Card, AutoComplete, Button, App, Tag } from "antd";
import { useMutation } from "@connectrpc/connect-query";
import { useClient } from "../utils/hooks/connect";
import { useDebounce } from "../utils/debounce";
import { AutocompleteService } from "../gen/api/proto/user_service/user_service_pb";
import {
  inviteUserToLeagues,
  revokeUserFromLeagues,
} from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";

type UserSearchResult = {
  uuid: string;
  username: string;
};

type UserSearchState = {
  options: UserSearchResult[];
  selected: string;
  inputValue: string;
};

const emptySearch: UserSearchState = {
  options: [],
  selected: "",
  inputValue: "",
};

export const InviteUserToLeaguesWidget = () => {
  const acClient = useClient(AutocompleteService);
  const { notification } = App.useApp();

  const [inviteSearch, setInviteSearch] = useState<UserSearchState>(emptySearch);
  const [revokeSearch, setRevokeSearch] = useState<UserSearchState>(emptySearch);

  const inviteUserMutation = useMutation(inviteUserToLeagues, {
    onSuccess: () => {
      notification.success({
        message: "User invited successfully",
        description:
          "The user has been granted access to participate in leagues.",
        placement: "topRight",
      });
      setInviteSearch(emptySearch);
    },
    onError: (error) => {
      notification.error({
        message: "Failed to invite user",
        description: error.message || "An error occurred while inviting user",
        placement: "topRight",
      });
    },
  });

  const revokeUserMutation = useMutation(revokeUserFromLeagues, {
    onSuccess: () => {
      notification.success({
        message: "Access revoked",
        description: "The user's league access has been revoked.",
        placement: "topRight",
      });
      setRevokeSearch(emptySearch);
    },
    onError: (error) => {
      notification.error({
        message: "Failed to revoke access",
        description: error.message || "An error occurred while revoking access",
        placement: "topRight",
      });
    },
  });

  const makeUsernameSearchHandler = useCallback(
    (setter: React.Dispatch<React.SetStateAction<UserSearchState>>) =>
      async (searchQuery: string) => {
        if (!searchQuery || searchQuery.length < 2) {
          setter((s) => ({ ...s, options: [] }));
          return;
        }
        try {
          const response = await acClient.getCompletion({ prefix: searchQuery });
          const users: UserSearchResult[] = (response.users || [])
            .map((user) => ({ uuid: user.uuid, username: user.username }))
            .sort((a, b) =>
              a.username.toLowerCase().localeCompare(b.username.toLowerCase()),
            );
          setter((s) => ({ ...s, options: users }));
        } catch {
          setter((s) => ({ ...s, options: [] }));
        }
      },
    [acClient],
  );

  const inviteSearchDebounced = useDebounce(
    makeUsernameSearchHandler(setInviteSearch),
    300,
  );
  const revokeSearchDebounced = useDebounce(
    makeUsernameSearchHandler(setRevokeSearch),
    300,
  );

  const handleSelect =
    (setter: React.Dispatch<React.SetStateAction<UserSearchState>>) =>
    (data: string) => {
      const parts = data.split(":");
      setter({
        selected: data,
        inputValue: parts.length === 2 ? parts[1] : data,
        options: [],
      });
    };

  const handleAction = (
    search: UserSearchState,
    mutate: (args: { userId: string }) => void,
  ) => {
    if (!search.selected) return;
    const parts = search.selected.split(":");
    if (parts.length !== 2) {
      notification.error({
        message: "Invalid selection",
        description: "Please select a valid user from the dropdown",
        placement: "topRight",
      });
      return;
    }
    mutate({ userId: parts[0] });
  };

  const renderAutocomplete = (
    state: UserSearchState,
    setter: React.Dispatch<React.SetStateAction<UserSearchState>>,
    onSearch: (v: string) => void,
  ) => (
    <AutoComplete
      value={state.inputValue}
      placeholder="Search username..."
      onSearch={onSearch}
      onSelect={handleSelect(setter)}
      onChange={(value) => setter((s) => ({ ...s, inputValue: value, selected: "" }))}
      className="invite-autocomplete"
    >
      {state.options.map((user) => (
        <AutoComplete.Option
          key={user.uuid}
          value={`${user.uuid}:${user.username}`}
        >
          {user.username}
        </AutoComplete.Option>
      ))}
    </AutoComplete>
  );

  return (
    <Card
      className="invite-user-card"
      style={{ marginBottom: 32, marginTop: 32 }}
      title={
        <span>
          League Player Access <Tag color="blue">League Promoter</Tag>
        </span>
      }
    >
      <h3>Invite Users to Leagues</h3>
      <p className="invite-description">
        Search for a user and grant them access to participate in leagues.
      </p>
      <div className="invite-controls">
        {renderAutocomplete(inviteSearch, setInviteSearch, inviteSearchDebounced)}
        <Button
          type="primary"
          onClick={() =>
            handleAction(inviteSearch, (args) =>
              inviteUserMutation.mutate(args),
            )
          }
          loading={inviteUserMutation.isPending}
          disabled={!inviteSearch.selected}
        >
          Invite
        </Button>
      </div>

      <h3 style={{ marginTop: 24 }}>Revoke League Access</h3>
      <p className="invite-description">
        Search for a user and remove their ability to participate in leagues.
      </p>
      <div className="invite-controls">
        {renderAutocomplete(revokeSearch, setRevokeSearch, revokeSearchDebounced)}
        <Button
          danger
          onClick={() =>
            handleAction(revokeSearch, (args) =>
              revokeUserMutation.mutate(args),
            )
          }
          loading={revokeUserMutation.isPending}
          disabled={!revokeSearch.selected}
        >
          Revoke
        </Button>
      </div>
    </Card>
  );
};
