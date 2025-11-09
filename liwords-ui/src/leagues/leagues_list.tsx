import React, { useMemo } from "react";
import { Col, Row, Card, Spin, Collapse } from "antd";
import { TrophyOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { useQuery } from "@connectrpc/connect-query";
import { TopBar } from "../navigation/topbar";
import { useLoginStateStoreContext } from "../store/store";
import { getAllLeagues } from "../gen/api/proto/league_service/league_service-LeagueService_connectquery";
import { getSelfRoles } from "../gen/api/proto/user_service/user_service-AuthorizationService_connectquery";
import { InviteUserToLeaguesWidget } from "./invite_user_widget";
import "./leagues.scss";

const { Panel } = Collapse;

export const LeaguesList = () => {
  const { loginState } = useLoginStateStoreContext();
  const { data: leaguesData, isLoading } = useQuery(getAllLeagues, {
    activeOnly: true,
  });

  const loggedIn = useMemo(
    () => loginState.userID && loginState.username,
    [loginState.userID, loginState.username],
  );

  // Fetch user roles for permission checks
  const { data: selfRoles } = useQuery(
    getSelfRoles,
    {},
    { enabled: !!loggedIn },
  );

  const canInviteToLeagues = useMemo(() => {
    const roles = selfRoles?.roles || [];
    return (
      roles.includes("League Promoter") ||
      roles.includes("Admin") ||
      roles.includes("Manager")
    );
  }, [selfRoles?.roles]);

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
                  <Card
                    hoverable
                    className="league-card"
                    title={
                      <span>
                        <TrophyOutlined
                          style={{ marginRight: "8px", color: "#d4af37" }}
                        />{" "}
                        {league.name}
                      </span>
                    }
                  >
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

        {/* Invite user widget - visible to league promoters and admins */}
        {loggedIn && canInviteToLeagues && <InviteUserToLeaguesWidget />}

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

            <Panel
              header="What am I committing to when I join a league?"
              key="2"
            >
              <p>
                Each league season consists of roughly 14 games, which must all
                be completed within 21 days. The turns are relatively
                fast-paced, with each turn usually taking between 6 to 8 hours.
                You have a "time bank", usually several days long, to use if you
                need extra time for a turn.
              </p>
              <p>
                However, it's important to note that if you run out of time in
                your time bank, you will forfeit the game. So while the time
                bank provides some flexibility, it's crucial to manage your time
                wisely and make your moves within the allotted time to avoid
                forfeiting games.
              </p>
              <p>
                Please make sure that you can commit the time and attention
                required to avoid forfeiting your games and keep it fun for
                everyone involved. If you forfeit too many games on time, you
                may be restricted from registering in future seasons.
              </p>
            </Panel>

            <Panel header="How does promotion and relegation work?" key="3">
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

            <Panel header="How do time banks and timing work?" key="4">
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

            <Panel header="Do I need to register for each season?" key="5">
              <p>
                <strong>Yes, registration is required for each season.</strong>{" "}
                Seasons are not auto-enrollment - you must actively register for
                each new season you wish to participate in.
              </p>
              <p>
                This allows you to take breaks when needed, whether for personal
                commitments, vacations, or simply to recharge. Since league
                games require consistent attention over several weeks, we want
                to ensure every participant is ready and committed to their
                season.
              </p>
              <p>
                When registration opens for a new season, you'll see a
                notification banner on the league page reminding you to register
                if you'd like to participate. Registration typically opens about
                a week before the season starts.
              </p>
              <p>
                <strong>Note:</strong> If you take an extended break from the
                league, you may find yourself placed in a slightly lower
                division when you return. This helps ensure you're matched with
                players at a similar current skill level and makes your comeback
                season more enjoyable and competitive.
              </p>
            </Panel>

            <Panel header="What is a League Promoter?" key="6">
              <p>
                A League Promoter is a volunteer who invites people to leagues.
                You may find that you can't register for a league by clicking
                the button. In order to ensure people know about the time
                commitment and fair play required, we want to make sure that
                players are invited by a League Promoter before they can
                register for a season. This helps ensure that everyone who joins
                is aware of the expectations and requirements of participating
                in a league season.
              </p>
              <p>
                Ask other players in the league who the League Promoters are,
                and get an invite!
              </p>
            </Panel>

            <Panel header="What do you mean by Fair Play?" key="7">
              <p>
                These are rated games in which you are not allowed to use any
                external assistance to make your moves. It is very important
                that this be adhered to, as it ruins the fun of the league if
                anyone is cheating. Woogles runs periodic anti-cheat algorithms
                that will suspend you if you are determined to be cheating.
                Let's keep it fun for everyone and play fair.
              </p>
            </Panel>
          </Collapse>
        </div>
      </div>
    </>
  );
};
