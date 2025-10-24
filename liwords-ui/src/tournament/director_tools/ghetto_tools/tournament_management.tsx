// Tournament lifecycle management forms

import { DeleteOutlined } from "@ant-design/icons";
import { Button, Divider, Form, message, Popconfirm, Switch } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useEffect, useMemo, useState } from "react";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { singularCount } from "../../../utils/plural";
import { username } from "./shared";

export const UnstartTournament = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  const onSubmit = async (vals: Store) => {
    try {
      await tClient.unstartTournament({ id: props.tournamentID });
      window.location.reload();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form form={form} onFinish={onSubmit}>
      <div className="readable-text-color">
        This button will <strong style={{ color: "red" }}>DELETE</strong> all
        tournament game results. It is as if it had never started. IT IS NOT
        UNDOABLE. Once you click it, the tournament will RESET!
      </div>

      <Form.Item>
        <Button htmlType="submit" type="primary" danger>
          Unstart this tournament
        </Button>
      </Form.Item>
    </Form>
  );
};

export const UnfinishTournament = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();

  const onSubmit = async (vals: Store) => {
    try {
      await tClient.unfinishTournament({ id: props.tournamentID });
      window.location.reload();
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form form={form} onFinish={onSubmit}>
      <div className="readable-text-color">
        This button will unfinish a tournament that has already finished. This
        allows you to edit scores or even more.
      </div>
      <Form.Item>
        <Button htmlType="submit" type="primary">
          Unfinish this tournament
        </Button>
      </Form.Item>
    </Form>
  );
};

