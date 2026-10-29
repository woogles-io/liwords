import React, { useState } from "react";

import { Link, useNavigate } from "react-router";
import "./topbar.scss";
import {
  DisconnectOutlined,
  RightOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { App, Dropdown } from "antd";
import {
  useLoginStateStoreContext,
  useResetStoreContext,
  useTournamentStoreContext,
} from "../store/store";
import { LoginModal } from "../lobby/login";
import { isClubType } from "../store/constants";
import { flashError, useClient } from "../utils/hooks/connect";
import { AuthenticationService } from "../gen/api/proto/user_service/user_service_pb";

const ext = (href: string, text: string) => (
  <a href={href} target="_blank" rel="noopener noreferrer">
    {text}
  </a>
);

const TopMenu = React.memo((props: { username: string; loggedIn: boolean }) => {
  const playWatchMenuItems = [
    { key: "omgwords", label: <Link to="/">OMGWords</Link> },
    { key: "leagues", label: <Link to="/leagues">Leagues</Link> },
    { key: "tournaments", label: <Link to="/tournaments">Tournaments</Link> },
    { key: "puzzles", label: <Link to="/puzzle">Puzzles</Link> },
    { key: "broadcasts", label: <Link to="/broadcasts">Broadcasts</Link> },
    { key: "clubs", label: <Link to="/clubs">Clubs</Link> },
  ];

  const learnMenuItems = [
    { key: "analyze", label: <Link to="/editor">Analyze a game</Link> },
    ...(props.loggedIn
      ? [
          {
            key: "collections",
            label: (
              <Link to={`/profile/${encodeURIComponent(props.username)}`}>
                My collections
              </Link>
            ),
          },
        ]
      : []),
    {
      key: "library",
      label: "Library",
      children: [
        {
          key: "articles",
          label: ext("https://blog.woogles.io/articles", "Articles"),
        },
        {
          key: "guides",
          label: ext("https://blog.woogles.io/guides", "Guides"),
        },
        { key: "manuals", label: <Link to="/docs">Feature manuals</Link> },
        {
          key: "archives",
          label: ext("https://blog.woogles.io/posts", "Archives"),
        },
      ],
    },
  ];

  const communityMenuItems = [
    { key: "blog", label: ext("https://blog.woogles.io", "Blog") },
    { key: "clubs", label: <Link to="/clubs">Clubs</Link> },
    {
      key: "discord",
      label: ext("https://discord.gg/GqkUqA7ENm", "Discord / feedback"),
    },
  ];

  const moreMenuItems = [
    {
      key: "friends",
      type: "group" as const,
      label: "Friends of Woogles",
      children: [
        { key: "aerolith", label: ext("https://aerolith.org", "Aerolith") },
        { key: "wordvault", label: ext("https://wordvault.io", "WordVault") },
        {
          key: "randomracer",
          label: ext("http://randomracer.com/", "Random Racer"),
        },
        {
          key: "wordtree",
          label: ext("https://seattlephysicstutor.com/tree.html", "Word Tree"),
        },
        {
          key: "anagrams",
          label: ext("//anagrams.mynetgear.com/", "Anagrams"),
        },
        {
          key: "licensetospell",
          label: ext(
            "https://seattlephysicstutor.com/plates.html",
            "License to Spell",
          ),
        },
        {
          key: "leaves",
          label: ext(
            "https://www.cross-tables.com/leaves.php",
            "Static Leave Evaluator",
          ),
        },
        {
          key: "breakingthegame",
          label: ext("http://breakingthegame.net", "Breaking the Game"),
        },
        {
          key: "quackle",
          label: ext("http://people.csail.mit.edu/jasonkb/quackle/", "Quackle"),
        },
      ],
    },
    { type: "divider" as const },
    { key: "donate", label: <a href="/donate">Donate</a> },
    { key: "team", label: <Link to="/team">Meet the Woogles team</Link> },
  ];

  const menus = [
    { title: "Play / Watch", items: playWatchMenuItems },
    { title: "Learn", items: learnMenuItems },
    { title: "Community", items: communityMenuItems },
    { title: "More", items: moreMenuItems },
  ];

  return (
    <div className="top-header-menu">
      {menus.map((m) => (
        <div key={m.title}>
          <Dropdown
            overlayClassName="user-menu"
            menu={{ items: m.items }}
            placement="bottom"
            trigger={["click"]}
            getPopupContainer={() =>
              document.getElementById("root") as HTMLElement
            }
          >
            <p>{m.title}</p>
          </Dropdown>
        </div>
      ))}
    </div>
  );
});

type Props = {
  tournamentID?: string;
  leagueSlug?: string;
  broadcastSlug?: string;
  nextCorresGameID?: string;
  corresGamesWaiting?: number;
};

export const TopBar = React.memo((props: Props) => {
  const { loginState } = useLoginStateStoreContext();
  const { resetStore } = useResetStoreContext();
  const { tournamentContext } = useTournamentStoreContext();
  const { username, loggedIn, connectedToSocket } = loginState;
  const [loginModalVisible, setLoginModalVisible] = useState(false);
  const authClient = useClient(AuthenticationService);
  const navigate = useNavigate();
  const { notification } = App.useApp();

  const handleLogout = async () => {
    try {
      await authClient.logout({});
      notification.info({
        message: "Success",
        description: "You have been logged out.",
      });
      resetStore();
    } catch (e) {
      flashError(e);
    }
  };

  const userMenuItems = [
    {
      label: (
        <Link to={`/profile/${encodeURIComponent(username)}`}>Profile</Link>
      ),
      key: "profile",
    },
    {
      label: "Settings",
      key: "settings",
      children: [
        {
          key: "settings-personal",
          label: <Link to="/settings/personal">Personal info</Link>,
        },
        {
          key: "settings-preferences",
          label: <Link to="/settings/preferences">Preferences</Link>,
        },
        {
          key: "settings-password",
          label: <Link to="/settings/password">Change password</Link>,
        },
        {
          key: "settings-integrations",
          label: <Link to="/settings/integrations">Integrations</Link>,
        },
        {
          key: "settings-blocked",
          label: <Link to="/settings/blocked">Blocked players</Link>,
        },
        {
          key: "settings-secret",
          label: <Link to="/settings/secret">Secret features</Link>,
        },
        {
          key: "settings-api",
          label: <Link to="/settings/api">API</Link>,
        },
        {
          key: "settings-roles",
          label: <Link to="/settings/roles">Roles &amp; permissions</Link>,
        },
      ],
    },
    {
      label: <Link to="/terms">Terms of Service</Link>,
      key: "tos",
    },
    {
      label: <a>Log out</a>,
      key: "logout",
    },
  ];

  const homeLink = props.tournamentID
    ? tournamentContext.metadata?.slug
    : props.leagueSlug
      ? `/leagues/${props.leagueSlug}`
      : props.broadcastSlug
        ? `/broadcasts/${props.broadcastSlug}`
        : "/";

  const handleNextCorresGame = () => {
    if (props.nextCorresGameID) {
      navigate(`/game/${encodeURIComponent(props.nextCorresGameID)}`);
    }
  };

  return (
    <nav className="top-header" id="main-nav">
      <div className="container">
        <Link
          to={homeLink}
          className={`logo${
            props.tournamentID
              ? " tournament-mode"
              : props.leagueSlug
                ? " league-mode"
                : props.broadcastSlug
                  ? " league-mode"
                  : ""
          }`}
        >
          <div className="site-icon-rect">
            <div className="site-icon-w">W</div>
          </div>

          <div className="site-name">Woogles.io</div>
          {props.tournamentID ? (
            <div className="tournament">
              Back to
              {isClubType(tournamentContext.metadata?.type)
                ? " Club"
                : " Tournament"}
            </div>
          ) : props.leagueSlug ? (
            <div className="tournament">Back to League</div>
          ) : props.broadcastSlug ? (
            <div className="tournament">Back to Broadcast</div>
          ) : null}
        </Link>
        {props.nextCorresGameID && (
          <div
            className="next-corres-game-topbar"
            onClick={handleNextCorresGame}
            style={{
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              textTransform: "uppercase",
              letterSpacing: "1px",
              fontWeight: "bold",
              fontSize: "14px",
              marginLeft: "24px",
              whiteSpace: "nowrap",
              flexShrink: 0,
            }}
          >
            <RightOutlined style={{ marginRight: "6px" }} /> Next game
            {props.corresGamesWaiting && props.corresGamesWaiting > 1 && (
              <span style={{ marginLeft: "6px", opacity: 0.7 }}>
                ({props.corresGamesWaiting})
              </span>
            )}
          </div>
        )}
        <TopMenu username={username} loggedIn={loggedIn} />
        {loggedIn ? (
          <div className="user-info">
            <Dropdown
              overlayClassName="user-menu"
              menu={{
                items: userMenuItems,
                onClick: ({ key }) => {
                  if (key === "logout") {
                    handleLogout();
                  }
                },
              }}
              trigger={["click"]}
              placement="bottomRight"
              getPopupContainer={() =>
                document.getElementById("root") as HTMLElement
              }
            >
              <button className="link">
                {username}
                <SettingOutlined />
              </button>
            </Dropdown>
            {!connectedToSocket ? (
              <DisconnectOutlined style={{ color: "red", marginLeft: 5 }} />
            ) : null}
          </div>
        ) : (
          <div className="user-info">
            <button className="link" onClick={() => setLoginModalVisible(true)}>
              Log In
            </button>
            <Link to="/register">
              <button className="primary">Sign Up</button>
            </Link>
            <LoginModal {...{ loginModalVisible, setLoginModalVisible }} />
          </div>
        )}
      </div>
    </nav>
  );
});
