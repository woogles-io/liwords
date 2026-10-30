import React, { useEffect, useMemo, useState } from "react";
import {
  Accordion,
  Alert,
  Anchor,
  Badge,
  Box,
  Button,
  Card,
  Container,
  Group,
  List,
  Loader,
  NumberInput,
  Radio,
  Select,
  SimpleGrid,
  Stack,
  Table,
  Text,
  Title,
} from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { ConnectError } from "@connectrpc/connect";
import { Link } from "react-router";
import {
  PlannerIntent,
  PreviewPairingsResponse,
  SchemePreview,
  TournamentService,
} from "../../gen/api/proto/tournament_service/tournament_service_pb";
import { PairingMethod } from "../../gen/api/proto/ipc/tournament_pb";
import { useTournamentStoreContext } from "../../store/store";
import { useClient } from "../../utils/hooks/connect";
import classes from "./pairing_planner.module.css";
import { ApplyModal } from "./apply_modal";

// The pairing planner previews how each pre-paired scheme would pair the
// first rounds of a division, and suggests one for what the director wants.
// Nothing changes until the director uses a scheme and confirms.

type Props = {
  tournamentID: string;
  // Where "Back to the tournament" goes.
  tournamentPath: string;
};

const intents = [
  {
    value: PlannerIntent.PLANNER_FAIR_START,
    label: "A fair start, then COP",
    description:
      "Everyone gets a similar schedule, and the top seeds are kept apart until COP takes over.",
  },
  {
    value: PlannerIntent.PLANNER_WHOLE_EVENT,
    label: "The whole event",
    description: "The pre-paired rounds are every round of the division.",
  },
  {
    value: PlannerIntent.PLANNER_TWO_TEAMS,
    label: "Two teams",
    description:
      "Two teams play each other, such as Portland vs Seattle. Assign each player a team first.",
  },
];

const errorMessage = (e: unknown) =>
  e instanceof ConnectError ? e.rawMessage : String(e);

const minMax = (values: number[]) =>
  values.length ? [Math.min(...values), Math.max(...values)] : [0, 0];

const range = ([lo, hi]: number[], unit: string) => {
  const s = lo === hi ? `${lo}` : `${lo}–${hi}`;
  return `${s} ${unit}${hi === 1 ? "" : "s"}`;
};

// What a scheme's groups are called: the teams it plays against each other,
// or the Fontes groups.
const groupWord = (s: SchemePreview) =>
  s.pairingMethod === PairingMethod.INITIAL_FONTES ? "group" : "team";

const hasGroups = (s: SchemePreview) => s.groups.some((g) => g >= 0);