export const ManageCheckIns = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();

  const tClient = useClient(TournamentService);
  const [form] = Form.useForm();
  const [checkinsOpen, setCheckinsOpen] = useState(false); // Actual state from metadata
  const [desiredCheckinsState, setDesiredCheckinsState] = useState(false); // Desired state for the button
  const [registrationsOpen, setRegistrationsOpen] = useState(false); // Actual state from metadata
  const [desiredRegistrationsState, setDesiredRegistrationsState] =
    useState(false); // Desired state for the button

  const setCheckinsState = async (vals: Store) => {
    if (!vals.checkinsOpen) {
      try {
        await tClient.closeCheckins({
          id: props.tournamentID,
        });
        message.info({
          content:
            "Closed check-ins successfully. If you would like to delete players who have not checked in, please click the Delete Unchecked-In Players button.",
          duration: 10,
        });
      } catch (e) {
        flashError(e);
      }
    } else {
      try {
        await tClient.openCheckins({
          id: props.tournamentID,
        });
        message.info({
          content: "Opened check-ins successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  const setRegistrationState = async (vals: Store) => {
    if (!vals.registrationOpen) {
      try {
        await tClient.closeRegistration({
          id: props.tournamentID,
        });
        message.info({
          content: "Closed registration successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    } else {
      try {
        await tClient.openRegistration({
          id: props.tournamentID,
        });
        message.info({
          content: "Opened registration successfully.",
          duration: 3,
        });
      } catch (e) {
        flashError(e);
      }
    }
  };

  useEffect(() => {
    const metadata = tournamentContext.metadata;
    setCheckinsOpen(metadata.checkinsOpen);
    setDesiredCheckinsState(metadata.checkinsOpen); // Initialize desired state to match actual state
    setRegistrationsOpen(metadata.registrationOpen);
    setDesiredRegistrationsState(metadata.registrationOpen); // Initialize desired state to match actual state
    form.setFieldsValue({
      checkinsOpen: metadata.checkinsOpen,
      registrationOpen: metadata.registrationOpen,
    });
  }, [form, tournamentContext.metadata]);

  const uncheckedText = useMemo(() => {
    const uncheckedInPlayers = Object.values(
      tournamentContext.divisions,
    ).flatMap((division) =>
      division.players.filter((player) => !player.checkedIn),
    );

    const uncheckedInCount = uncheckedInPlayers.length;

    if (uncheckedInCount > 0) {
      const uncheckedInNames = uncheckedInPlayers
        .slice(0, 3)
        .map((player) => username(player.id))
        .join(", ");

      const remainingCount = uncheckedInCount - 3;

      return (
        <>
          <div>
            <strong>
              Players who have not checked in:{" "}
              {remainingCount > 0
                ? `${uncheckedInNames}, and ${remainingCount} more...`
                : uncheckedInNames}
            </strong>
          </div>
          <div style={{ marginTop: 8 }}>
            To delete these players from the tournament, click this button,
            after closing check-ins:
          </div>
          <div>
            <Popconfirm
              title="Are you sure you want to delete these players from the tournament? This cannot be undone."
              onConfirm={async () => {
                try {
                  await tClient.removeAllPlayersNotCheckedIn({
                    id: props.tournamentID,
                  });
                  message.info({
                    content: "Deleted unchecked-in players successfully.",
                    duration: 3,
                  });
                } catch (e) {
                  flashError(e);
                }
              }}
            >
              <Button
                type="primary"
                danger
                disabled={tournamentContext.metadata.checkinsOpen}
              >
                <DeleteOutlined />
                Delete&nbsp;
                {singularCount(uncheckedInCount, "player", "players")}
              </Button>
            </Popconfirm>
          </div>
        </>
      );
    } else {
      return <div>All players have checked in.</div>;
    }
  }, [
    tournamentContext.divisions,
    tournamentContext.metadata.checkinsOpen,
    tClient,
    props.tournamentID,
  ]);

  return (
    <Form form={form}>
      <h3>Check-ins</h3>
      <div>Check-ins are currently: {checkinsOpen ? "Open" : "Closed"}</div>
      <Form.Item name="checkinsOpen" label="Allow players to check in">
        <Switch
          checked={desiredCheckinsState}
          onChange={(checked) => setDesiredCheckinsState(checked)}
        />
      </Form.Item>

      <Form.Item>
        <Button
          type="primary"
          onClick={() => setCheckinsState(form.getFieldsValue())}
          disabled={checkinsOpen === desiredCheckinsState}
        >
          {desiredCheckinsState ? "Open" : "Close"} check-ins
        </Button>
      </Form.Item>
      <div style={{ fontSize: "12px", marginBottom: "8px" }}>
        If check-ins are on, players can check in to the tournament. If they
        don't check in, they can be removed from the tournament.
      </div>
      {uncheckedText}
      <Divider />
      <h3>Clear all check-ins</h3>
      <div style={{ fontSize: "12px", marginBottom: "8px" }}>
        This will uncheck all players without deleting them. Useful for
        resetting check-ins for a new session or day.
      </div>
      <Form.Item>
        <Popconfirm
          title="Are you sure you want to clear all check-ins? Players will remain registered but will need to check in again."
          onConfirm={async () => {
            try {
              await tClient.uncheckAllIn({
                id: props.tournamentID,
              });
              message.info({
                content: "All check-ins cleared successfully.",
                duration: 3,
              });
            } catch (e) {
              flashError(e);
            }
          }}
        >
          <Button type="default">Clear all check-ins</Button>
        </Popconfirm>
      </Form.Item>
      <Divider />
      <h3>Registrations</h3>

      <div>
        Registration is currently: {registrationsOpen ? "Open" : "Closed"}
      </div>

      <div style={{ fontSize: "12px", marginBottom: "8px" }}>
        If self-register is on, players can register themselves for the
        tournament and choose their division. Otherwise, you have to add players
        before they can check in.
      </div>
      <Form.Item name="registrationOpen" label="Allow players to self-register">
        <Switch
          checked={desiredRegistrationsState}
          onChange={(checked) => setDesiredRegistrationsState(checked)}
        />
      </Form.Item>
      <Form.Item>
        <Button
          type="primary"
          onClick={() => setRegistrationState(form.getFieldsValue())}
          disabled={registrationsOpen === desiredRegistrationsState}
        >
          {desiredRegistrationsState ? "Open" : "Close"} registration
        </Button>
      </Form.Item>
    </Form>
  );
};
