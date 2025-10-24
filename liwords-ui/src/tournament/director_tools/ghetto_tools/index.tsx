// Main entry point for tournament director tools (temporary "ghetto" tools)

import { Button, Divider, Space } from "antd";
import React, { useState } from "react";
import { TType } from "../../../gen/api/proto/tournament_service/tournament_service_pb";
import { useTournamentStoreContext } from "../../../store/store";
import { Modal } from "../../../utils/focus_modal";
import { lowerAndJoin } from "./shared";
import { AddDivision, RemoveDivision, RenameDivision } from "./divisions";
import { AddPlayers, RemovePlayer, MovePlayer } from "./players";
import { PairRound, SetPairing, SetResult, UnpairRound } from "./pairings";
import {
  SetDivisionRoundControls,
  SetSingleRoundControls,
  SetTournamentControls,
} from "./round_controls";
import { CreatePrintableScorecards, ExportTournament } from "./export";
import { EditDescription } from "./metadata";
import {
  ManageCheckIns,
  UnfinishTournament,
  UnstartTournament,
} from "./tournament_management";
import { ManageDirectorsModal } from "../../manage_directors_modal";
import { UserOutlined } from "@ant-design/icons";

type ModalProps = {
  title: string;
  visible: boolean;
  type: string;
  handleOk: () => void;
  handleCancel: () => void;
  tournamentID: string;
};

const FormModal = (props: ModalProps) => {
  const forms = {
    "add-division": <AddDivision tournamentID={props.tournamentID} />,
    "rename-division": <RenameDivision tournamentID={props.tournamentID} />,
    "remove-division": <RemoveDivision tournamentID={props.tournamentID} />,
    "add-players": <AddPlayers tournamentID={props.tournamentID} />,
    "remove-player": <RemovePlayer tournamentID={props.tournamentID} />,
    "move-player-to-division": <MovePlayer tournamentID={props.tournamentID} />,
    "set-single-pairing": <SetPairing tournamentID={props.tournamentID} />,
    "set-game-result": <SetResult tournamentID={props.tournamentID} />,
    "pair-entire-round": <PairRound tournamentID={props.tournamentID} />,
    "unpair-entire-round": <UnpairRound tournamentID={props.tournamentID} />,
    "set-tournament-controls": (
      <SetTournamentControls tournamentID={props.tournamentID} />
    ),
    "set-round-controls": (
      <SetDivisionRoundControls tournamentID={props.tournamentID} />
    ),
    "set-single-round-controls": (
      <SetSingleRoundControls tournamentID={props.tournamentID} />
    ),
    "create-printable-scorecards": (
      <CreatePrintableScorecards tournamentID={props.tournamentID} />
    ),
    "export-tournament": <ExportTournament tournamentID={props.tournamentID} />,
    "edit-description-and-other-settings": (
      <EditDescription tournamentID={props.tournamentID} />
    ),
    "unstart-tournament": (
      <UnstartTournament tournamentID={props.tournamentID} />
    ),
    "unfinish-tournament": (
      <UnfinishTournament tournamentID={props.tournamentID} />
    ),
    "manage-check-ins-and-registrations": (
      <ManageCheckIns tournamentID={props.tournamentID} />
    ),
  };

  type FormKeys = keyof typeof forms;

  return (
    <Modal
      title={props.title}
      open={props.visible}
      footer={null}
      destroyOnClose={true}
      onCancel={props.handleCancel}
      className="seek-modal" // temporary display hack
    >
      {forms[props.type as FormKeys]}
    </Modal>
  );
};

type Props = {
  tournamentID: string;
};

export const GhettoTools = (props: Props) => {
  const [modalTitle, setModalTitle] = useState("");
  const [modalVisible, setModalVisible] = useState(false);
  const [modalType, setModalType] = useState("");
  const [manageDirectorsVisible, setManageDirectorsVisible] = useState(false);
  const { tournamentContext } = useTournamentStoreContext();

  const showModal = (key: string, title: string) => {
    setModalType(key);
    setModalVisible(true);
    setModalTitle(title);
  };

  const makeButton = (label: string) => {
    const key = lowerAndJoin(label);
    return (
      <Button key={key} onClick={() => showModal(key, label)} size="small">
        {label}
      </Button>
    );
  };

  const makeButtonGroup = (labels: string[]) => (
    <Space wrap size="small">
      {labels.map(makeButton)}
    </Space>
  );

  return (
    <>
      <h3>Tournament Tools</h3>
      <h4>General Settings</h4>
      <Space direction="vertical" size="small" style={{ width: "100%" }}>
        <Button
          onClick={() => setManageDirectorsVisible(true)}
          size="small"
          icon={<UserOutlined />}
        >
          Manage Directors
        </Button>
        {makeButton("Edit description and other settings")}
      </Space>
      <ManageDirectorsModal
        visible={manageDirectorsVisible}
        onClose={() => setManageDirectorsVisible(false)}
      />
      {(tournamentContext.metadata.type === TType.STANDARD ||
        tournamentContext.metadata.type === TType.CHILD) && (
        <>
          <Divider />
          <h4>Pre-Tournament Setup</h4>
          <div style={{ marginLeft: 12 }}>
            <h5
              style={{
                marginTop: 8,
                marginBottom: 8,
                fontWeight: 600,
                color: "#666",
              }}
            >
              Divisions
            </h5>
            {makeButtonGroup([
              "Add division",
              "Rename division",
              "Remove division",
            ])}
            <h5
              style={{
                marginTop: 12,
                marginBottom: 8,
                fontWeight: 600,
                color: "#666",
              }}
            >
              Controls
            </h5>
            {makeButtonGroup(["Set tournament controls", "Set round controls"])}
            <div style={{ marginTop: 8 }}>
              {makeButtonGroup([
                "Manage check-ins and registrations",
                "Create printable scorecards",
              ])}
            </div>
          </div>

          <Divider />
          <h4>Players</h4>
          {makeButtonGroup([
            "Add players",
            "Remove player",
            "Move player to division",
          ])}

          <Divider />
          <h4>In-Tournament Management</h4>
          {makeButtonGroup(["Set single round controls", "Set single pairing"])}
          <div style={{ marginTop: 8 }}>
            {makeButtonGroup(["Pair entire round", "Unpair entire round"])}
          </div>
          <div style={{ marginTop: 8 }}>{makeButton("Set game result")}</div>

          <Divider />
          <h4>Post-Tournament Utilities</h4>
          {makeButtonGroup(["Export tournament", "Unfinish tournament"])}

          <Divider />
          <h4>Danger!</h4>
          {makeButton("Unstart tournament")}
          <Divider />
        </>
      )}
      <FormModal
        title={modalTitle}
        visible={modalVisible}
        type={modalType}
        handleOk={() => setModalVisible(false)}
        handleCancel={() => setModalVisible(false)}
        tournamentID={props.tournamentID}
      />
    </>
  );
};