export const PairingPlanner = (props: Props) => {
  const { tournamentContext } = useTournamentStoreContext();
  const tClient = useClient(TournamentService);

  const divisionNames = useMemo(
    () => Object.keys(tournamentContext.divisions).sort(),
    [tournamentContext.divisions],
  );
  const [division, setDivision] = useState<string>("");
  useEffect(() => {
    if (!divisionNames.includes(division) && divisionNames.length) {
      setDivision(divisionNames[0]);
    }
  }, [division, divisionNames]);

  const players = tournamentContext.divisions[division]?.players ?? [];
  const registered = players.filter((p) => !p.suspended).length;
  const anyTeams = players.some((p) => p.team);

  const numRounds = tournamentContext.divisions[division]?.numRounds ?? 0;
  const started =
    (tournamentContext.divisions[division]?.currentRound ?? -1) >= 0;
  const [applying, setApplying] = useState<SchemePreview>();
  const [applied, setApplied] = useState("");

  const [intent, setIntent] = useState(PlannerIntent.PLANNER_FAIR_START);
  const [rounds, setRounds] = useState(3);
  // The whole event pre-pairs every round of the division.
  useEffect(() => {
    if (intent === PlannerIntent.PLANNER_WHOLE_EVENT && numRounds > 0) {
      setRounds(numRounds);
    }
  }, [intent, numRounds]);
  const [expected, setExpected] = useState<number | "">("");

  // Memoized, so the debounced value (and the request) only changes when an
  // input does.
  const inputs = useMemo(
    () => ({ division, intent, rounds, expected }),
    [division, intent, rounds, expected],
  );
  const [request] = useDebouncedValue(inputs, 300);
  const [response, setResponse] = useState<PreviewPairingsResponse>();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!request.division || !request.rounds) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    tClient
      .previewPairings({
        id: props.tournamentID,
        division: request.division,
        rounds: request.rounds,
        expectedPlayers: request.expected || 0,
        intent: request.intent,
      })
      .then((r) => {
        if (!cancelled) {
          setResponse(r);
          setError("");
        }
      })
      .catch((e) => {
        if (!cancelled) {
          setResponse(undefined);
          setError(errorMessage(e));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [request, props.tournamentID, tClient]);

  const available = response?.schemes.filter((s) => s.available) ?? [];
  const suggested = available.find((s) => s.suggested);
  const others = available.filter((s) => s !== suggested);
  const unavailable = response?.schemes.filter((s) => !s.available) ?? [];

  return (
    <Container size="lg" py="md" className={classes.planner}>
      <Stack gap="lg">
        <div>
          <Anchor component={Link} to={props.tournamentPath} size="sm">
            ← Back to the tournament
          </Anchor>
          <Title order={2} mt="xs">
            Pairing planner
          </Title>
          <Text c="dimmed" size="sm">
            See how each pre-paired scheme would pair the first rounds of a
            division, then use the one you want. Nothing changes until you
            confirm.
          </Text>
        </div>

        <Card withBorder>
          <Stack gap="md">
            <Group align="flex-end" gap="md">
              {divisionNames.length > 1 && (
                <Select
                  label="Division"
                  data={divisionNames}
                  value={division}
                  onChange={(v) => v && setDivision(v)}
                  allowDeselect={false}
                  w={180}
                />
              )}
              <NumberInput
                label="Rounds to pre-pair"
                // Not capped at the division's rounds: a division without
                // round controls yet has none, and can still be planned.
                description={
                  numRounds > 0 && (
                    <Text
                      span
                      inherit
                      c={rounds > numRounds ? "red" : undefined}
                    >
                      {rounds > numRounds
                        ? `More than the division's ${numRounds} rounds`
                        : `The division has ${numRounds} rounds`}
                    </Text>
                  )
                }
                min={1}
                max={50}
                value={rounds}
                onChange={(v) => typeof v === "number" && setRounds(v)}
                w={160}
              />
              <NumberInput
                label="Expected players"
                description={`${registered} registered so far`}
                placeholder={`${registered}`}
                min={registered}
                max={200}
                value={expected}
                onChange={(v) => setExpected(typeof v === "number" ? v : "")}
                w={180}
              />
            </Group>
            <Radio.Group
              label="What should the pre-paired rounds do?"
              value={`${intent}`}
              onChange={(v) => {
                const next = Number(v) as PlannerIntent;
                if (intent === PlannerIntent.PLANNER_WHOLE_EVENT) {
                  setRounds(3);
                }
                setIntent(next);
              }}
            >
              <SimpleGrid cols={{ base: 1, sm: 3 }} mt="xs">
                {intents.map((i) => (
                  <Radio.Card
                    key={i.value}
                    value={`${i.value}`}
                    className={classes.intent}
                    p="sm"
                  >
                    <Group wrap="nowrap" align="flex-start" gap="sm">
                      <Radio.Indicator />
                      <div>
                        <Text fw={600} size="sm">
                          {i.label}
                        </Text>
                        <Text c="dimmed" size="xs">
                          {i.description}
                        </Text>
                      </div>
                    </Group>
                  </Radio.Card>
                ))}
              </SimpleGrid>
            </Radio.Group>
            {intent === PlannerIntent.PLANNER_FAIR_START && (
              <Text size="xs" c="dimmed">
                Want to finish with King of the Hill? COP already does that. In
                the final round it pairs the prize contenders 1 vs 2, 3 vs 4,
                and so on, and in the round before it can pair 1st vs 4th, 2nd
                vs 5th and 3rd vs 6th, or keep the top four together, when the
                race is close. So you can pre-pair as many rounds as you like,
                even all but the last one or two, and let COP pair the rest.
              </Text>
            )}
            {intent === PlannerIntent.PLANNER_TWO_TEAMS && !anyTeams && (
              <Alert color="yellow" title="No teams yet">
                Give each player a team in the director tools (Players → Edit
                player, or when adding players) to see the Team Round Robin.
                Until then, the schemes below split the field by seed.
              </Alert>
            )}
          </Stack>
        </Card>

        {started && (
          <Alert color="gray">
            {division} has started, so its round controls can't change. You can
            still preview schemes here.
          </Alert>
        )}
        {applied && (
          <Alert
            color="green"
            title="Round controls set"
            withCloseButton
            onClose={() => setApplied("")}
          >
            {applied}{" "}
            <Anchor component={Link} to={props.tournamentPath} inherit>
              Back to the tournament
            </Anchor>
          </Alert>
        )}
        {error && (
          <Alert color="red" title="Couldn't preview this division">
            {error}
          </Alert>
        )}
        {loading && !response && <Loader mx="auto" />}

        {response && (
          <Box className={loading ? classes.stale : undefined}>
            <Stack gap="lg">
              {response.players.length < 2 ? (
                <Text c="dimmed">
                  Add players, or enter how many you expect, to see a preview.
                </Text>
              ) : suggested ? (
                <SchemeCard
                  scheme={suggested}
                  response={response}
                  suggested
                  defaultOpen
                  onUse={started ? undefined : setApplying}
                />
              ) : (
                <Alert color="yellow">
                  No scheme suits {rounds} pre-paired rounds of this field well.
                  Try one of the alternatives below, or compare the schemes
                  yourself.
                </Alert>
              )}

              {response.alternatives.length > 0 && (
                <Card withBorder>
                  <Text fw={600} size="sm" mb="xs">
                    Other round counts
                  </Text>
                  <Stack gap={6}>
                    {response.alternatives.map((a) => (
                      <Group key={`${a.rounds}-${a.pairingMethod}`} gap="sm">
                        <Button
                          size="compact-sm"
                          variant="default"
                          onClick={() => setRounds(a.rounds)}
                        >
                          {a.rounds} {a.rounds === 1 ? "round" : "rounds"}
                        </Button>
                        <Text size="sm">{a.reason}</Text>
                      </Group>
                    ))}
                  </Stack>
                </Card>
              )}

              {others.length > 0 && (
                <div>
                  <Title order={4} mb="xs">
                    {suggested ? "Other schemes" : "Schemes"}
                  </Title>
                  <Stack gap="sm">
                    {others.map((s) => (
                      <SchemeCard
                        key={s.name}
                        scheme={s}
                        response={response}
                        onUse={started ? undefined : setApplying}
                      />
                    ))}
                  </Stack>
                </div>
              )}

              {unavailable.length > 0 && (
                <div>
                  <Text fw={600} size="sm" mb={4}>
                    Not available here
                  </Text>
                  <List size="sm" c="dimmed">
                    {unavailable.map((s) => (
                      <List.Item key={s.name}>
                        {s.name}: {s.unavailableReason}
                      </List.Item>
                    ))}
                  </List>
                </div>
              )}
            </Stack>
          </Box>
        )}
      </Stack>
      {applying && (
        <ApplyModal
          tournamentID={props.tournamentID}
          division={division}
          scheme={applying}
          rounds={request.rounds}
          numRounds={numRounds}
          hasTeams={anyTeams}
          onClose={() => setApplying(undefined)}
          onDone={(summary) => {
            setApplying(undefined);
            setApplied(summary);
          }}
        />
      )}
    </Container>
  );
};

const SchemeCard = (props: {
  scheme: SchemePreview;
  response: PreviewPairingsResponse;
  suggested?: boolean;
  defaultOpen?: boolean;
  // Set the division's round controls from this scheme; absent once the
  // division has started.
  onUse?: (s: SchemePreview) => void;
}) => {
  const { scheme: s, response } = props;
  const stats = s.playerStats;
  const word = groupWord(s);

  const facts: string[] = [];
  if (hasGroups(s)) {
    facts.push(
      `${range(
        minMax(stats.map((p) => p.ownGroupGames)),
        "game",
      )} vs own ${word}`,
      `${range(
        minMax(stats.map((p) => p.otherGroupGames)),
        "game",
      )} vs ${word === "team" ? "the other team" : "other groups"}`,
    );
  }
  facts.push(
    `Average opponent seed ${s.minAverageOpponentSeed.toFixed(1)}–${s.maxAverageOpponentSeed.toFixed(1)}`,
  );
  if (response.topSeeds > 1) {
    facts.push(
      `${s.topSeedGames} ${s.topSeedGames === 1 ? "game" : "games"} among the top ${response.topSeeds} seeds`,
    );
  }
  if (s.repeats) {
    facts.push(`${s.repeats} repeat${s.repeats === 1 ? "" : "s"}`);
  }
  if (s.maxByes) {
    facts.push(`Up to ${range([s.maxByes, s.maxByes], "bye")} each`);
  }
  facts.push(
    s.maxFirstImbalance
      ? `Firsts within ${s.maxFirstImbalance} of even`
      : "Firsts even",
  );

  return (
    <Card
      withBorder
      className={props.suggested ? classes.suggested : undefined}
    >
      <Group gap="xs" mb={4}>
        <Text fw={700}>{s.name}</Text>
        {props.suggested && <Badge>Suggested</Badge>}
        {s.cycleLength > 0 && (
          <Badge variant="light" color="gray">
            Full cycle: {s.cycleLength}{" "}
            {s.cycleLength === 1 ? "round" : "rounds"}
          </Badge>
        )}
      </Group>
      <Text size="sm" c="dimmed" mb="xs">
        {facts.join(" · ")}
      </Text>
      {s.notes.length > 0 && (
        <List size="sm" mb="xs" spacing={2}>
          {s.notes.map((n) => (
            <List.Item key={n}>{n}</List.Item>
          ))}
        </List>
      )}
      <Accordion
        variant="contained"
        defaultValue={props.defaultOpen ? "grid" : null}
      >
        <Accordion.Item value="grid">
          <Accordion.Control>Round-by-round pairings</Accordion.Control>
          <Accordion.Panel>
            <PairingGrid scheme={s} response={response} />
          </Accordion.Panel>
        </Accordion.Item>
      </Accordion>
      {props.onUse && (
        <Group justify="flex-end" mt="sm">
          <Button
            variant={props.suggested ? "filled" : "default"}
            onClick={() => props.onUse?.(s)}
          >
            Use this scheme…
          </Button>
        </Group>
      )}
    </Card>
  );
};

// One row per player, in seed order; each cell is that round's opponent, by
// seed, and whether the player goes first.
const PairingGrid = (props: {
  scheme: SchemePreview;
  response: PreviewPairingsResponse;
}) => {
  const { scheme: s, response } = props;
  const n = response.players.length;
  const showTeams = response.players.some((p) => p.team);
  const showGroups = hasGroups(s);

  const cells = useMemo(
    () =>
      s.rounds.map((r) => {
        const row: Array<{ opp: number; first: boolean } | null> =
          Array(n).fill(null);
        for (const g of r.games) {
          row[g.first] = { opp: g.second, first: true };
          row[g.second] = { opp: g.first, first: false };
        }
        return row;
      }),
    [s.rounds, n],
  );

  const top = response.topSeeds;
  const groupLabel = (g: number) =>
    s.pairingMethod === PairingMethod.INITIAL_FONTES
      ? `${g + 1}`
      : String.fromCharCode(65 + g);

  return (
    // Native scrolling: the ScrollArea one doesn't measure its width inside
    // an accordion panel that starts open, so it showed no scrollbar.
    <Table.ScrollContainer type="native" minWidth={300 + s.rounds.length * 64}>
      <Table
        striped
        highlightOnHover
        withColumnBorders
        verticalSpacing={4}
        horizontalSpacing="xs"
        fz="xs"
        className={classes.grid}
      >
        <Table.Thead>
          <Table.Tr>
            <Table.Th>Seed</Table.Th>
            <Table.Th>Player</Table.Th>
            {showTeams && <Table.Th>Team</Table.Th>}
            {showGroups && (
              <Table.Th>{groupWord(s) === "team" ? "Side" : "Group"}</Table.Th>
            )}
            {s.rounds.map((_, i) => (
              <Table.Th key={i} ta="center">
                R{i + 1}
              </Table.Th>
            ))}
            <Table.Th ta="center">1st–2nd</Table.Th>
          </Table.Tr>
        </Table.Thead>
        <Table.Tbody>
          {response.players.map((p, i) => {
            const st = s.playerStats[i];
            return (
              <Table.Tr key={i}>
                <Table.Td>{i + 1}</Table.Td>
                <Table.Td c={p.placeholder ? "dimmed" : undefined}>
                  {p.name}
                  {p.rating > 0 && (
                    <Text span c="dimmed" fz="xs">
                      {" "}
                      {p.rating}
                    </Text>
                  )}
                </Table.Td>
                {showTeams && <Table.Td>{p.team}</Table.Td>}
                {showGroups && (
                  <Table.Td>
                    {s.groups[i] >= 0 ? groupLabel(s.groups[i]) : ""}
                  </Table.Td>
                )}
                {cells.map((row, r) => {
                  const c = row[i];
                  if (!c) {
                    return (
                      <Table.Td key={r} ta="center" c="dimmed">
                        bye
                      </Table.Td>
                    );
                  }
                  const topGame = i < top && c.opp < top;
                  return (
                    <Table.Td
                      key={r}
                      ta="center"
                      className={topGame ? classes.topGame : undefined}
                      title={`${c.first ? "Goes first" : "Goes second"} vs ${response.players[c.opp].name}`}
                    >
                      {c.opp + 1}
                      <Text span fz={10} c="dimmed">
                        {c.first ? " 1st" : " 2nd"}
                      </Text>
                    </Table.Td>
                  );
                })}
                <Table.Td ta="center">
                  {st ? `${st.firsts}–${st.seconds}` : ""}
                </Table.Td>
              </Table.Tr>
            );
          })}
        </Table.Tbody>
      </Table>
      {top > 1 && (
        <Text fz="xs" c="dimmed" mt={4}>
          Shaded cells are games between two of the top {top} seeds.
        </Text>
      )}
    </Table.ScrollContainer>
  );
};
