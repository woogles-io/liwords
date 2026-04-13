import React, { useEffect, useMemo, useState } from "react";
import { Button, Pagination } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import {
  TournamentsAndLeaguesContent,
  TournamentCard,
} from "../lobby/announcements";
import { useClient } from "../utils/hooks/connect";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import type { TournamentMetadata } from "../gen/api/proto/tournament_service/tournament_service_pb";
import "./tournaments_page.scss";

const PAGE_SIZE = 5;

const MyTournamentsSection = () => {
  const [myTournaments, setMyTournaments] = useState<TournamentMetadata[]>([]);
  const [page, setPage] = useState(1);
  const tournamentClient = useClient(TournamentService);

  useEffect(() => {
    (async () => {
      try {
        const resp = await tournamentClient.getMyTournaments({});
        setMyTournaments(resp.tournaments);
      } catch (error) {
        console.error("Error fetching my tournaments:", error);
      }
    })();
  }, [tournamentClient]);

  if (myTournaments.length === 0) return null;

  const start = (page - 1) * PAGE_SIZE;
  const paginated = myTournaments.slice(start, start + PAGE_SIZE);

  return (
    <div className="my-tournaments-section">
      <h2>My Tournaments</h2>
      <div className="tournaments-list">
        {paginated.map((t) => (
          <TournamentCard key={t.id} tournament={t} />
        ))}
      </div>
      {myTournaments.length > PAGE_SIZE && (
        <div className="my-tournaments-pagination">
          <Pagination
            current={page}
            pageSize={PAGE_SIZE}
            total={myTournaments.length}
            onChange={setPage}
            size="small"
            showSizeChanger={false}
          />
        </div>
      )}
    </div>
  );
};

export const TournamentsPage = () => {
  const { loginState } = useLoginStateStoreContext();

  const canCreateTournament = useMemo(
    () => loginState.perms.includes("toc") || loginState.perms.includes("adm"),
    [loginState.perms],
  );

  const isLoggedIn = !!loginState.username;

  return (
    <div>
      <TopBar />
      <div className="tournaments-page">
        <div className="tournaments-page-header">
          <h1>Tournaments</h1>
          {canCreateTournament && (
            <Link to="/new-tournament">
              <Button type="primary" icon={<PlusOutlined />} size="large">
                Create Tournament
              </Button>
            </Link>
          )}
        </div>
        <div className="tournaments-page-content">
          <TournamentsAndLeaguesContent
            showLeagues={false}
            showBroadcasts={false}
          />
        </div>
        {isLoggedIn && <MyTournamentsSection />}
      </div>
    </div>
  );
};
