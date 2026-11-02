import React, { useState } from "react";
import { Alert, Button, Group, Modal, Stack, Table, Text } from "@mantine/core";
import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import {
  DivisionRoundControlsSchema,
  PairingMethod,
  RoundControl,
  RoundControlSchema,
} from "../../gen/api/proto/ipc/tournament_pb";
import {
  SchemePreview,
  TournamentService,
} from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { useClient } from "../../utils/hooks/connect";
import { rdCtrlFromSetting } from "../director_tools/ghetto_tools/round_control_fields";

// Sets a division's round controls from a planner scheme: the scheme for the
// pre-paired rounds, then one method for the rest.

type Props = {
  tournamentID: string;
  division: string;
  scheme: SchemePreview;
  // How many rounds to pre-pair with the scheme.
  rounds: number;
  // What pairs the rounds after them, to start with.
  then: PairingMethod;
  // For the Team Round Robin: how many rounds in a row each pair plays.
  perOpponent: number;
  // The division's rounds now; 0 if it has no round controls yet.
  numRounds: number;
  // The rounds it will have.
  total: number;
  hasTeams: boolean;
  onClose: () => void;
  onDone: (summary: string) => void;
};

const methodName = (m: PairingMethod) =>
  m === PairingMethod.PAIRING_METHOD_COP
    ? "COP"
    : m === PairingMethod.WIN_GROUPS
      ? "Win Groups"
      : "Manual";

const roundsLabel = (first: number, last: number) =>
  first === last ? `Round ${first}` : `Rounds ${first}–${last}`;

export const ApplyModal = (props: Props) => {
  const tClient = useClient(TournamentService);
  const { scheme, rounds, numRounds, hasTeams } = props;

  const total = props.total;
  // Chosen on the planner page.
  const then = props.then;
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  // Teammates may only be paired by the Team Round Robin or by hand.
  const teamProblem =
    hasTeams && scheme.pairingMethod !== PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN
      ? "This division has teams, and teammates can only be paired by the Team Round Robin or by hand. Clear the teams, or use the Team Round Robin."
      : "";

  const rest = total > rounds;
  const rows: Array<[string, string]> = [[roundsLabel(1, rounds), scheme.name]];
  if (rest) {
    rows.push([roundsLabel(rounds + 1, total), methodName(then)]);
  }

  const apply = async () => {
    setError("");
    const controls: RoundControl[] = [];
    try {
      for (let r = 1; r <= total; r++) {
        const setting =
          r <= rounds
            ? create(RoundControlSchema, {
                pairingMethod: scheme.pairingMethod,
                playWithinTeam: scheme.playWithinTeam,
                gamesPerRound:
                  scheme.pairingMethod ===
                  PairingMethod.ASSIGNED_TEAM_ROUND_ROBIN
                    ? props.perOpponent
                    : 1,
              })
            : create(RoundControlSchema, { pairingMethod: then });
        controls.push(rdCtrlFromSetting(setting, total, r));
      }
      setSaving(true);
      await tClient.setRoundControls(
        create(DivisionRoundControlsSchema, {
          id: props.tournamentID,
          division: props.division,
          roundControls: controls,
        }),
      );
      props.onDone(rows.map(([r, m]) => `${r}: ${m}.`).join(" "));
    } catch (e) {
      setError(e instanceof ConnectError ? e.rawMessage : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      opened
      onClose={props.onClose}
      title={`Set round controls for ${props.division}`}
      // The title is an h2, which the app's global h2 style would enlarge.
      styles={{
        title: { fontSize: "var(--mantine-font-size-lg)", fontWeight: 700 },
      }}
      size="lg"
    >
      <Stack gap="md">
        {numRounds > 0 && total !== numRounds && (
          <Text size="sm">
            This changes the division from {numRounds} to {total} rounds.
          </Text>
        )}
        {rounds > total && (
          <Alert color="red">
            You're pre-pairing {rounds} rounds, but the division has {total}.
            Change one of them on the planner page.
          </Alert>
        )}
        <Table withTableBorder>
          <Table.Tbody>
            {rows.map(([r, m]) => (
              <Table.Tr key={r}>
                <Table.Td>{r}</Table.Td>
                <Table.Td>{m}</Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
        <Text size="sm" c="dimmed">
          This replaces the division's round controls. Woogles pairs the
          pre-paired rounds right away from the players registered now, and
          pairs them again whenever players are added or removed before the
          division starts. Methods use their default settings; change them in
          Set round controls.
        </Text>
        {teamProblem && <Alert color="yellow">{teamProblem}</Alert>}
        {error && (
          <Alert color="red" title="Couldn't set the round controls">
            {error}
          </Alert>
        )}
        <Group justify="flex-end">
          <Button variant="default" onClick={props.onClose}>
            Cancel
          </Button>
          <Button
            onClick={apply}
            loading={saving}
            disabled={!!teamProblem || rounds > total}
          >
            Set round controls
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
};
