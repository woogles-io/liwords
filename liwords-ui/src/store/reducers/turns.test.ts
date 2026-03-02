import { create } from "@bufbuild/protobuf";
import {
  GameEvent_Type,
  GameEventSchema,
} from "../../gen/api/proto/vendored/macondo/macondo_pb";
import { gameEventsToTurns } from "./turns";

it("test turns simple", () => {
  const evt1 = create(GameEventSchema, {
    playerIndex: 1,
    rack: "?AEELRX",
    cumulative: 92,
    row: 7,
    column: 7,
    position: "8H",
    playedTiles: "RELAXEs",
    score: 92,
  });

  const evt2 = create(GameEventSchema, {
    playerIndex: 1,
    type: GameEvent_Type.CHALLENGE_BONUS,
    cumulative: 97,
    bonus: 5,
  });

  const turns = gameEventsToTurns([evt1, evt2]);
  expect(turns).toStrictEqual([
    {
      events: [evt1, evt2],
      firstEvtIdx: 0,
    },
  ]);
});

it("test turns simple 2", () => {
  const evt1 = create(GameEventSchema, {
    playerIndex: 1,
    rack: "?AEELRX",
    cumulative: 92,
    row: 7,
    column: 7,
    position: "8H",
    playedTiles: "RELAXEs",
    score: 92,
  });

  const evt2 = create(GameEventSchema, {
    playerIndex: 1,
    type: GameEvent_Type.CHALLENGE_BONUS,
    cumulative: 97,
    bonus: 5,
  });

  const evt3 = create(GameEventSchema, {
    playerIndex: 0,
    rack: "ABCDEFG",
    cumulative: 38,
    row: 6,
    column: 12,
    position: "M7",
    playedTiles: "F.EDBAG",
    score: 38,
  });

  const turns = gameEventsToTurns([evt1, evt2, evt3]);
  expect(turns.length).toBe(2);
  expect(turns).toStrictEqual([
    {
      events: [evt1, evt2],
      firstEvtIdx: 0,
    },
    {
      events: [evt3],
      firstEvtIdx: 2,
    },
  ]);
});

it("test turns simple 3", () => {
  const evt1 = create(GameEventSchema, {
    playerIndex: 1,
    rack: "?AEELRX",
    cumulative: 92,
    row: 7,
    column: 7,
    position: "8H",
    playedTiles: "RELAXEs",
    score: 92,
  });

  const evt2 = create(GameEventSchema, {
    playerIndex: 1,
    type: GameEvent_Type.CHALLENGE_BONUS,
    cumulative: 97,
    bonus: 5,
  });

  const evt3 = create(GameEventSchema, {
    playerIndex: 0,
    rack: "ABCDEFG",
    cumulative: 40,
    row: 6,
    column: 12,
    position: "M7",
    playedTiles: "F.EDBAC",
    score: 40,
  });

  const evt4 = create(GameEventSchema, {
    playerIndex: 0,
    type: GameEvent_Type.PHONY_TILES_RETURNED,
    cumulative: 0,
  });

  const turns = gameEventsToTurns([evt1, evt2, evt3, evt4]);
  expect(turns.length).toBe(2);
  expect(turns).toStrictEqual([
    {
      events: [evt1, evt2],
      firstEvtIdx: 0,
    },
    {
      events: [evt3, evt4],
      firstEvtIdx: 2,
    },
  ]);
});
