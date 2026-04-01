import React, { useMemo } from "react";
import { Button } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { TournamentsAndLeaguesContent } from "../lobby/announcements";
import "./tournaments_page.scss";

export const TournamentsPage = () => {
  const { loginState } = useLoginStateStoreContext();

  const canCreateTournament = useMemo(
    () =>
      loginState.perms.includes("toc") || loginState.perms.includes("adm"),
    [loginState.perms],
  );

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
          <TournamentsAndLeaguesContent showLeagues={false} />
        </div>
      </div>
    </div>
  );
};
