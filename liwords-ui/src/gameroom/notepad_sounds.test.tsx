import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BoopSounds } from "../sound/boop";
import { NotepadContextProvider } from "./notepad";

vi.mock("../sound/boop", () => ({
  BoopSounds: { playSound: vi.fn(), soundIsEnabled: () => true },
}));

const played = () =>
  vi.mocked(BoopSounds.playSound).mock.calls.map((c) => c[0]);

// The provider holds the notes for every notepad on the page (a game page
// shows two), so its sounds must play once however many notepads there are.
describe("NotepadContextProvider sounds", () => {
  beforeEach(() => {
    vi.mocked(BoopSounds.playSound).mockClear();
    localStorage.clear();
  });

  it("screeches once when a saved note with kofi loads", () => {
    localStorage.setItem("notepad_g1", "careful here, kofi");
    render(
      <NotepadContextProvider feRackInfo gameID="g1">
        <div>notepad one</div>
        <div>notepad two</div>
      </NotepadContextProvider>,
    );
    expect(played()).toEqual(["eagleScreechSound"]);
  });

  it("plays the wolges sound once for a note with wolges", () => {
    localStorage.setItem("notepad_g1", "wolges says");
    render(
      <NotepadContextProvider feRackInfo gameID="g1">
        <div />
      </NotepadContextProvider>,
    );
    expect(played()).toEqual(["wolgesSound"]);
  });

  it("plays again only when there are more of them", () => {
    localStorage.setItem("notepad_g1", "kofi");
    localStorage.setItem("notepad_g2", "kofi kofi");
    localStorage.setItem("notepad_g3", "kofi");
    const { rerender } = render(
      <NotepadContextProvider feRackInfo gameID="g1">
        <div />
      </NotepadContextProvider>,
    );
    rerender(
      <NotepadContextProvider feRackInfo gameID="g2">
        <div />
      </NotepadContextProvider>,
    );
    rerender(
      <NotepadContextProvider feRackInfo gameID="g3">
        <div />
      </NotepadContextProvider>,
    );
    expect(played()).toEqual(["eagleScreechSound", "eagleScreechSound"]);
  });

  it("is silent for a note without them", () => {
    localStorage.setItem("notepad_g1", "just notes");
    render(
      <NotepadContextProvider feRackInfo gameID="g1">
        <div />
      </NotepadContextProvider>,
    );
    expect(played()).toEqual([]);
  });
});
