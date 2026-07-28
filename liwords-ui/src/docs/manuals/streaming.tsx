import React from "react";
import type { Manual } from "../types";
import { C, P, H } from "../prose";
import { OBSFieldPreview } from "../../broadcasts/OBSFieldPreview";
import type { OBSSuffix } from "../../broadcasts/constants";
import { Tag } from "antd";

/**
 * The streaming manual: broadcast slots, stream slots, and getting them into
 * OBS.
 *
 * The system is hard to explain because it is three layers of indirection, and
 * each layer exists only so the layer above it never has to change. The manual
 * leads with that chain and then never re-derives it — every walkthrough is
 * phrased in terms of it.
 */
export type StreamingSection =
  | "overview"
  | "solo"
  | "director"
  | "streaming"
  | "recipes"
  | "fields"
  | "styling"
  | "trouble";

const origin = () =>
  typeof window !== "undefined" ? window.location.origin : "https://woogles.io";

const Overview = () => (
  <>
    <H>What this is for</H>
    <P>
      An OBS browser source points at one URL and never changes. What you want
      on screen changes constantly — a new turn, a new table, a new round, a new
      tournament. This system bridges that gap with a chain of pointers, where
      each link exists so the link above it can stay still.
    </P>
    <pre className="doc-diagram">
      {`your OBS browser source   ── a URL you paste once, and never touch again
  └─ stream slot            ── yours; you choose what it follows
       └─ broadcast slot    ── the event's; its directors aim it at a table
            └─ table        ── e.g. Division A, Round 7, Table 1
                 └─ the annotated game being played there`}
    </pre>
    <P>
      Read it downward: your browser source shows whatever your stream slot
      follows; your stream slot follows whatever a broadcast slot is aimed at;
      that broadcast slot is aimed at a table; and the game at that table is
      what viewers see. Change any link and everything above it keeps working.
    </P>

    <H>The two kinds of slot</H>
    <P>
      Almost all confusion comes from these being different things with similar
      names. The short version:{" "}
      <b>a broadcast slot belongs to an event, a stream slot belongs to you.</b>
    </P>
    <table className="doc-table">
      <thead>
        <tr>
          <th />
          <th>Broadcast slot</th>
          <th>Stream slot</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>Owned by</td>
          <td>The broadcast</td>
          <td>You, your account</td>
        </tr>
        <tr>
          <td>Points at</td>
          <td>A table (division, round, table)</td>
          <td>A broadcast slot, or your own latest annotated game</td>
        </tr>
        <tr>
          <td>Answers</td>
          <td>"Which table is this event featuring?"</td>
          <td>"What is my stream showing?"</td>
        </tr>
        <tr>
          <td>Moved by</td>
          <td>The event's directors, as rounds progress</td>
          <td>You, whenever you switch what you cover</td>
        </tr>
        <tr>
          <td>URL contains</td>
          <td>
            The broadcast's slug — so it <b>changes with every event</b>
          </td>
          <td>
            Only your username — so it <b>never changes</b>
          </td>
        </tr>
        <tr>
          <td>Needs a broadcast?</td>
          <td>Yes, it lives inside one</td>
          <td>No</td>
        </tr>
      </tbody>
    </table>
    <P>
      You can put a broadcast slot URL straight into OBS and skip stream slots
      entirely. It works, and for a one-off event it is the shortest path. The
      cost is that the URL dies with the event: next month's tournament is a new
      broadcast with a new slug, and you rebuild your scene. A stream slot is
      the fix for that, and for switching between two events mid-stream.
    </P>

    <H>The three URL shapes</H>
    <table className="doc-table">
      <tbody>
        <tr>
          <td>Stream slot</td>
          <td>
            <C>
              /api/annotations/obs/user/{"<you>"}/{"<slot>"}/{"<field>"}
            </C>
          </td>
        </tr>
        <tr>
          <td>Broadcast slot</td>
          <td>
            <C>
              /api/broadcasts/obs/{"<broadcast>"}/{"<slot>"}/{"<field>"}
            </C>
          </td>
        </tr>
        <tr>
          <td>One specific game</td>
          <td>
            <C>
              /api/annotations/obs/game/{"<game-id>"}/{"<field>"}
            </C>
          </td>
        </tr>
      </tbody>
    </table>
    <P>
      You never have to type these. The <b>OBS Builder</b> button next to any
      slot builds the URL for you, previews it, and copies it.
    </P>
  </>
);

