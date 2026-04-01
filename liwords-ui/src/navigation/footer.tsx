import React from "react";
import { Link } from "react-router";
import { Tooltip } from "antd";

import { useLagStoreContext } from "../store/store";
import "./footer.scss";
import {
  FacebookFilled,
  InstagramFilled,
  TwitterCircleFilled,
} from "@ant-design/icons";

const Footer = React.memo(() => {
  const { currentLagMs } = useLagStoreContext();
  const currentYear = new Date().getFullYear();
  return (
    <footer>
      <div className="footer-container ">
        <Tooltip
          placement="bottomLeft"
          title={`Latency: ${currentLagMs || "..."} ms.`}
        >
          <Link to="/" className="logo">
            <div className="site-icon-rect">
              <div className="site-icon-w">W</div>
            </div>
            <div className="site-name">Woogles.io</div>
          </Link>
        </Tooltip>
        <div className="links-social">
          <a
            href="https://www.facebook.com/groups/659791751547407"
            target="_blank"
            rel="noopener noreferrer"
          >
            <FacebookFilled />
          </a>
          <a
            href="https://twitter.com/woogles_io"
            target="_blank"
            rel="noopener noreferrer"
          >
            <TwitterCircleFilled />
          </a>
          <a
            href="https://www.instagram.com/woogles.io/?hl=en"
            target="_blank"
            rel="noopener noreferrer"
          >
            <InstagramFilled />
          </a>
          <a
            href="https://discord.gg/GqkUqA7ENm"
            target="_blank"
            rel="noopener noreferrer"
          >
            <i className="fa-brands fa-discord" />
          </a>
        </div>
        <div className="links-play-learn link-group">
          <h4>Play</h4>
          <Link to="/">OMGWords</Link>
          <Link to="/leagues">Leagues</Link>
          <Link to="/puzzle">Puzzles</Link>
          <Link to="/editor">Board editor</Link>
          <a
            href="//anagrams.mynetgear.com/"
            target="_blank"
            rel="noopener noreferrer"
          >
            Anagrams
          </a>
          <a
            href="https://seattlephysicstutor.com/plates.html"
            target="_blank"
            rel="noopener noreferrer"
          >
            License to Spell
          </a>
          <h4>Study</h4>
          <a
            href="https://aerolith.org"
            target="_blank"
            rel="noopener noreferrer"
          >
            Aerolith
          </a>
          <a
            href="http://randomracer.com/"
            target="_blank"
            rel="noopener noreferrer"
          >
            Random Racer
          </a>
          <a
            href="https://seattlephysicstutor.com/tree.html"
            target="_blank"
            rel="noopener noreferrer"
          >
            Word Tree
          </a>
          <h4>Blog site</h4>
          <a href="https://blog.woogles.io/articles">Articles</a>
          <a href="https://blog.woogles.io/posts">Blog posts</a>
          <a href="https://blog.woogles.io/guides">Guides</a>
        </div>

        <div className="links-about-resources link-group">
          <h4>About</h4>
          <a href="/team">Meet the Woogles team</a>
          <a href="/terms">Terms of service</a>
          <h4>Community Resources</h4>
          <a
            href="https://www.cross-tables.com/leaves.php"
            target="_blank"
            rel="noopener noreferrer"
          >
            Static Leave Evaluator
          </a>
          <a
            href="http://breakingthegame.net"
            target="_blank"
            rel="noopener noreferrer"
          >
            Breaking the Game
          </a>
          <a
            href="http://people.csail.mit.edu/jasonkb/quackle/"
            target="_blank"
            rel="noopener noreferrer"
          >
            Quackle
          </a>
        </div>
        <div className="links-settings link-group">
          <h4>The Woogles Experience</h4>
          <a href="/profile">Profile</a>
          <a href="/settings">Settings</a>
          <a href="/clubs">Clubs</a>
          <a href="/donate">Support Woogles</a>
          <a href="/logout">Log out</a>
        </div>
        <p className="copyright">{`©2020-${currentYear} by Woogles.io`}</p>
      </div>
    </footer>
  );
});

export default Footer;
