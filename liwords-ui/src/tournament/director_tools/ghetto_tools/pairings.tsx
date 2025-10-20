// Pairing and result management forms

import { Button, Form, InputNumber, message, Select, Switch } from "antd";
import { Store } from "rc-field-form/lib/interface";
import React, { useEffect, useState } from "react";
import { GameEndReason } from "../../../gen/api/proto/ipc/omgwords_pb";
import { TournamentGameResult } from "../../../gen/api/proto/ipc/tournament_pb";
import { TournamentService } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { getEnumValue } from "../../../utils/protobuf";
import {
  DivisionFormItem,
  fullPlayerID,
  PlayersFormItem,
  userUUID,
} from "./shared";

export const SetPairing = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [division, setDivision] = useState("");
  const [selfplay, setSelfplay] = useState(false);
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    if (!vals.p1) {
      message.error("Player 1 is required.");
      return;
    }
    if (!vals.selfplay && !vals.p2) {
      message.error("Player 2 is required.");
      return;
    }

    const p1id = fullPlayerID(
      vals.p1,
      tournamentContext.divisions[vals.division],
    );
    let p2id;

    if (vals.selfplay) {
      p2id = p1id;
      if (!vals.selfplayresult) {
        message.error("Desired result for Player 1 is required.");
        return;
      }
    } else {
      p2id = fullPlayerID(vals.p2, tournamentContext.divisions[vals.division]);
    }

    const obj = {
      id: props.tournamentID,
      division: vals.division,
      pairings: [
        {
          playerOneId: p1id,
          playerTwoId: p2id,
          round: vals.round - 1, // 1-indexed input
          // use self-play result only if it was set.
          selfPlayResult: vals.selfplay ? vals.selfplayresult : undefined,
        },
      ],
    };
    try {
      await tClient.setPairing(obj);
      message.info({
        content: "Pairing set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="p1"
        label="Player 1 username"
        division={division}
        required
      />

      <Form.Item name="selfplay" label="Player has no opponent">
        <Switch checked={selfplay} onChange={(c) => setSelfplay(c)} />
      </Form.Item>

      {!selfplay ? (
        <PlayersFormItem
          name="p2"
          label="Player 2 username"
          division={division}
          required
        />
      ) : (
        <Form.Item
          name="selfplayresult"
          label="Desired result for this player"
          required
        >
          <Select>
            <Select.Option value={TournamentGameResult.BYE}>
              Bye (+50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.FORFEIT_LOSS}>
              Forfeit loss (-50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.VOID}>
              Void (no record change)
            </Select.Option>
          </Select>
        </Form.Item>
      )}

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const SetResult = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [division, setDivision] = useState("");
  const [score1, setScore1] = useState(0);
  const [score2, setScore2] = useState(0);
  const [form] = Form.useForm();
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      playerOneId: userUUID(
        vals.p1,
        tournamentContext.divisions[vals.division],
      ),
      playerTwoId: userUUID(
        vals.p2,
        tournamentContext.divisions[vals.division],
      ),
      round: vals.round - 1, // 1-indexed input
      playerOneScore: vals.p1score,
      playerTwoScore: vals.p2score,
      playerOneResult: getEnumValue(TournamentGameResult, vals.p1result),
      playerTwoResult: getEnumValue(TournamentGameResult, vals.p2result),
      gameEndReason: getEnumValue(GameEndReason, vals.gameEndReason),
      amendment: vals.amendment,
    };
    try {
      await tClient.setResult(obj);
      message.info({
        content: "Result set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  useEffect(() => {
    if (score1 > score2) {
      form.setFieldsValue({
        p1result: "WIN",
        p2result: "LOSS",
      });
    } else if (score1 < score2) {
      form.setFieldsValue({
        p1result: "LOSS",
        p2result: "WIN",
      });
    } else {
      form.setFieldsValue({
        p1result: "DRAW",
        p2result: "DRAW",
      });
    }
  }, [form, score1, score2]);

  const score1Change = (v: number | string | null | undefined) => {
    if (typeof v !== "number") {
      return;
    }
    setScore1(v);
  };
  const score2Change = (v: number | string | null | undefined) => {
    if (typeof v !== "number") {
      return;
    }
    setScore2(v);
  };

  return (
    <Form
      form={form}
      onFinish={onFinish}
      initialValues={{ gameEndReason: "STANDARD" }}
    >
      <DivisionFormItem onChange={(div: string) => setDivision(div)} />

      <PlayersFormItem
        name="p1"
        label="Player 1 username"
        division={division}
        required
      />
      <PlayersFormItem
        name="p2"
        label="Player 2 username"
        division={division}
        required
      />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item name="p1score" label="Player 1 score">
        <InputNumber
          inputMode="numeric"
          onChange={score1Change}
          value={score1}
        />
      </Form.Item>

      <Form.Item name="p2score" label="Player 2 score">
        <InputNumber
          inputMode="numeric"
          onChange={score2Change}
          value={score2}
        />
      </Form.Item>

      <Form.Item name="p1result" label="Player 1 result">
        <Select>
          <Select.Option value="VOID">VOID (no win or loss)</Select.Option>
          <Select.Option value="WIN">WIN</Select.Option>
          <Select.Option value="LOSS">LOSS</Select.Option>
          <Select.Option value="DRAW">DRAW</Select.Option>
          <Select.Option value="BYE">BYE</Select.Option>
          <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
          <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="p2result" label="Player 2 result">
        <Select>
          <Select.Option value="VOID">VOID</Select.Option>
          <Select.Option value="WIN">WIN</Select.Option>
          <Select.Option value="LOSS">LOSS</Select.Option>
          <Select.Option value="DRAW">DRAW</Select.Option>
          <Select.Option value="BYE">BYE</Select.Option>
          <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
          <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
          <Select.Option value="NO_RESULT">NO_RESULT</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="gameEndReason" label="Game End Reason">
        <Select>
          <Select.Option value="NONE">NONE</Select.Option>
          <Select.Option value="TIME">TIME</Select.Option>
          <Select.Option value="STANDARD">STANDARD</Select.Option>
          <Select.Option value="CONSECUTIVE_ZEROES">
            CONSECUTIVE_ZEROES
          </Select.Option>
          <Select.Option value="RESIGNED">RESIGNED</Select.Option>
          <Select.Option value="ABORTED">ABORTED</Select.Option>
          <Select.Option value="TRIPLE_CHALLENGE">
            TRIPLE_CHALLENGE
          </Select.Option>
          <Select.Option value="CANCELLED">CANCELLED</Select.Option>
          <Select.Option value="FORCE_FORFEIT">FORCE_FORFEIT</Select.Option>
        </Select>
      </Form.Item>

      <Form.Item name="amendment" label="Amendment" valuePropName="checked">
        <Switch />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const PairRound = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      preserveByes: vals.preserveByes,
    };
    try {
      await tClient.pairRound(obj);
      message.info({
        content: "Pair round completed",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />

      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>

      <Form.Item name="preserveByes" label="Preserve byes">
        <Switch />
      </Form.Item>

      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};

export const UnpairRound = (props: { tournamentID: string }) => {
  const tClient = useClient(TournamentService);
  const onFinish = async (vals: Store) => {
    const obj = {
      id: props.tournamentID,
      division: vals.division,
      round: vals.round - 1, // 1-indexed input
      deletePairings: true,
    };
    try {
      await tClient.pairRound(obj);
      message.info({
        content: "Pairings for selected round have been deleted",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };
  return (
    <Form onFinish={onFinish}>
      <DivisionFormItem />
      <Form.Item name="round" label="Round (1-indexed)">
        <InputNumber inputMode="numeric" min={1} required />
      </Form.Item>
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Submit
        </Button>
      </Form.Item>
    </Form>
  );
};
