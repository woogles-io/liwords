import React, { useCallback, useEffect, useState } from "react";
import { Button, Checkbox } from "antd";
import { Modal } from "../utils/focus_modal";
import { ChallengeRule } from "../gen/api/proto/vendored/macondo/macondo_pb";

type Props = {
  wordsFormed: string[];
  onCancel: () => void;
  onConfirm: (selectedIndices: number[]) => void;
  modalVisible: boolean;
  challengeRule: ChallengeRule;
};

export const ChallengeWordsModal = React.memo((props: Props) => {
  const { wordsFormed, modalVisible, onConfirm, onCancel, challengeRule } =
    props;

  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(
    new Set(),
  );
  const [selectAll, setSelectAll] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string>("");

  // Reset selection when modal opens - default to NO words selected
  useEffect(() => {
    if (modalVisible) {
      setSelectedIndices(new Set());
      setSelectAll(false);
      setErrorMessage("");
    }
  }, [modalVisible]);

  const toggleWord = useCallback(
    (index: number) => {
      const newSet = new Set(selectedIndices);
      if (newSet.has(index)) {
        newSet.delete(index);
      } else {
        newSet.add(index);
      }
      setSelectedIndices(newSet);
      setSelectAll(newSet.size === wordsFormed.length);
      // Clear error when user makes a selection
      if (newSet.size > 0 && errorMessage) {
        setErrorMessage("");
      }
    },
    [selectedIndices, wordsFormed.length, errorMessage],
  );

  const handleSelectAll = useCallback(
    (checked: boolean) => {
      if (checked) {
        const allIndices = new Set(wordsFormed.map((_, i) => i));
        setSelectedIndices(allIndices);
        // Clear error when selecting all
        if (errorMessage) {
          setErrorMessage("");
        }
      } else {
        setSelectedIndices(new Set());
      }
      setSelectAll(checked);
    },
    [wordsFormed, errorMessage],
  );

  const handleConfirm = useCallback(() => {
    if (selectedIndices.size === 0) {
      setErrorMessage("You must select at least one word to challenge");
      return;
    }
    const indices = Array.from(selectedIndices).sort((a, b) => a - b);
    onConfirm(indices);
  }, [selectedIndices, onConfirm]);

  const is5Point = challengeRule === ChallengeRule.FIVE_POINT;
  const potentialBonus = is5Point ? selectedIndices.size * 5 : 0;

  return (
    <Modal
      className="challenge-words"
      title="Challenge Words"
      open={modalVisible}
      onOk={handleConfirm}
      onCancel={onCancel}
      width={400}
      footer={
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          {is5Point && selectedIndices.size > 0 && (
            <div style={{ color: "var(--warning-color)", fontSize: "14px" }}>
              Risk: {potentialBonus} point{potentialBonus !== 1 ? "s" : ""} if
              all valid
            </div>
          )}
          {(!is5Point || selectedIndices.size === 0) && <div />}
          <div>
            <Button onClick={onCancel}>Cancel</Button>
            <Button
              type="primary"
              onClick={handleConfirm}
              style={{ marginLeft: "8px" }}
            >
              Challenge
            </Button>
          </div>
        </div>
      }
    >
      <div className="challenge-words-list">
        {errorMessage && (
          <div
            style={{
              color: "var(--error-color, #ff4d4f)",
              fontSize: "14px",
              marginBottom: "12px",
              padding: "8px",
              backgroundColor: "var(--error-bg-color, #fff2f0)",
              borderRadius: "4px",
              border: "1px solid var(--error-border-color, #ffccc7)",
            }}
          >
            {errorMessage}
          </div>
        )}
        <Checkbox
          checked={selectAll}
          indeterminate={
            selectedIndices.size > 0 &&
            selectedIndices.size < wordsFormed.length
          }
          onChange={(e) => handleSelectAll(e.target.checked)}
          style={{ marginBottom: "12px" }}
        >
          Challenge entire play
        </Checkbox>
        <hr style={{ margin: "12px 0" }} />
        {wordsFormed.map((word, index) => (
          <div key={index} className="word-item" style={{ padding: "8px 0" }}>
            <Checkbox
              checked={selectedIndices.has(index)}
              onChange={() => toggleWord(index)}
            >
              <span
                style={{
                  fontFamily: "monospace",
                  fontSize: "16px",
                  fontWeight: "bold",
                }}
              >
                {word}
              </span>
              {index === 0 && (
                <span
                  style={{
                    fontSize: "12px",
                    color: "var(--text-secondary)",
                    marginLeft: "8px",
                  }}
                >
                  (main word)
                </span>
              )}
            </Checkbox>
          </div>
        ))}
      </div>
    </Modal>
  );
});
