import { useQuery } from "@connectrpc/connect-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Table } from "antd";
import moment from "moment";
import React from "react";
import { Link } from "react-router";
import { getMyTournaments } from "../gen/api/proto/tournament_service/tournament_service-TournamentService_connectquery";
import {
  TournamentMetadata,
  TType,
} from "../gen/api/proto/tournament_service/tournament_service_pb";

// The server lists at most this many, newest first.
const LISTED = 50;

type Row = {
  key: string;
  name: string;
  link: string;
  kind: string;
  start: string;
};

// What a tournament is, and whether it is played in person.
export const tournamentKind = (t: TType, irl: boolean): string => {
  const kind =
    t === TType.CLUB ? "Club" : t === TType.CHILD ? "Session" : "Tournament";
  return irl ? `${kind} (in person)` : kind;
};

// Where a tournament's page is: its slug, or failing that its id.
export const tournamentLink = (id: string, slug: string): string =>
  slug || `/tournament/${encodeURIComponent(id)}`;

// One row per tournament or club, linked to its page.
export const directedRows = (tournaments: TournamentMetadata[]): Row[] =>
  tournaments.map((t) => ({
    key: t.id,
    name: t.name || t.id,
    link: tournamentLink(t.id, t.slug),
    kind: tournamentKind(t.type, t.irlMode),
    start: t.scheduledStartTime
      ? moment(timestampDate(t.scheduledStartTime)).format("YYYY-MM-DD")
      : "",
  }));

// The tournaments and clubs a user directs, newest first, for the admin user
// lookup.
export const UserDirectedTournaments = (props: { username: string }) => {
  const { data, error, isLoading } = useQuery(
    getMyTournaments,
    { username: props.username },
    { retry: false },
  );
  const rows = directedRows(data?.tournaments ?? []);

  return (
    <>
      <h3>Tournaments and clubs they direct</h3>
      {rows.length === LISTED && <p>The newest {LISTED}.</p>}
      <Table
        dataSource={rows}
        size="small"
        pagination={false}
        loading={isLoading}
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
          { title: "Scheduled start", dataIndex: "start", key: "start" },
        ]}
      />
    </>
  );
};
