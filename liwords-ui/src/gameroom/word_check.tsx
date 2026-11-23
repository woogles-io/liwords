import React, { useRef, useState } from "react";
import { Button, Input, InputRef, Popover, Typography } from "antd";
import { ConnectError } from "@connectrpc/connect";
import { GameMetadataService } from "../gen/api/proto/game_service/game_service_pb";
import { useClient } from "../utils/hooks/connect";

const { Text } = Typography;

type Checked = {
  word: string;
  valid?: boolean;
  // why the server would not check it
  error?: string;
};

type WordCheckProps = { gameID: string; disabled?: boolean };

// In a VOID game, where Challenge would be: a word typed here is checked
// against the game's lexicon, which the terms allow while playing. The server
// answers only the game's players while it is in play, and logs each check.
// Several words may be typed at once, apart by spaces or commas: each is
// checked, and answered, on its own, shown as the server read it, each tile by
// its own label (naiad as NAIAD); one it would not check, as typed. The
// answers are the
// game's: they stay listed, newest first, in a list that scrolls, while the
// game's controls show Check (opening and closing the popover keeps them),
// and go when Check does (the game ends, or is examined, or the page is
// left) or another game's page opens.
export const WordCheck = (props: WordCheckProps) => (
  // another game starts afresh: no answers, the popover shut
  <GameWordCheck key={props.gameID} {...props} />
);

const GameWordCheck = (props: WordCheckProps) => {
  const client = useClient(GameMetadataService);
  const [word, setWord] = useState("");
  const [checked, setChecked] = useState<Array<Checked>>([]);
  const [open, setOpen] = useState(false);
  const inputRef = useRef<InputRef>(null);
  // the entries being asked, one after another
  const asking = useRef(Promise.resolve());

  const answer = async (asked: string): Promise<Checked> => {
    try {
      const resp = await client.checkWord({
        gameId: props.gameID,
        word: asked,
      });
      return { word: resp.word, valid: resp.valid };
    } catch (e) {
      return {
        word: asked,
        error: e instanceof ConnectError ? e.rawMessage : "could not check",
      };
    }
  };

  // The words are read from the field itself when Enter is pressed, so the
  // answers are for exactly what it shows. They are asked one at a time, and
  // so are the entries, however many words come at once; each answer goes up
  // as it comes, an entry's in the order typed, the entry above those before.
  const check = (typed: string) => {
    const words = typed.split(/[\s,]+/).filter((w) => w !== "");
    if (words.length === 0) return;
    setWord("");
    asking.current = asking.current.then(async () => {
      for (const [i, asked] of words.entries()) {
        const a = await answer(asked);
        // the entry's answers so far are the first i
        setChecked((c) => [...c.slice(0, i), a, ...c.slice(i)]);
      }
    });
  };

  // The keys typed here stay here. The popover's events still pass up to the
  // board around the game's controls, which would take a letter to place a
  // tile, Enter to play, and keep each key's character out of the field.
  // Esc closes it (on a Mac, Cmd+. comes as Esc).
  const keepKeys = (e: React.KeyboardEvent) => {
    e.stopPropagation();
    if (e.type === "keydown" && e.key === "Escape") {
      setOpen(false);
    }
  };

  const content = (
    <div className="word-check" onKeyDown={keepKeys} onKeyPress={keepKeys}>
      {/* Enter asks, and so does the Check beside the box, for a phone's
          keyboard that may have no Enter */}
      <Input.Search
        ref={inputRef}
        autoFocus
        placeholder="Word"
        enterButton="Check"
        value={word}
        onChange={(e) => setWord(e.target.value)}
        onSearch={check}
      />
      <div className="word-check-results">
        {checked.map((c, i) => (
          <div className="word-check-result" key={`${i}-${c.word}`}>
            <Text strong>{c.word}</Text>{" "}
            {c.error ? (
              <Text type="secondary">{c.error}</Text>
            ) : c.valid ? (
              <Text type="success">valid</Text>
            ) : (
              <Text type="danger">not valid</Text>
            )}
          </div>
        ))}
      </div>
    </div>
  );

  return (
    <Popover
      content={content}
      trigger="click"
      placement="top"
      open={open}
      onOpenChange={setOpen}
      // the box takes the keys as soon as it shows
      afterOpenChange={(shown) => {
        if (shown) inputRef.current?.focus();
      }}
    >
      <Button disabled={props.disabled}>Check</Button>
    </Popover>
  );
};