const Solo = () => (
  <>
    <H>Streaming your own games, without a broadcast</H>
    <P>
      You do not need a broadcast to stream. If you annotate games yourself —
      club night, a casual match, a small event you are running alone — a stream
      slot can simply follow whatever game you are currently annotating.
    </P>

    <H>Setup, once</H>
    <ol className="doc-ol">
      <li>
        Open any annotated game in the editor and expand{" "}
        <b>Share &amp; broadcast</b>. (Or use the director panel of a broadcast,
        if you have one — it is the same panel.)
      </li>
      <li>
        Under <b>My stream slots</b>, type a name like <C>stream1</C> and click{" "}
        <b>Add stream slot</b>. Lowercase letters, numbers and dashes.
      </li>
      <li>
        Click <b>Point at…</b> and choose <b>My latest annotated game</b>.
      </li>
      <li>
        Click <b>OBS Builder</b>, pick a field (start with <C>score</C>), style
        it, and copy the URL.
      </li>
      <li>Add it to OBS as a browser source — see below.</li>
      <li>Repeat steps 4–5 for each field you want on screen.</li>
    </ol>

    <H>Adding one to OBS</H>
    <P>
      Every field is a separate browser source, so you do this once per field.
      In OBS:
    </P>
    <ol className="doc-ol">
      <li>
        In the <b>Sources</b> panel at the bottom of the window, click <b>+</b>{" "}
        and choose <b>Browser</b>.
      </li>
      <li>
        Name it after the field — <C>score</C>, <C>last play</C> — so the
        Sources list stays readable once you have six of them. Click <b>OK</b>.
      </li>
      <li>
        In the properties dialog that opens, paste your URL into <b>URL</b>,
        replacing the placeholder address already in the box.
      </li>
      <li>
        Set <b>Width</b> and <b>Height</b> to the space you want the overlay to
        occupy. The page fills whatever box you give it and centres the text
        vertically, so size the box, not the text.
      </li>
      <li>
        Leave everything else at its default, click <b>OK</b>, then drag and
        resize the source on the canvas.
      </li>
    </ol>
    <P>
      Two settings in that dialog are worth knowing about. <b>Custom CSS</b> is
      not needed — the styling options in the OBS Builder cover the same ground
      and travel with the URL. And <b>Shutdown source when not visible</b>{" "}
      should stay <i>off</i>, so the overlay stays connected and current while
      the scene is hidden rather than reconnecting each time you cut to it.
    </P>

    <H>Every session after that</H>
    <P>
      Nothing. Create or claim an annotated game and your overlay follows it,
      updating as you enter moves.
    </P>
    <P>
      Be careful about what "latest" means here, though: the slot follows the
      annotated game you most recently <i>started</i> — created or claimed — not
      the one you happen to be typing into. Entering moves in an older game does
      not bring it back to the front. So if you keep several annotated games on
      the go at once, start them in the order you intend to stream them, or use
      a slot pointed at a broadcast slot instead, which tracks a table rather
      than a game and can be re-aimed whenever you like.
    </P>

    <H>What you get</H>
    <P>
      All the game fields — scores, unseen tiles, last play, blanks, names —
      plus <C>opponent_name</C>, which only works in this mode because it is the
      only one where "you" is a known player.
    </P>
    <P>
      The tournament fields (records, standings, spread) stay blank, because
      there is no tournament feed behind a plain annotated game. If the game
      does happen to belong to a broadcast — you claimed it from a broadcast's
      pairings table — then those fields fill in automatically. You get the
      extras when they exist and lose nothing when they don't.
    </P>
    <P>
      One caveat worth knowing: because this mode follows <i>your</i>{" "}
      annotations, it is not the right choice when several people annotate the
      same event. Whoever edits last would yank the overlay around. For that,
      use a broadcast slot — see <b>Streaming a broadcast</b>.
    </P>
  </>
);

