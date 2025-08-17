import React from "react";
import ReactDOM from "react-dom/client";
import { StandaloneEmbed } from "./standalone-embed";
import { GameDocument } from "../gen/api/proto/ipc/omgwords_pb";

// Global interface for embed data
declare global {
  interface Window {
    WooglesEmbed?: {
      gameId: string;
      containerId: string;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      gameDocument: any; // Will be serialized JSON
      options?: {
        width?: number;
        height?: number;
        showControls?: boolean;
        showScores?: boolean;
        showMoveList?: boolean;
        theme?: "light" | "dark";
      };
    };
  }
}

// Initialize the embed when the script loads
function initEmbed() {
  if (!window.WooglesEmbed) {
    console.error("WooglesEmbed data not found");
    return;
  }

  const { containerId, gameDocument, options } = window.WooglesEmbed;

  // Find the container element
  const container = document.getElementById(containerId);
  if (!container) {
    console.error(`Container element with id "${containerId}" not found`);
    return;
  }

  // Parse the game document if it's a string
  let parsedGameDocument: GameDocument;
  try {
    if (typeof gameDocument === "string") {
      parsedGameDocument = JSON.parse(gameDocument);
    } else {
      parsedGameDocument = gameDocument;
    }
    // Log the structure to debug
    console.log("Parsed game document:", parsedGameDocument);
    if (parsedGameDocument.events && parsedGameDocument.events.length > 0) {
      console.log("First event structure:", parsedGameDocument.events[0]);
    }
  } catch (error) {
    console.error("Failed to parse game document:", error);
    return;
  }

  // Create React root and render the embed
  try {
    const root = ReactDOM.createRoot(container);
    root.render(
      <React.StrictMode>
        <StandaloneEmbed gameDocument={parsedGameDocument} options={options} />
      </React.StrictMode>,
    );
  } catch (error) {
    console.error("Failed to render embed:", error);
  }
}

// Simple logging for development
console.log("Woogles embed loaded");

// Wait for DOM to be ready
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initEmbed);
} else {
  // DOM is already ready
  initEmbed();
}

// Also support manual initialization
// eslint-disable-next-line @typescript-eslint/no-explicit-any
(window as any).initWooglesEmbed = initEmbed;
