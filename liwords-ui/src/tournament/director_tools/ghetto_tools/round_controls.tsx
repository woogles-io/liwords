// Round control forms for tournament management

import {
  Button,
  Collapse,
  Divider,
  Form,
  InputNumber,
  message,
  Select,
  Switch,
} from "antd";
import { clone, create } from "@bufbuild/protobuf";
import React, { useEffect, useState } from "react";
import {
  DivisionControlsSchema,
  DivisionRoundControlsSchema,
  PairingMethod,
  RoundControl,
  RoundControlSchema,
  TournamentGameResult,
} from "../../../gen/api/proto/ipc/tournament_pb";
import { GameRequest } from "../../../gen/api/proto/ipc/omgwords_pb";
import {
  SingleRoundControlsRequestSchema,
  TournamentService,
} from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { flashError, useClient } from "../../../utils/hooks/connect";
import { DisplayedGameSetting, SettingsForm } from "../game_settings_form";
import { HelptipLabel } from "../helptip_label";
import { RoundSetting, settingsEqual } from "../pairing_methods";
import { DivisionSelector, showError } from "./shared";
import { RoundControlFields, rdCtrlFromSetting } from "./round_control_fields";
import { SingleRoundControlFields } from "./round_control_fields";
import { Modal } from "../../../utils/focus_modal";