const Director = () => (
  <>
    <H>Running a broadcast</H>
    <P>
      As a director you are answering one question for everybody downstream:{" "}
      <b>which table is this event featuring right now?</b> You answer it by
      creating broadcast slots and moving them as rounds progress. Streamers —
      including you — attach to those slots and stop worrying about tables.
    </P>

    <H>Create your slots, once per broadcast</H>
    <ol className="doc-ol">
      <li>
        Open your broadcast and find <b>OBS Slots</b> in the director panel.
      </li>
      <li>
        Give the slot a name that describes its <i>role</i>, not its current
        contents: <C>featured</C>, <C>board2</C>, <C>final</C>. It will point at
        many different tables over the event's life, so <C>smith-vs-jones</C>{" "}
        will be a lie within the hour.
      </li>
      <li>Set its starting division, round and table, then click Add slot.</li>
      <li>
        Add a second slot if you will ever show two boards at once. One slot
        drives <i>all</i> the fields for one game, so you do not need a slot per
        field — <C>featured</C> alone feeds the score, the names, the last play
        and everything else.
      </li>
    </ol>

    <H>Move them, as the event runs</H>
    <P>
      Go to the round's pairings table. Each row has a <b>Slot</b> column; click{" "}
      <b>+ Assign</b> on the table you want to feature and pick the slot. Every
      overlay attached to that slot follows instantly — no one downstream
      touches anything.
    </P>
    <P>
      If you reassign a slot away from a game that is still live, you will get a
      confirmation first, because somebody is very likely streaming it.
    </P>

    <H>Games have to be claimed</H>
    <P>
      A slot points at a <i>table</i>, but there is only something to show once
      an annotator has claimed that table and started entering moves. A slot
      aimed at an unclaimed table shows <C>(no game assigned)</C> — that is
      expected, not a fault. Add annotators in the director panel; they claim
      tables from the same pairings view.
    </P>

    <H>Who is streaming you</H>
    <P>
      A slot marked <Tag color="purple">streamed by 2</Tag> has two stream slots
      mirroring it, so deleting it would blank two people's overlays — the
      delete confirmation says so. Re-pointing it does not warn, because that is
      the normal thing to do at the end of a round and everyone mirroring it is
      meant to follow along.
    </P>
  </>
);

const Streaming = () => (
  <>
    <H>Streaming a broadcast</H>
    <P>
      Here the event's directors decide which table is featured, and you decide
      which event you are showing. Those are two different jobs, which is
      exactly why there are two layers.
    </P>

    <H>Setup</H>
    <ol className="doc-ol">
      <li>
        Open the broadcast. In the director panel, under <b>My stream slots</b>,
        create a slot — <C>stream1</C>.
      </li>
      <li>
        Click <b>Point at…</b> and choose the broadcast slot to mirror, e.g.{" "}
        <b>Albany Open: featured</b>.
      </li>
      <li>
        Use <b>OBS Builder</b> to copy a URL per field into OBS, once.
      </li>
    </ol>
    <P>
      From now on your overlay shows whatever <C>featured</C> is aimed at. When
      the directors move it to the next round's top table, you do nothing.
    </P>

    <H>You need director rights on the broadcast</H>
    <P>
      Pointing a stream slot at someone else's broadcast requires you to be a
      director of it. Ask the organisers to add you in their director panel. (If
      you would rather not be added, you can always paste the broadcast slot URL
      directly into OBS — you just give up the permanent URL.)
    </P>

    <H>Why not just use the broadcast slot URL?</H>
    <P>
      You can. The difference shows up the second time. A broadcast slot URL
      contains that broadcast's slug, so next month's event means editing every
      browser source in your scene. A stream slot URL contains only your
      username and slot name, so next month is one click on this page. See{" "}
      <b>Recipes</b> for the two situations where this matters most.
    </P>
  </>
);

