import React, { useState, useCallback } from "react";
import { Card, AutoComplete, Button, notification } from "antd";
import { useMutation } from "@connectrpc/connect-query";
import { useClient } from "../utils/hooks/connect";
import { useDebounce } from "../utils/debounce";
import { AutocompleteService } from "../gen/api/proto/user_service/user_service_pb";
import { inviteUserToLeagues } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";

type UserSearchResult = {
  uuid: string;
  username: string;
};

export const InviteUserToLeaguesWidget = () => {
  const acClient = useClient(AutocompleteService);
  const [usernameOptions, setUsernameOptions] = useState<UserSearchResult[]>(
    [],
  );
  const [selectedUserForInvite, setSelectedUserForInvite] =
    useState<string>("");
  const [inputValue, setInputValue] = useState<string>("");

  const inviteUserMutation = useMutation(inviteUserToLeagues, {
    onSuccess: () => {
      notification.success({
        message: "User invited successfully",
        description:
          "The user has been granted access to participate in leagues.",
        placement: "topRight",
      });
      setInputValue("");
      setSelectedUserForInvite("");
      setUsernameOptions([]);
    },
    onError: (error) => {
      notification.error({
        message: "Failed to invite user",
        description: error.message || "An error occurred while inviting user",
        placement: "topRight",
      });
    },
  });

  const onUsernameSearch = useCallback(
    async (searchQuery: string) => {
      if (!searchQuery || searchQuery.length < 2) {
        setUsernameOptions([]);
        return;
      }

      try {
        const response = await acClient.getCompletion({
          prefix: searchQuery,
        });
        const users: UserSearchResult[] = (response.users || []).map(
          (user) => ({
            uuid: user.uuid,
            username: user.username,
          }),
        );
        setUsernameOptions(users);
      } catch (error) {
        console.error("Error searching usernames:", error);
        setUsernameOptions([]);
      }
    },
    [acClient],
  );

  const searchUsernameDebounced = useDebounce(onUsernameSearch, 300);

  const handleUsernameSelect = useCallback((data: string) => {
    setSelectedUserForInvite(data);
    // Extract just the username from "uuid:username" format
    const parts = data.split(":");
    if (parts.length === 2) {
      setInputValue(parts[1]); // Set display to just username
    }
  }, []);

  const handleInviteUser = () => {
    if (!selectedUserForInvite) return;

    const parts = selectedUserForInvite.split(":");
    if (parts.length !== 2) {
      notification.error({
        message: "Invalid selection",
        description: "Please select a valid user from the dropdown",
        placement: "topRight",
      });
      return;
    }

    const [userId] = parts;
    inviteUserMutation.mutate({
      userId,
    });
  };

  return (
    <Card
      className="invite-user-card"
      style={{ marginBottom: 32, marginTop: 32 }}
    >
      <h3>Invite Users to Leagues</h3>
      <p className="invite-description">
        Search for a user and grant them access to participate in leagues.
      </p>
      <div className="invite-controls">
        <AutoComplete
          value={inputValue}
          placeholder="Search username..."
          onSearch={searchUsernameDebounced}
          onSelect={handleUsernameSelect}
          onChange={(value) => setInputValue(value)}
          className="invite-autocomplete"
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
        <Button
          type="primary"
          onClick={handleInviteUser}
          loading={inviteUserMutation.isPending}
          disabled={!selectedUserForInvite}
        >
          Invite
        </Button>
      </div>
    </Card>
  );
};
