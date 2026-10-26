import { useQuery } from "@connectrpc/connect-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Button, Table } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router";
import { getEnteredTournaments } from "../gen/api/proto/tournament_service/tournament_service-TournamentService_connectquery";
import { EnteredTournament } from "../gen/api/proto/tournament_service/tournament_service_pb";
import { tournamentKind, tournamentLink } from "./user_directed_tournaments";

type Row = {
  key: string;
  name: string;
  link: string;
  kind: string;
  games: number;
  lastPlayed: string;
  registered: string;
};

// One row per tournament or club, linked to its page: how many games the user
// played there and when the last was, and whether they are registered in it.
export const enteredRows = (tournaments: EnteredTournament[]): Row[] =>
  tournaments.map((t) => ({
    key: t.id,
    name: t.name || t.id,
    link: tournamentLink(t.id, t.slug),
    kind: tournamentKind(t.type, t.irlMode),
    games: t.games,
    lastPlayed: t.lastPlayed
      ? moment(timestampDate(t.lastPlayed)).format("YYYY-MM-DD")
      : "",
    registered: t.registered ? "yes" : "",
  }));

// The tournaments and clubs a user played in or is registered in, for the
// admin user lookup. They are listed on request: finding the registrations
// reads the divisions of every unfinished tournament.
export const UserEnteredTournaments = (props: { username: string }) => {
  const { data, error, isFetching, refetch } = useQuery(
    getEnteredTournaments,
    { username: props.username },
    { enabled: false, retry: false },
  );
  const rows = enteredRows(data?.tournaments ?? []);

  return (
    <>
      <h3>Tournaments and clubs they entered</h3>
      <p>
        Those they played a game in, the last played first, then those not
        finished that they are registered in.
      </p>
      <Button onClick={() => refetch()} loading={isFetching}>
        List their tournaments
      </Button>
      {(data || error) && (
        <Table
          dataSource={rows}
          size="small"
          pagination={false}
          locale={{ emptyText: error ? "Error: " + error.message : "None" }}
          columns={[
            {
              title: "Name",
              key: "name",
              render: (_: unknown, row: Row) => (
                <Link to={row.link} target="_blank">
                  {row.name}
                </Link>
              ),
            },
            { title: "Kind", dataIndex: "kind", key: "kind" },
            { title: "Games", dataIndex: "games", key: "games" },
            { title: "Last game", dataIndex: "lastPlayed", key: "lastPlayed" },
            { title: "Registered", dataIndex: "registered", key: "registered" },
          ]}
        />
      )}
    </>
  );
};