const Recipes = () => (
  <>
    <H>One stream, two events running at once</H>
    <P>
      Two events are two separate broadcasts, but you have one stream and want
      to cut back and forth between them.
    </P>
    <ol className="doc-ol">
      <li>
        Make sure each event has a broadcast slot — say both are named{" "}
        <C>featured</C>.
      </li>
      <li>
        Create <i>one</i> stream slot, <C>stream1</C>, and build your OBS scene
        against it.
      </li>
      <li>
        To cut to event A: open event A, <b>Point at…</b> →{" "}
        <b>Event A: featured</b>. To cut back to B, the same on event B's page.
      </li>
    </ol>
    <P>
      Two things make this work well. First, one re-point moves your{" "}
      <i>whole scene</i> — every field shares the slot, so you are not editing
      six browser sources. Second, each event's own directors keep their{" "}
      <C>featured</C> slot up to date the entire time you are away, so cutting
      back to A lands on whatever A is featuring <i>now</i>, not the table it
      was on when you left.
    </P>
    <P>
      If you want both events on screen simultaneously rather than alternating,
      make two stream slots — <C>stream1</C> and <C>stream2</C> — and point one
      at each.
    </P>

    <H>A stream that runs every few weeks</H>
    <P>
      Different tournament each time, same OBS scene. Build the scene once
      against <C>stream1</C>. Each new event: the organisers create a broadcast
      slot, you open the broadcast and <b>Point at…</b> it. Your scene is
      already correct. You never open OBS's source settings again.
    </P>

    <H>Growing from solo to a broadcast</H>
    <P>
      Start with <C>stream1</C> following your latest annotated game. When you
      graduate to a real broadcast with several annotators, re-point that same
      slot at a broadcast slot. Because the target lives on the slot and not in
      the URL, your scene does not change. This is the main reason "follow my
      latest game" is a setting on a slot rather than a separate kind of URL.
    </P>
  </>
);

const Fields = () => (
  <>
    <H>Fields</H>
    <P>
      Each field is its own URL and its own browser source, so you can place and
      style them independently. Every field below works with any kind of slot,
      except where the last column says otherwise.
    </P>

    <H>What they look like</H>
    <P>
      These are rendered by the same component the OBS Builder previews with, at
      each field's default size, over a checkerboard standing in for
      transparency. Sample data, not a live game.
    </P>
    <div className="doc-preview-grid">
      {(
        [
          "score",
          "combined_names",
          "unseen_tiles",
          "blank1",
          "p1_spread",
          "last_play",
        ] as OBSSuffix[]
      ).map((f) => (
        <div className="doc-preview" key={f}>
          <div className="doc-preview-label">{f}</div>
          <OBSFieldPreview field={f} minHeight={64} />
        </div>
      ))}
    </div>
    <P>
      <C>last_play</C> scrolls — give it a wide, short browser source. The
      lowercase letter in <C>blank1</C> is the one that was played as a blank.
    </P>
    <table className="doc-table">
      <thead>
        <tr>
          <th>Field</th>
          <th>Shows</th>
          <th>Requires</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>
            <C>score</C>
          </td>
          <td>
            Both scores, e.g. <C>345 - 298</C>
          </td>
          <td />
        </tr>
        <tr>
          <td>
            <C>p1_score</C> / <C>p2_score</C>
          </td>
          <td>One player's score</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>p1_name</C> / <C>p2_name</C>
          </td>
          <td>One player's name</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>combined_names</C>
          </td>
          <td>
            <C>Alice Smith - Bob Jones</C>
          </td>
          <td />
        </tr>
        <tr>
          <td>
            <C>unseen_tiles</C>
          </td>
          <td>Tiles not yet seen, grouped</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>unseen_count</C>
          </td>
          <td>Count, with vowel/consonant split</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>last_play</C>
          </td>
          <td>The last play, scrolling, with definition</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>blank1</C> / <C>blank2</C>
          </td>
          <td>Words played with a blank, blank letter coloured</td>
          <td />
        </tr>
        <tr>
          <td>
            <C>p1_record</C> / <C>p2_record</C>
          </td>
          <td>
            Win-loss, e.g. <C>6-1</C>
          </td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>p1_place</C> / <C>p2_place</C>
          </td>
          <td>
            Standing, e.g. <C>2nd</C>
          </td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>p1_spread</C> / <C>p2_spread</C>
          </td>
          <td>
            Cumulative spread, e.g. <C>+245</C>
          </td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>p1_rating</C> / <C>p2_rating</C>
          </td>
          <td>Rating from the feed</td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>division</C>
          </td>
          <td>Division name</td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>tournament</C>
          </td>
          <td>Broadcast name</td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>round</C>
          </td>
          <td>
            <C>7 of 31</C>
          </td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>table</C>
          </td>
          <td>Table number</td>
          <td>tournament feed</td>
        </tr>
        <tr>
          <td>
            <C>opponent_name</C>
          </td>
          <td>Whichever player isn't you</td>
          <td>slot following your own annotations</td>
        </tr>
      </tbody>
    </table>
    <P>
      "Tournament feed" means the field only resolves when there is a broadcast
      behind the game, since the numbers come from the event's results feed.
      That covers broadcast slots, stream slots mirroring one, and a
      latest-annotation slot that happens to land on a claimed broadcast game.
    </P>

    <H>Two other endings</H>
    <P>
      Add <C>.txt</C> to any field URL for the bare value with no HTML — useful
      if you would rather style it yourself or pipe it somewhere else. And{" "}
      <C>/events</C> in place of the field name is the raw live stream the pages
      themselves listen to.
    </P>
  </>
);

