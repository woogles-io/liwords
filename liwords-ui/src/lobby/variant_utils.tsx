import React, { useState } from "react";
import { Modal } from "antd";
import { VariantIcon } from "../shared/variant_icons";

// Variant descriptions for modals and info displays
export const variantDescriptions: {
  [key: string]: {
    title: string;
    description: React.ReactNode;
  };
} = {
  wordsmog: {
    title: "WordSmog",
    description: (
      <>
        <p style={{ margin: "0 0 8px 0" }}>
          <strong>WordSmog</strong> is a variant where plays are acceptable if
          they form an anagram of a valid word. For example, ACT, CAT, TAC,
          CTA, ATC, and TCA are all valid plays in WordSmog as they all form
          the word "CAT" (and ACT).
        </p>
        <p style={{ margin: 0 }}>
          This variant requires careful strategic planning, defense, and great
          anagramming ability!
        </p>
      </>
    ),
  },
  classic_super: {
    title: "ZOMGWords",
    description: (
      <>
        <p style={{ margin: "0 0 8px 0" }}>
          <strong>ZOMGWords</strong> is the game you love but on a bigger
          (21x21) board with 200 tiles.
        </p>
        <p style={{ margin: 0 }}>
          Scores of over 1000 points per player are not unheard of!
        </p>
      </>
    ),
  },
};

// Get display name for a variant
export const getVariantDisplayName = (variant: string): string => {
  switch (variant) {
    case "":
    case "classic":
      return "OMGWords Classic";
    case "wordsmog":
      return "WordSmog";
    case "classic_super":
      return "ZOMGWords";
    default:
      return variant;
  }
};

// Normalize variant (treat empty string and "classic" the same)
export const normalizeVariant = (variant: string): string => {
  if (variant === "" || variant === "classic") {
    return "classic";
  }
  return variant;
};

type VariantSectionHeaderProps = {
  variant: string;
};

export const VariantSectionHeader = (props: VariantSectionHeaderProps) => {
  const [showInfo, setShowInfo] = useState(false);
  const normalized = normalizeVariant(props.variant);
  const displayName = getVariantDisplayName(normalized);
  const hasDescription = variantDescriptions[normalized];

  return (
    <>
      <h4>
        {displayName}
        {hasDescription && (
          <>
            {" "}
            <a
              onClick={() => setShowInfo(true)}
              style={{
                fontSize: 13,
                fontWeight: "normal",
                cursor: "pointer",
                marginLeft: 8,
              }}
            >
              What's this?
            </a>
          </>
        )}
      </h4>
      {hasDescription && (
        <Modal
          title={
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <VariantIcon vcode={normalized} />
              <span>{variantDescriptions[normalized].title}</span>
            </div>
          }
          open={showInfo}
          onCancel={() => setShowInfo(false)}
          footer={null}
          width={500}
        >
          <div style={{ fontSize: 14, lineHeight: 1.6 }}>
            {variantDescriptions[normalized].description}
          </div>
        </Modal>
      )}
    </>
  );
};
