import React from "react";
import { Col, Row, Card, Spin } from "antd";
import { Link } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import { getAllLeagues } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import "./leagues.scss";

export const LeaguesList = () => {
  const { data: leaguesData, isLoading } = useQuery(getAllLeagues, {
    activeOnly: true,
  });

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <div className="leagues-container">
        <h1>Leagues</h1>
        <p className="leagues-description">
          Join competitive correspondence-based leagues with skill-based
          divisions. Compete in recurring seasonal rounds with
          promotion/relegation systems.
        </p>

        {isLoading && (
          <div className="loading-container">
            <Spin size="large" />
          </div>
        )}

        {!isLoading && leaguesData && (
          <Row gutter={[16, 16]}>
            {leaguesData.leagues.map((league) => (
              <Col xs={24} sm={12} lg={8} key={league.uuid}>
                <Link to={`/leagues/${league.slug}`}>
                  <Card hoverable className="league-card" title={league.name}>
                    <p>{league.description}</p>
                    <div className="league-meta">
                      {league.settings && (
                        <>
                          <div>
                            <strong>Lexicon:</strong> {league.settings.lexicon}
                          </div>
                          <div>
                            <strong>Season Length:</strong>{" "}
                            {league.settings.seasonLengthDays} days
                          </div>
                          <div>
                            <strong>Ideal Division Size:</strong>{" "}
                            {league.settings.idealDivisionSize} players
                          </div>
                        </>
                      )}
                    </div>
                  </Card>
                </Link>
              </Col>
            ))}
          </Row>
        )}

        {!isLoading && leaguesData?.leagues.length === 0 && (
          <div className="no-leagues">
            <p>No active leagues at this time. Check back soon!</p>
          </div>
        )}
      </div>
    </>
  );
};