const Styling = () => (
  <>
    <H>Styling</H>
    <P>
      The <b>OBS Builder</b> covers all of this with a live preview, so reach
      for it first. The options become query parameters on the URL, which you
      can also hand-edit later without rebuilding.
    </P>
    <table className="doc-table">
      <thead>
        <tr>
          <th>Option</th>
          <th>Does</th>
          <th>Default</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>
            <C>bg</C>
          </td>
          <td>
            Background colour, or <C>transparent</C>
          </td>
          <td>transparent</td>
        </tr>
        <tr>
          <td>
            <C>color</C>
          </td>
          <td>Text colour</td>
          <td>
            <C>#000000</C>
          </td>
        </tr>
        <tr>
          <td>
            <C>size</C>
          </td>
          <td>Font size in pixels</td>
          <td>varies by field</td>
        </tr>
        <tr>
          <td>
            <C>font</C>
          </td>
          <td>Font family</td>
          <td>monospace</td>
        </tr>
        <tr>
          <td>
            <C>bold</C>
          </td>
          <td>
            <C>0</C> turns bold off
          </td>
          <td>bold</td>
        </tr>
        <tr>
          <td>
            <C>align</C>
          </td>
          <td>
            <C>left</C>, <C>center</C>, <C>right</C>
          </td>
          <td>center</td>
        </tr>
        <tr>
          <td>
            <C>padding</C>
          </td>
          <td>Inner padding in pixels</td>
          <td>8</td>
        </tr>
        <tr>
          <td>
            <C>speed</C>
          </td>
          <td>
            Scroll speed, px/sec (<C>last_play</C>)
          </td>
          <td>80</td>
        </tr>
        <tr>
          <td>
            <C>blank</C>
          </td>
          <td>Colour of blank-played letters</td>
          <td>
            <C>#d33300</C>
          </td>
        </tr>
        <tr>
          <td>
            <C>wrap</C>
          </td>
          <td>
            Wrap width in characters (<C>unseen_tiles</C>)
          </td>
          <td>off</td>
        </tr>
      </tbody>
    </table>
    <P>
      Example — a large white score on a transparent background:
      <br />
      <C>
        {origin()}
        {"/api/annotations/obs/user/you/stream1/score"}
        {"?bg=transparent&color=%23ffffff&size=64"}
      </C>
    </P>

    <H>Sizing the browser source in OBS</H>
    <P>
      The page fills whatever box you give it and centres the text vertically,
      so set the browser source's width and height to the space you want the
      overlay to occupy rather than trying to match the text. Give{" "}
      <C>last_play</C> a wide, short box — it scrolls horizontally, and needs
      the room to do it.
    </P>

    <H>What a finished scene looks like</H>
    <P>
      One browser source per field, arranged around whatever you are showing the
      game on — a camera pointed at the board, a screen capture of the Woogles
      game, or a static background. A typical layout puts <C>combined_names</C>{" "}
      and <C>score</C> in a bar across the top, <C>last_play</C> as a scrolling
      ticker across the bottom, and the standings fields (<C>p1_record</C>,{" "}
      <C>p1_spread</C>) in a corner beside each player's name.
    </P>
    <P>
      Because every field is its own source, you can build that up one piece at
      a time and rearrange freely — and none of it needs revisiting when the
      event changes, provided the sources point at a stream slot.
    </P>
  </>
);

