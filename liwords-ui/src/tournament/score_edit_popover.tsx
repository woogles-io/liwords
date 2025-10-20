import React, { useState, useEffect } from "react";
import {
  Popover,
  InputNumber,
  Button,
  Space,
  Select,
  Checkbox,
  Form,
  message,
} from "antd";
import { TournamentService } from "../gen/api/proto/tournament_service/tournament_service_pb";
import {
  TournamentGameResult,
  TournamentPerson,
} from "../gen/api/proto/ipc/tournament_pb";
import { GameEndReason } from "../gen/api/proto/ipc/omgwords_pb";
import { useClient, flashError } from "../utils/hooks/connect";
import { getEnumValue } from "../utils/protobuf";

type Props = {
  tournamentID: string;
  division: string;
  round: number; // 0-indexed
  players: TournamentPerson[];
  currentScores?: number[];
  currentResults?: TournamentGameResult[];
  onSuccess?: () => void;
  children: React.ReactElement;
  isUnpaired?: boolean; // True if this is for a single unpaired player
};

export const ScoreEditPopover: React.FC<Props> = (props) => {
  const [open, setOpen] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [score1, setScore1] = useState<number>(0);
  const [score2, setScore2] = useState<number>(0);
  const [form] = Form.useForm();
  const tClient = useClient(TournamentService);

  // Extract usernames from player IDs (format: uuid:username)
  const player1Username = props.players[0]?.id.split(":")[1] || "Player 1";
  const player2Username = props.players[1]?.id.split(":")[1] || "Player 2";

  // Initialize form values when popover opens (only for paired players)
  useEffect(() => {
    if (open && !props.isUnpaired) {
      const existingScore1 = props.currentScores?.[0] || 0;
      const existingScore2 = props.currentScores?.[1] || 0;

      setScore1(existingScore1);
      setScore2(existingScore2);

      form.setFieldsValue({
        p1score: existingScore1,
        p2score: existingScore2,
        p1result: props.currentResults?.[0]
          ? TournamentGameResult[props.currentResults[0]]
          : "NO_RESULT",
        p2result: props.currentResults?.[1]
          ? TournamentGameResult[props.currentResults[1]]
          : "NO_RESULT",
        gameEndReason: "STANDARD",
        amendment: false,
      });
    }
  }, [open, props.currentScores, props.currentResults, props.isUnpaired, form]);

  // Auto-calculate results based on scores (only for paired players)
  useEffect(() => {
    if (props.isUnpaired) return;

    if (score1 !== score2) {
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
      }
    } else if (score1 === score2) {
      form.setFieldsValue({
        p1result: "DRAW",
        p2result: "DRAW",
      });
    }
  }, [score1, score2, props.isUnpaired, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();

      const obj = {
        id: props.tournamentID,
        division: props.division,
        playerOneId: props.players[0].id,
        playerTwoId: props.players[1].id,
        round: props.round,
        playerOneScore: values.p1score,
        playerTwoScore: values.p2score,
        playerOneResult: getEnumValue(TournamentGameResult, values.p1result),
        playerTwoResult: getEnumValue(TournamentGameResult, values.p2result),
        gameEndReason: getEnumValue(GameEndReason, values.gameEndReason),
        amendment: values.amendment || false,
      };

      await tClient.setResult(obj);
      message.success("Result set successfully");
      setOpen(false);
      setShowAdvanced(false);
      if (props.onSuccess) {
        props.onSuccess();
      }
    } catch (e) {
      flashError(e);
    }
  };

  const handleQuickAction = async (action: string) => {
    // For bye, forfeit_loss, and void, use setPairing API (player has no opponent)
    const selfPlayActions = [
      "bye_p1",
      "bye_p2",
      "forfeit_loss_p1",
      "forfeit_loss_p2",
      "void_p1",
      "void_p2",
    ];

    if (selfPlayActions.includes(action)) {
      const isPlayer1 = action.endsWith("_p1");
      const playerId = isPlayer1 ? props.players[0].id : props.players[1].id;
      let selfPlayResult: TournamentGameResult;

      if (action.startsWith("bye_")) {
        selfPlayResult = TournamentGameResult.BYE;
      } else if (action.startsWith("forfeit_loss_")) {
        selfPlayResult = TournamentGameResult.FORFEIT_LOSS;
      } else {
        selfPlayResult = TournamentGameResult.VOID;
      }

      try {
        await tClient.setPairing({
          id: props.tournamentID,
          division: props.division,
          pairings: [
            {
              playerOneId: playerId,
              playerTwoId: playerId,
              round: props.round,
              selfPlayResult: selfPlayResult,
            },
          ],
        });
        message.success("Pairing set successfully");
        setOpen(false);
        setShowAdvanced(false);
        if (props.onSuccess) {
          props.onSuccess();
        }
      } catch (e) {
        flashError(e);
      }
    } else {
      // For regular forfeits (with opponent), use setResult API
      switch (action) {
        case "forfeit_p1":
          form.setFieldsValue({
            p1score: 0,
            p2score: 0,
            p1result: "FORFEIT_LOSS",
            p2result: "FORFEIT_WIN",
            gameEndReason: "FORCE_FORFEIT",
          });
          break;
        case "forfeit_p2":
          form.setFieldsValue({
            p1score: 0,
            p2score: 0,
            p1result: "FORFEIT_WIN",
            p2result: "FORFEIT_LOSS",
            gameEndReason: "FORCE_FORFEIT",
          });
          break;
      }
      // Auto-submit after setting forfeit values
      await handleSubmit();
    }
  };

  const content = props.isUnpaired ? (
    // Simplified UI for unpaired players
    <div style={{ width: 280 }}>
      <Form form={form} layout="vertical">
        <h4 style={{ marginBottom: 16 }}>{player1Username}</h4>

        <div style={{ marginBottom: 8 }}>
          <strong>Set result for unpaired player:</strong>
        </div>
        <Space direction="vertical" style={{ width: "100%", marginBottom: 16 }}>
          <Button
            style={{ width: "100%" }}
            onClick={() => handleQuickAction("bye_p1")}
          >
            Bye (+50)
          </Button>
          <Button
            style={{ width: "100%" }}
            onClick={() => handleQuickAction("forfeit_loss_p1")}
          >
            Forfeit Loss (-50)
          </Button>
        </Space>

        <Space style={{ width: "100%", justifyContent: "flex-end" }}>
          <Button size="small" onClick={() => setOpen(false)}>
            Cancel
          </Button>
        </Space>
      </Form>
    </div>
  ) : (
    // Full UI for paired players
    <div style={{ width: 320 }}>
      <Form form={form} layout="vertical">
        <h4 style={{ marginBottom: 16 }}>
          {player1Username} vs {player2Username}
        </h4>

        <Form.Item
          name="p1score"
          label={`${player1Username} Score`}
          style={{ marginBottom: 8 }}
        >
          <InputNumber
            style={{ width: "100%" }}
            inputMode="numeric"
            onChange={(v) => setScore1(v as number)}
          />
        </Form.Item>

        <Form.Item
          name="p2score"
          label={`${player2Username} Score`}
          style={{ marginBottom: 16 }}
        >
          <InputNumber
            style={{ width: "100%" }}
            inputMode="numeric"
            onChange={(v) => setScore2(v as number)}
          />
        </Form.Item>

        {!showAdvanced ? (
          <>
            {/* Hidden form fields for results - auto-calculated in simple mode */}
            <Form.Item name="p1result" hidden>
              <input type="hidden" />
            </Form.Item>
            <Form.Item name="p2result" hidden>
              <input type="hidden" />
            </Form.Item>
            <Form.Item name="gameEndReason" hidden>
              <input type="hidden" />
            </Form.Item>
            <Form.Item name="amendment" hidden>
              <input type="hidden" />
            </Form.Item>

            <div style={{ marginBottom: 8 }}>
              <strong>Quick actions:</strong>
            </div>
            <Space wrap style={{ marginBottom: 16 }}>
              <Button size="small" onClick={() => handleQuickAction("bye_p1")}>
                Bye {player1Username}
              </Button>
              <Button size="small" onClick={() => handleQuickAction("bye_p2")}>
                Bye {player2Username}
              </Button>
              <Button
                size="small"
                onClick={() => handleQuickAction("forfeit_loss_p1")}
              >
                Forfeit Loss {player1Username}
              </Button>
              <Button
                size="small"
                onClick={() => handleQuickAction("forfeit_loss_p2")}
              >
                Forfeit Loss {player2Username}
              </Button>
              <Button size="small" onClick={() => handleQuickAction("void_p1")}>
                Void Result {player1Username}
              </Button>
              <Button size="small" onClick={() => handleQuickAction("void_p2")}>
                Void Result {player2Username}
              </Button>
            </Space>
          </>
        ) : null}

        {showAdvanced && (
          <>
            <Form.Item name="p1result" label={`${player1Username} Result`}>
              <Select>
                <Select.Option value="VOID">
                  VOID (no win or loss)
                </Select.Option>
                <Select.Option value="WIN">WIN</Select.Option>
                <Select.Option value="LOSS">LOSS</Select.Option>
                <Select.Option value="DRAW">DRAW</Select.Option>
                <Select.Option value="BYE">BYE</Select.Option>
                <Select.Option value="FORFEIT_WIN">FORFEIT_WIN</Select.Option>
                <Select.Option value="FORFEIT_LOSS">FORFEIT_LOSS</Select.Option>
                <Select.Option value="NO_RESULT">NO_RESULT</Select.Option>
              </Select>
            </Form.Item>

            <Form.Item name="p2result" label={`${player2Username} Result`}>
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
                <Select.Option value="FORCE_FORFEIT">
                  FORCE_FORFEIT
                </Select.Option>
              </Select>
            </Form.Item>

            <Form.Item name="amendment" valuePropName="checked">
              <Checkbox>Amendment</Checkbox>
            </Form.Item>
          </>
        )}

        <Space style={{ width: "100%", justifyContent: "space-between" }}>
          <Button
            size="small"
            type="link"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {showAdvanced ? "Simple ▲" : "Advanced ▼"}
          </Button>
          <Space>
            <Button size="small" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button size="small" type="primary" onClick={handleSubmit}>
              Submit
            </Button>
          </Space>
        </Space>
      </Form>
    </div>
  );

  return (
    <Popover
      content={content}
      title="Set Result"
      trigger="click"
      open={open}
      onOpenChange={setOpen}
      placement="left"
      zIndex={1100}
    >
      {props.children}
    </Popover>
  );
};
