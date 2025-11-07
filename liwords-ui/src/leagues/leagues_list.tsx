import React from "react";
import { Col, Row, Card, Spin, Collapse } from "antd";
import { Link } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import { getAllLeagues } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import "./leagues.scss";

const { Panel } = Collapse;

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

        <div className="leagues-memorial">
          <p>
            In memory of <strong>Elliott Manley</strong>, creator of playscrab
            and pioneer of correspondence word game leagues. We hope to continue
            his vision of competitive, accessible, fun leagues.
          </p>
        </div>

        <div className="leagues-faq">
          <h2>Frequently Asked Questions</h2>
          <Collapse accordion>
            <Panel header="What are leagues?" key="1">
              <p>
                Leagues are competitive, correspondence-based competitions where
                players compete in seasonal rounds with promotion and relegation
                systems.
              </p>
              <p>
                Each season, players in a division play{" "}
                <strong>14 games</strong> over approximately{" "}
                <strong>3 weeks</strong>. Games are correspondence-style,
                meaning you don't need to be online at the same time as your
                opponent - you make your moves when it's convenient for you.
              </p>
              <p>
                At the end of each season, top performers are promoted to higher
                divisions, while those at the bottom may be relegated to lower
                divisions, creating a dynamic competitive environment.
              </p>
            </Panel>

            <Panel header="How does promotion and relegation work?" key="2">
              <p>
                Leagues use a skill-based division system with automatic
                promotion and relegation between seasons:
              </p>
              <ul>
                <li>
                  <strong>Promotion:</strong> Top-performing players in each
                  division advance to a higher-skilled division in the next
                  season
                </li>
                <li>
                  <strong>Relegation:</strong> Lower-performing players move to
                  a lower division to compete at a more appropriate skill level
                </li>
                <li>
                  <strong>Stability:</strong> Mid-table players remain in their
                  current division
                </li>
              </ul>
              <p>
                This system ensures you're always competing against players of
                similar skill level, with opportunities to climb the ranks or
                find your competitive balance.
              </p>
            </Panel>

            <Panel header="How do time banks and timing work?" key="3">
              <p>
                League games use a correspondence time control designed to
                complete 14 games within 3 weeks while accommodating real-life
                schedules:
              </p>
              <ul>
                <li>
                  <strong>8 hours per turn:</strong> You have 8 hours to make
                  each move under normal circumstances
                </li>
                <li>
                  <strong>72-hour time bank:</strong> Each player starts with a
                  72-hour (3-day) time bank that provides flexibility
                </li>
              </ul>
              <p>
                <strong>How the time bank helps:</strong> When your 8-hour
                per-turn timer runs out, time starts consuming from your 72-hour
                time bank. This means if you're busy for a day or two and can't
                make your moves within 8 hours, you have plenty of buffer time
                available. Even if you have a hectic couple of days, you won't
                lose games on time as long as you manage your time bank wisely.
              </p>
              <p>
                This timing system strikes a balance between keeping games
                moving at a reasonable pace and giving players the flexibility
                that correspondence word games provide.
              </p>
            </Panel>
          </Collapse>
        </div>
      </div>
    </>
  );
};