const Trouble = () => (
  <>
    <H>It says "(no game assigned)"</H>
    <P>
      The chain is intact but the far end is empty. This is the normal display
      whenever nothing resolves, so work down the chain:
    </P>
    <ul className="doc-ul">
      <li>
        <b>Stream slot not pointed.</b> It will say <i>not pointed</i> in the
        panel. Use <b>Point at…</b>.
      </li>
      <li>
        <b>Nobody has claimed the table.</b> The broadcast slot is aimed at a
        table with no annotator on it. Claim it from the pairings view, or aim
        the slot somewhere live.
      </li>
      <li>
        <b>
          Following your own annotations, but you haven't annotated anything.
        </b>{" "}
        Create an annotated game in the editor.
      </li>
    </ul>

    <H>It's showing the wrong one of my annotated games</H>
    <P>
      A slot following your latest annotation picks the game you most recently{" "}
      <i>started</i>, not the one you are currently entering moves into — so
      going back to an earlier game does not pull the overlay back to it. Either
      start that game again after the others, or point the slot at a broadcast
      slot, which follows a table and can be re-aimed at will.
    </P>

    <H>It says "target missing"</H>
    <P>
      Your stream slot mirrors a broadcast slot that has since been renamed or
      deleted by that event's directors. Pick a current one with{" "}
      <b>Point at…</b>.
    </P>

    <H>Scores work but records and standings are blank</H>
    <P>
      Those come from the tournament feed, and are matched to players{" "}
      <i>by name</i>. If the name in the annotated game does not match the name
      in the feed exactly, the standings fields can't line up. Check the
      spelling in the game against the standings, and use <b>Force poll</b> in
      the director panel if the feed itself looks stale.
    </P>
    <P>
      If <i>every</i> tournament field is blank, the game likely has no
      broadcast behind it at all — see the Fields chapter.
    </P>

    <H>The overlay is frozen on the previous game</H>
    <P>
      It shouldn't be. Whenever a slot stops resolving to a game — reassigned to
      an unclaimed table, the game unclaimed, the slot deleted — the overlay
      clears to <C>(no game assigned)</C> rather than leaving stale numbers up.
      If a browser source really is stuck, refresh it in OBS; it reconnects and
      re-reads the current state.
    </P>

    <H>An old URL stopped working</H>
    <P>
      URLs of the form{" "}
      <C>
        /api/annotations/obs/user/{"<you>"}/{"<field>"}
      </C>{" "}
      — with no slot name — have been replaced. Create a stream slot, point it
      at <b>My latest annotated game</b>, and use its URL. It behaves the same
      way, and can be re-pointed later.
    </P>

    <H>Nothing updates live</H>
    <P>
      The pages hold an open connection and update within about a second. If one
      goes stale, refresh the browser source. If they all do, something between
      OBS and the site is closing long-lived connections — a proxy or VPN is the
      usual culprit.
    </P>
  </>
);

const SECTIONS: { key: StreamingSection; label: string; body: React.FC }[] = [
  { key: "overview", label: "How it works", body: Overview },
  { key: "solo", label: "Stream your own games", body: Solo },
  { key: "director", label: "Run a broadcast", body: Director },
  { key: "streaming", label: "Stream a broadcast", body: Streaming },
  { key: "recipes", label: "Recipes", body: Recipes },
  { key: "fields", label: "Fields", body: Fields },
  { key: "styling", label: "Styling", body: Styling },
  { key: "trouble", label: "Troubleshooting", body: Trouble },
];

export const streamingManual: Manual = {
  id: "streaming",
  title: "Streaming to OBS",
  blurb:
    "Put live scores, racks, standings and the last play on your stream, and keep the URLs working when the event changes.",
  audience: "Streamers, broadcast directors and annotators",
  sections: SECTIONS,
};
