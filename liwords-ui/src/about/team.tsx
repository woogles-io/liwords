import React, { useCallback, useState } from "react";
import { TopBar } from "../navigation/topbar";
import "./about.scss";
import { Col, Dropdown, Row } from "antd";
import andy from "../assets/bio/bio_andy.jpg";
import cesar from "../assets/bio/bio_cesar.jpg";
import conrad from "../assets/bio/bio_conrad.jpg";
import doug from "../assets/bio/bio_doug.jpg";
import jesse from "../assets/bio/bio_jesse.jpg";
import josh from "../assets/bio/bio_josh.jpg";
import lola from "../assets/bio/bio_lola.jpg";
import macondo from "../assets/bio/bio_macondo.jpg";
import woogles from "../assets/bio/bio_woogles.jpg";
import benjy from "../assets/bio/bio_benjy.jpg";

export const Team = () => {
  const [bnjyMode, setBnjyMode] = useState(
    localStorage?.getItem("bnjyMode") === "true",
  );
  const toggleBnjyMode = useCallback(() => {
    const useBenjyMode = localStorage?.getItem("bnjyMode") !== "true";
    localStorage.setItem("bnjyMode", useBenjyMode ? "true" : "false");
    if (useBenjyMode) {
      document?.body?.classList?.add("bnjyMode");
    } else {
      document?.body?.classList?.remove("bnjyMode");
    }
    setBnjyMode((x) => !x);
  }, []);

  const bnjyMenuItems = [
    {
      label: (
        <span className="link plain" onClick={toggleBnjyMode}>
          {bnjyMode ? "Disable wonky tiles" : "Enable wonky tiles"}
        </span>
      ),
      key: "bnjyMode",
    },
  ];

  return (
    <>
      <Row>
        <Col span={24}>
          <TopBar />
        </Col>
      </Row>
      <Row>
        <Col span={24} className="section-head">
          <h1>Meet the team</h1>
        </Col>
      </Row>
      <div className="bios">
        <Row>
          <Col span={24} className="intro">
            <h4>We're reinventing what it means to play word games online.</h4>
            <p>
              Let’s face it, the status quo is pretty bad. After so many years
              of being inured to “average”, we realize change will only start at
              the grassroots level. We’re a group of techy word gamers with big
              dreams, and while it’ll take time to achieve them all, we believe
              we can get there.
            </p>
            <h4>Three main principles guide the creation of this community.</h4>
            <p>1) Create a great place to play word games online.</p>
            <p>
              2) Create a tool that lets people of all skill levels from all
              over the world improve at our favorite board game
            </p>
            <p>3) Build the best AI our community has ever seen.</p>
            <h4>Can I help?</h4>
            <p>
              Absolutely. Financially, you can help the team by donating. We are
              funded completely by players like you. We are registered as a
              non-profit in the state of New Jersey.{" "}
              <a href="https://docs.google.com/spreadsheets/d/1RCdyjgq-QF2OihNKXDOhW6WDQyayxhmjhMakfUNNuLQ">
                Here's our plan for the funds.
              </a>
            </p>
            <p>
              If you have skills you think we can use, please let us know. We
              have big dreams and this community can't grow without all of us.
            </p>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={cesar} alt="César Del Solar" />
              <div className="team-info">
                <h3>César Del Solar</h3>
                <p>
                  Although César hovers around the top 20 in North America in
                  Scrabble®, he loves coding even more than playing, and thus
                  hasn’t reached the heights he’s dreamed of because he spends
                  most of his free time thinking up and building new apps. He
                  built Aerolith, a word study app which is used by hundreds of
                  competitive word gamers, and has many years of experience as a
                  professional developer. His Y Combinator startup, Leftronic
                  (S2010), was acquired in 2014, and he currently works as a
                  Chief Technology Officer at a startup in the developer tools
                  space. He also likes biking, playing guitar, artisanal
                  ketchup, and hanging out with his wifie# in their Montclair
                  home.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={jesse} alt="Jesse Day" />
              <div className="team-info">
                <h3>Jesse Day</h3>
                <p>
                  Jesse has been playing competitive Scrabble® since 2004. His
                  career highlights include winning the 2019 US National
                  Championship and finishing second at the World Championship
                  twice. His lowlight was playing the phony two-letter word IR
                  in the final of US Nationals, which ended up in a viral
                  Youtube video! Professionally, Jesse works in tech as a
                  product manager/data scientist chimera, which is roughly his
                  role within the Woogles project. He likes baguettes and
                  penguins and dislikes ketchup.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={conrad} alt="Conrad Bassett-Bouchard" />
              <div className="team-info">
                <h3>Conrad Bassett-Bouchard</h3>
                <p>
                  Conrad is a product designer whose work in the tech industry
                  is used daily by millions of people around the world. He
                  started playing competitive Scrabble® at the ripe old age of
                  14, peaking competitively in his early 20s: he held the world
                  #1 ranking at 22, and won the 2014 US National Championship at
                  24. As one of the word game community’s only professional
                  designers, his goal for Woogles is to craft fun, intuitive,
                  and inclusive experiences for word gamers worldwide. He’s also
                  a heavy metal drummer, beer nerd, and former food truck cook
                  (ketchup rules; Jesse is wrong.)
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={josh} alt="Josh Castellano" />
              <div className="team-info">
                <h3>Josh Castellano</h3>
                <p>
                  Josh is a competitive Scrabble® player and the creator of
                  RandomRacer.com, a site that aggregates annotated games from
                  cross-tables.com into a variety of notable statistics for each
                  player. He hopes to use his experience with GCG parsing and
                  database management to derive a fascinating variety of
                  accessible game data from the vast repository of games played
                  and imported on Woogles. He works as a software engineer at
                  Google during the day and enjoys juggling and studying
                  Japanese in his spare time. Josh prefers vinegar-based
                  condiments.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={lola} alt="BriAnna 'Lola' McKissen" />
              <div className="team-info">
                <h3>BriAnna "Lola" McKissen</h3>
                <p>
                  As someone who has been learning to play competitive Scrabble®
                  since 2002, Lola considers herself the voice of the passionate
                  non-expert on the team. She has been coding for 3 decades,
                  spending 12 of those years on her own startups. She currently
                  works as a Principal Software Development Engineer creating
                  user interfaces used by over a million experts in their
                  fields. In 2016, she was a finalist for the Women’s Tech
                  Council annual awards. Besides Scrabble®, she's also
                  enthusiastically bad at standup comedy, playing the bass
                  guitar, singing, and painting. Ketchup is gross.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={doug} alt="Doug Brockmeier" />
              <div className="team-info">
                <h3>Doug Brockmeier</h3>
                <p>
                  Doug has been playing Scrabble® for over 30 years -- a
                  Guinness World Record. This makes him feel old, but still
                  dignified. Early on, his Grandma Elaine taught him the
                  importance of maximizing Power Tiles through crushing two-way
                  Double-Letter-Score plays for a whopping 20 points at a time!
                  As a teen, he discovered the club and tournament scene at a
                  local bookstore, where he was welcomed to compete responsibly
                  for $1 scratchoff jackpots against a wide range of Scrabble®
                  personalities who proved that scoring 300 points per game
                  wouldn't cut it anymore, kid. Now an expert player, Doug is
                  dizzily euphoric to be part of the Woogles team and believes,
                  of course, that all Woogles are good Woogles. Doug has never
                  met a condiment he didn't like.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={andy} alt="Andy Kurnia" />
              <div className="team-info">
                <h3>Andy Kurnia</h3>
                <p>
                  Andy Kurnia is a veteran technology aficionado and Woogles
                  contributor. As OMGWords is more a math game than a word game,
                  Andy approaches it by computing odd ratios and probability of
                  word occurrences! Among Andy's achievements are winning
                  several coding competitions and gold and silver medals in the
                  International Olympiad in Informatics. He has made no public
                  statement regarding his feelings about ketchup.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={benjy} alt="Ben Schoenbrun" />
              <div className="team-info">
                <h3 style={{ cursor: "pointer" }}>
                  <Dropdown
                    overlayClassName="user-menu"
                    menu={{ items: bnjyMenuItems }}
                    getPopupContainer={() =>
                      document.getElementById("root") as HTMLElement
                    }
                    overlayStyle={{
                      width: 240,
                    }}
                    placement="bottomLeft"
                    trigger={["click"]}
                  >
                    <span>Ben Schoenbrun</span>
                  </Dropdown>
                </h3>
                <p>
                  Ben Schoenbrun, better known as just "bnjy", was having fun
                  playing at his high school Scrabble® club. A quick google
                  search for a way to play online was all it took to completely
                  change his life. He dabbles in other games, but for the past
                  15 years Scrabble® has remained his one true love. He's
                  constantly thinking of new ways to grow the game and will
                  happily shout about them from his soapbox to anyone who will
                  listen. [He'll also never shut up about being on twitch back
                  in 2016, before it was cool.] He loves bad puns, and he's
                  mustered the strength to tell you all that ketchup is gross.
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={macondo} alt="Macondo" />
              <div className="team-info">
                <h3>Macondo</h3>
                <p>
                  Macondo is César's brainchild, a word game analysis engine
                  designed to outthink everything else. Macondo is open source
                  and can be used on the command line, but his creation demanded
                  a site that could match his stature, and allow his genius to
                  enlighten and teach everyone. He likes enthusiastic students,
                  treats and bingos. (Photo by Jeremy Hildebrand. Bot wrangling
                  by Martin DeMello.)
                </p>
              </div>
            </div>
          </Col>
        </Row>
        <Row>
          <Col span={24} className="bio">
            <div className="container">
              <img src={woogles} alt="Woogles the Greek Dog of Word Games" />
              <div className="team-info">
                <h3>Woogles</h3>
                <p>
                  What exactly IS Woogles? In Greek mythology, Woogles was the
                  trusty hound of Hermes, the Greek God of word games and
                  handbags (maybe not that last one, but Wikipedia has him down
                  for "trade, wealth, luck, fertility, animal husbandry, sleep,
                  language, thieves, and travel"). Legend has it that Woogles
                  would bark in warning when a phony word was played, which is
                  why word games aren't in the Olympics. He is committed to
                  inclusivity, community, and expanding the game. He hates
                  phonies.
                </p>
              </div>
            </div>
          </Col>
        </Row>
      </div>
    </>
  );
};