const SetTournamentControls = (props: { tournamentID: string }) => {
  const [modalVisible, setModalVisible] = useState(false);
  const [selectedGameRequest, setSelectedGameRequest] = useState<
    GameRequest | undefined
  >(undefined);

  const [division, setDivision] = useState("");
  const [copyFromDivision, setCopyFromDivision] = useState("");
  const [gibsonize, setGibsonize] = useState(false);
  const [gibsonSpread, setGibsonSpread] = useState(500);

  // min placement is 0-indexed, but we want to display 1-indexed
  // this variable will be the display variable:
  const [gibsonMinPlacement, setGibsonMinPlacement] = useState(1);
  // bye max placement is 0-indexed, this is also the display variable
  const [byeMaxPlacement, setByeMaxPlacement] = useState(1);
  const [spreadCap, setSpreadCap] = useState(0);
  const [suspendedResult, setSuspendedResult] = useState<TournamentGameResult>(
    TournamentGameResult.FORFEIT_LOSS,
  );
  const { tournamentContext } = useTournamentStoreContext();

  useEffect(() => {
    if (!division) {
      setSelectedGameRequest(undefined);
      return;
    }
    const div = tournamentContext.divisions[division];
    const gameRequest = div.divisionControls?.gameRequest;
    if (gameRequest) {
      setSelectedGameRequest(gameRequest);
    } else {
      setSelectedGameRequest(undefined);
    }
    if (div.divisionControls) {
      setGibsonize(div.divisionControls.gibsonize);
      setGibsonSpread(div.divisionControls.gibsonSpread);
      setGibsonMinPlacement(div.divisionControls.minimumPlacement + 1);
      setByeMaxPlacement(div.divisionControls.maximumByePlacement + 1);
      setSuspendedResult(div.divisionControls.suspendedResult);
      setSpreadCap(div.divisionControls.spreadCap);
    }
  }, [division, tournamentContext.divisions]);

  const SettingsModalForm = (mprops: {
    visible: boolean;
    onCancel: () => void;
  }) => {
    return (
      <Modal
        title="Set Game Request"
        open={mprops.visible}
        onCancel={mprops.onCancel}
        className="seek-modal"
        okButtonProps={{ style: { display: "none" } }}
      >
        <SettingsForm
          setGameRequest={(gr) => {
            setSelectedGameRequest(gr);
            setModalVisible(false);
          }}
          gameRequest={selectedGameRequest}
        />
      </Modal>
    );
  };

  const tClient = useClient(TournamentService);

  const submit = async () => {
    if (!selectedGameRequest) {
      showError("No game request");
      return;
    }
    const ctrls = create(DivisionControlsSchema, {
      id: props.tournamentID,
      division,
      gameRequest: selectedGameRequest,
      // can set this later to whatever values, along with a spread
      suspendedResult,
      autoStart: false,
      gibsonize,
      gibsonSpread,
      minimumPlacement: gibsonMinPlacement - 1,
      maximumByePlacement: byeMaxPlacement - 1,
      spreadCap: spreadCap,
    });

    if (suspendedResult === TournamentGameResult.BYE) {
      ctrls.suspendedSpread = 50;
    } else if (suspendedResult === TournamentGameResult.FORFEIT_LOSS) {
      ctrls.suspendedSpread = -50;
    }

    try {
      await tClient.setDivisionControls(ctrls);
      message.info({
        content: "Controls set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 10,
    },
    wrapperCol: {
      span: 12,
    },
  };

  const SuspendedGameResultHelptip = (
    <>
      What result would you like to assign to players who join your tournament
      late, for unplayed rounds?<p>&nbsp;</p>
      <p>- Recommended value for tournaments is Forfeit loss. </p>
      <p>- Clubs can probably use a Void result.</p>
    </>
  );

  const copySettings = React.useCallback(() => {
    // copy settings from copyFromDivision to division
    const cd = tournamentContext.divisions[copyFromDivision];
    if (!cd.divisionControls) {
      return;
    }
    const cdCopy = clone(DivisionControlsSchema, cd.divisionControls);
    setSelectedGameRequest(cdCopy.gameRequest);
    setSuspendedResult(cdCopy.suspendedResult);
    setGibsonize(cdCopy.gibsonize);
    setGibsonSpread(cdCopy.gibsonSpread);
    setSpreadCap(cdCopy.spreadCap);
    // These are display variables so add 1 since they're 0-indexed:
    setGibsonMinPlacement(cdCopy.minimumPlacement + 1);
    setByeMaxPlacement(cdCopy.maximumByePlacement + 1);
  }, [copyFromDivision, tournamentContext.divisions]);

  return (
    <>
      <Form>
        <Form.Item {...formItemLayout} label="Division">
          <DivisionSelector
            value={division}
            onChange={(value: string) => {
              setDivision(value);
              setCopyFromDivision("");
            }}
          />
        </Form.Item>

        {Object.keys(tournamentContext.divisions).length > 1 &&
          division !== "" &&
          selectedGameRequest == null && (
            <Collapse style={{ marginBottom: 10 }}>
              <Collapse.Panel header="Copy from division" key="copyFrom">
                <p className="readable-text-color" style={{ marginBottom: 10 }}>
                  Copy from existing division:
                  <HelptipLabel
                    labelText=""
                    help="If you want to copy settings from another division, select
                that division and click the Copy button. Note that you must still
                click Save tournament controls to save your settings after copying them."
                  />
                </p>
                <DivisionSelector
                  value={copyFromDivision}
                  onChange={(value: string) => setCopyFromDivision(value)}
                  exclude={[division]}
                />
                <Button onClick={() => copySettings()}>
                  Copy from {copyFromDivision}
                </Button>
                <p className="readable-text-color" style={{ marginTop: 10 }}>
                  Or, set from scratch:
                </p>
              </Collapse.Panel>
            </Collapse>
          )}

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibsonize"
              help="If Gibsonize is on, players who have won the tournament before it is over will be paired against players not in contention."
            />
          }
        >
          <Switch
            checked={gibsonize}
            onChange={(c: boolean) => setGibsonize(c)}
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibson spread"
              help="Gibson spread is used to determine whether a player should be Gibsonized. With one round to go, if the first-place player is one win and this much spread ahead of second place, they will be Gibsonized."
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={0}
            value={gibsonSpread}
            onChange={(v: number | string | undefined | null) =>
              setGibsonSpread(v as number)
            }
          />
        </Form.Item>

        <Form.Item {...formItemLayout} label=" ">
          <p style={{ color: "#666", fontSize: "12px", margin: 0 }}>
            <strong>Note:</strong> Gibson settings above are overridden when using COP (Castellano O'Connor) pairings.
            COP uses its own gibson_spreads array configured per round.
          </p>
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              labelText="Gibson min placement"
              help="If Gibsonize is on, you typically want this number to be at least 2. This number should be the number of places that have prizes."
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={1}
            value={gibsonMinPlacement}
            onChange={(p: number | string | undefined | null) =>
              setGibsonMinPlacement(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help="Byes may be assigned to players ranked this, and worse,
          if odd. Make this 1 if you wish everyone in the tournament to be eligible for byes."
              labelText="Bye cut-off"
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={1}
            value={byeMaxPlacement}
            onChange={(p: number | string | undefined | null) =>
              setByeMaxPlacement(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help="Limit spread from losses to this number. If set to 0, there is no spread cap."
              labelText="Spread cap"
            />
          }
        >
          <InputNumber
            inputMode="numeric"
            min={0}
            value={spreadCap}
            onChange={(p: number | string | undefined | null) =>
              setSpreadCap(p as number)
            }
          />
        </Form.Item>

        <Form.Item
          {...formItemLayout}
          label={
            <HelptipLabel
              help={SuspendedGameResultHelptip}
              labelText="Suspended game result"
            />
          }
        >
          <Select
            value={suspendedResult}
            onChange={(v) => setSuspendedResult(v)}
          >
            <Select.Option value={TournamentGameResult.NO_RESULT}>
              Please select an option
            </Select.Option>
            <Select.Option value={TournamentGameResult.FORFEIT_LOSS}>
              Forfeit loss (-50)
            </Select.Option>
            <Select.Option value={TournamentGameResult.BYE}>
              Bye +50
            </Select.Option>
            <Select.Option value={TournamentGameResult.VOID}>
              Void (No win or loss)
            </Select.Option>
          </Select>
        </Form.Item>
      </Form>

      <div>{DisplayedGameSetting(selectedGameRequest)}</div>

      <Button
        htmlType="button"
        style={{
          margin: "0 8px",
        }}
        onClick={() => setModalVisible(true)}
      >
        Edit game settings
      </Button>
      <Button type="primary" onClick={submit}>
        Save tournament controls
      </Button>

      <SettingsModalForm
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
      />
    </>
  );
};

const SetSingleRoundControls = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  const [division, setDivision] = useState("");
  const [roundSetting, setRoundSetting] = useState<RoundControl>(
    create(RoundControlSchema, {
      pairingMethod: PairingMethod.RANDOM,
    }),
  );
  const [userVisibleRound, setUserVisibleRound] = useState(1);
  const tClient = useClient(TournamentService);

  const setRoundControls = async () => {
    if (!division) {
      showError("Division is missing");
      return;
    }
    if (userVisibleRound <= 0) {
      showError("Round must be a positive round number");
      return;
    }
    if (!roundSetting) {
      showError("Missing round setting");
      return;
    }

    const totalRounds = tournamentContext.divisions[division]?.numRounds;
    const halfwayPoint = Math.ceil(totalRounds / 2);

    // Validate COP can only be used in second half
    if (roundSetting.pairingMethod === PairingMethod.PAIRING_METHOD_COP) {
      if (userVisibleRound <= halfwayPoint) {
        showError(
          `COP can only be used for the second half of the tournament (round ${halfwayPoint + 1} and later). ` +
            `You are trying to set it for round ${userVisibleRound}.`,
        );
        return;
      }
    }

    const ctrls = create(SingleRoundControlsRequestSchema, {
      id: props.tournamentID,
      division: division,
    });
    let rdCtrl;
    try {
      rdCtrl = rdCtrlFromSetting(roundSetting, totalRounds);
    } catch (e) {
      message.error({
        content: (e as Error).message,
        duration: 5,
      });
      return;
    }
    rdCtrl.round = userVisibleRound - 1; // round is 0-indexed on backend.
    ctrls.roundControls = rdCtrl;
    try {
      await tClient.setSingleRoundControls(ctrls);
      message.info({
        content: `Controls set for round ${userVisibleRound}`,
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };

  return (
    <>
      <Form>
        <Form.Item {...formItemLayout} label="Division">
          <DivisionSelector
            value={division}
            onChange={(value: string) => setDivision(value)}
          />
        </Form.Item>
        <Form.Item {...formItemLayout} label="Round">
          <InputNumber
            inputMode="numeric"
            value={userVisibleRound}
            onChange={(e) => e && setUserVisibleRound(e as number)}
          />
        </Form.Item>
      </Form>
      <Divider />
      <Form>
        <SingleRoundControlFields
          setting={roundSetting}
          onChange={(
            fieldName: keyof RoundControl,
            value:
              | string
              | number
              | boolean
              | PairingMethod
              | number[]
              | string[],
          ) => {
            const val = { ...roundSetting, [fieldName]: value };
            setRoundSetting(create(RoundControlSchema, val));
          }}
          totalRounds={
            division
              ? tournamentContext.divisions[division]?.numRounds
              : undefined
          }
        />
        <Form.Item>
          <Button type="primary" onClick={() => setRoundControls()}>
            Submit
          </Button>
        </Form.Item>
      </Form>
    </>
  );
};

const SetDivisionRoundControls = (props: { tournamentID: string }) => {
  const { tournamentContext } = useTournamentStoreContext();
  // This form is too complicated to use the Ant Design built-in forms;
  // So we're just going to use form components instead.

  const [roundArray, setRoundArray] = useState<Array<RoundSetting>>([]);
  const [division, setDivision] = useState("");
  const [copyFromDivision, setCopyFromDivision] = useState("");

  const roundControlsToDisplayArray = React.useCallback(
    (roundControls: RoundControl[]) => {
      const settings = new Array<RoundSetting>();

      let lastSetting: RoundControl | null = null;
      let min = 1;
      let max = 1;
      roundControls.forEach((v: RoundControl, rd: number) => {
        const thisSetting = create(RoundControlSchema, {
          pairingMethod: v.pairingMethod,
          gamesPerRound: v.gamesPerRound,
          factor: v.factor,
          maxRepeats: v.maxRepeats,
          allowOverMaxRepeats: v.allowOverMaxRepeats,
          repeatRelativeWeight: v.repeatRelativeWeight,
          winDifferenceRelativeWeight: v.winDifferenceRelativeWeight,
          // COP-specific fields
          gibsonSpreads: v.gibsonSpreads,
          hopefulnessThresholds: v.hopefulnessThresholds,
          placePrizes: v.placePrizes,
          controlLossActivationRound: v.controlLossActivationRound,
          divisionSims: v.divisionSims,
          controlLossSims: v.controlLossSims,
          controlLossThreshold: v.controlLossThreshold,
        });
        if (lastSetting !== null) {
          if (settingsEqual(lastSetting, thisSetting)) {
            max = rd + 1;
          } else {
            settings.push({
              beginRound: min,
              endRound: max,
              setting: lastSetting,
            });
            min = max + 1;
            max = max + 1;
          }
        }
        lastSetting = thisSetting;
      });

      if (lastSetting !== null) {
        settings.push({
          beginRound: min,
          endRound: max,
          setting: lastSetting,
        });
      }
      return settings;
    },
    [],
  );

  useEffect(() => {
    if (!division) {
      setRoundArray([]);
      return;
    }
    const div = tournamentContext.divisions[division];
    setRoundArray(roundControlsToDisplayArray(div.roundControls));
  }, [division, roundControlsToDisplayArray, tournamentContext.divisions]);
  const tClient = useClient(TournamentService);

  const setRoundControls = async () => {
    if (!division) {
      showError("Division is missing");
      return;
    }
    if (!roundArray.length) {
      showError("Round controls are missing");
      return;
    }
    // validate round array
    let lastRd = 0;
    const totalRounds = roundArray[roundArray.length - 1].endRound;
    const halfwayPoint = Math.ceil(totalRounds / 2);

    for (let i = 0; i < roundArray.length; i++) {
      const rdCtrl = roundArray[i];
      if (rdCtrl.beginRound <= lastRd) {
        showError("Round numbers must be consecutive and increasing");
        return;
      }
      if (rdCtrl.endRound < rdCtrl.beginRound) {
        showError("End round must not be smaller than begin round");
        return;
      }
      if (rdCtrl.beginRound > lastRd + 1) {
        showError("Round numbers must be consecutive; you cannot skip rounds");
        return;
      }
      // Validate COP can only be used in second half
      if (rdCtrl.setting.pairingMethod === PairingMethod.PAIRING_METHOD_COP) {
        if (rdCtrl.beginRound <= halfwayPoint) {
          showError(
            `COP can only be used for the second half of the tournament (round ${halfwayPoint + 1} and later). ` +
              `This round range starts at round ${rdCtrl.beginRound}.`,
          );
          return;
        }
      }
      lastRd = rdCtrl.endRound;
    }

    const ctrls = create(DivisionRoundControlsSchema, {
      id: props.tournamentID,
      division: division,
    });

    const roundControls = new Array<RoundControl>();

    for (let r = 0; r < roundArray.length; r++) {
      const v = roundArray[r];
      for (let i = v.beginRound; i <= v.endRound; i++) {
        let rdCtrl;
        try {
          rdCtrl = rdCtrlFromSetting(v.setting, totalRounds);
        } catch (e) {
          message.error({
            content: (e as Error).message,
            duration: 5,
          });
          return;
        }
        roundControls.push(rdCtrl);
      }
    }

    ctrls.roundControls = roundControls;
    try {
      await tClient.setRoundControls(ctrls);
      message.info({
        content: "Controls set",
        duration: 3,
      });
    } catch (e) {
      flashError(e);
    }
  };

  const formItemLayout = {
    labelCol: {
      span: 7,
    },
    wrapperCol: {
      span: 12,
    },
  };

  const copySettings = React.useCallback(() => {
    const div = tournamentContext.divisions[copyFromDivision];
    setRoundArray(roundControlsToDisplayArray(div.roundControls));
  }, [
    copyFromDivision,
    roundControlsToDisplayArray,
    tournamentContext.divisions,
  ]);

  return (
    <>
      <Form.Item {...formItemLayout} label="Division">
        <DivisionSelector
          value={division}
          onChange={(value: string) => setDivision(value)}
        />
      </Form.Item>

      {Object.keys(tournamentContext.divisions).length > 1 &&
        division !== "" &&
        roundArray.length === 0 && (
          <Collapse style={{ marginBottom: 10 }}>
            <Collapse.Panel header="Copy from division" key="copyFrom">
              <p className="readable-text-color" style={{ marginBottom: 10 }}>
                Copy from existing division:
                <HelptipLabel
                  labelText=""
                  help="If you want to copy round controls from another division, select
                that division and click the Copy button. Note that you must still
                click Save round controls to save your controls after copying them."
                />
              </p>
              <DivisionSelector
                value={copyFromDivision}
                onChange={(value: string) => setCopyFromDivision(value)}
                exclude={[division]}
              />
              <Button onClick={() => copySettings()}>
                Copy from {copyFromDivision}
              </Button>
              <p className="readable-text-color" style={{ marginTop: 10 }}>
                Or, set from scratch:
              </p>
            </Collapse.Panel>
          </Collapse>
        )}

      <Divider />
      {roundArray.map((v, idx) => {
        // Calculate total rounds from the last round control
        const totalRounds =
          roundArray.length > 0
            ? roundArray[roundArray.length - 1].endRound
            : 0;

        return (
          <RoundControlFields
            key={`rdctrl-${idx}`}
            setting={v}
            totalRounds={totalRounds}
            onChange={(
              fieldName: string,
              value: string | number | boolean | number[] | string[],
            ) => {
              const newRdArray = [...roundArray];

              if (fieldName === "beginRound" || fieldName === "endRound") {
                newRdArray[idx] = {
                  ...newRdArray[idx],
                  [fieldName]: value,
                };
              } else {
                newRdArray[idx] = {
                  ...newRdArray[idx],
                  setting: create(RoundControlSchema, {
                    ...newRdArray[idx].setting,
                    [fieldName]: value,
                  }),
                };
              }
              setRoundArray(newRdArray);
            }}
            onRemove={() => {
              const newRdArray = [...roundArray];
              newRdArray.splice(idx, 1);
              setRoundArray(newRdArray);
            }}
          />
        );
      })}
      <Button
        onClick={() => {
          const newRdArray = [...roundArray];
          const last = roundArray[roundArray.length - 1];
          newRdArray.push({
            beginRound: last?.endRound ? last.endRound + 1 : 1,
            endRound: last?.endRound ? last.endRound + 1 : 1,
            setting: create(RoundControlSchema, {
              pairingMethod: PairingMethod.MANUAL,
            }),
          });
          setRoundArray(newRdArray);
        }}
      >
        + Add more pairings
      </Button>

      <Button onClick={() => setRoundControls()}>Save round controls</Button>
    </>
  );
};

// Export the main form components
export {
  SetTournamentControls,
  SetSingleRoundControls,
  SetDivisionRoundControls,
};
