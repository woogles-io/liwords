import React from "react";
import { ConnectError, Code } from "@connectrpc/connect";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { WordCheck } from "./word_check";

// one client for the page's life, as useClient's memo gives
const { checkWord, client } = vi.hoisted(() => {
  const checkWord = vi.fn();
  return { checkWord, client: { checkWord } };
});

vi.mock("../utils/hooks/connect", () => ({
  useClient: () => client,
}));

// a server that checks words: QI is valid, QX is not, and ZA is asked after
// the game is over. A word comes back as it was read, each tile by its own
// label: here, a letter in upper case.
const answers = (req: { word: string }) => {
  const word = req.word.trim().toUpperCase();
  switch (word) {
    case "QI":
      return Promise.resolve({ valid: true, word });
    case "ZA":
      return Promise.reject(
        new ConnectError("the game is over", Code.PermissionDenied),
      );
    default:
      return Promise.resolve({ valid: false, word });
  }
};

afterEach(() => {
  cleanup();
  checkWord.mockReset();
});

const ask = async (word: string) => {
  const input = await screen.findByPlaceholderText("Word");
  fireEvent.change(input, { target: { value: word } });
  // a key's press and release, as a keyboard sends them: the field takes
  // one Enter per press
  fireEvent.keyDown(input, { key: "Enter", code: "Enter", keyCode: 13 });
  fireEvent.keyUp(input, { key: "Enter", code: "Enter", keyCode: 13 });
};

it("checks a typed word with the game and lists the answers, newest first", async () => {
  checkWord.mockImplementation(answers);
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask(" qi ");
  expect(await screen.findByText("valid")).toBeInTheDocument();
  expect(checkWord).toHaveBeenCalledWith({ gameId: "g1", word: "qi" });
  await ask("qx");
  expect(await screen.findByText("not valid")).toBeInTheDocument();
  // each as the server read it
  const words = screen.getAllByText(/^(QI|QX)$/).map((el) => el.textContent);
  expect(words).toEqual(["QX", "QI"]);
});

it("shows why the server would not check a word", async () => {
  checkWord.mockImplementation(answers);
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask("za");
  expect(await screen.findByText("the game is over")).toBeInTheDocument();
  // as typed: the server read nothing
  expect(screen.getByText("za")).toBeInTheDocument();
});

it("asks nothing for a blank entry", async () => {
  checkWord.mockImplementation(answers);
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask("   ");
  expect(checkWord).not.toHaveBeenCalled();
});

it("shows Check at once, asking nothing", () => {
  render(<WordCheck gameID="g1" />);
  expect(screen.getByText("Check")).toBeInTheDocument();
  expect(checkWord).not.toHaveBeenCalled();
});

it("keeps the keys typed in it from the board around it", async () => {
  checkWord.mockImplementation(answers);
  // the board takes a letter's keydown to place a tile, and keeps a key's
  // character out of any field but its own on keypress
  const boardKeyDown = vi.fn();
  const boardKeyPress = vi.fn();
  render(
    <div onKeyDown={boardKeyDown} onKeyPress={boardKeyPress}>
      <WordCheck gameID="g1" />
    </div>,
  );
  fireEvent.click(await screen.findByText("Check"));
  const input = await screen.findByPlaceholderText("Word");
  fireEvent.keyDown(input, { key: "q", code: "KeyQ", keyCode: 81 });
  fireEvent.keyPress(input, { key: "q", code: "KeyQ", charCode: 113 });
  fireEvent.keyDown(input, { key: "Enter", code: "Enter", keyCode: 13 });
  expect(boardKeyDown).not.toHaveBeenCalled();
  expect(boardKeyPress).not.toHaveBeenCalled();
});

it("checks each of several words typed at once, in the order typed", async () => {
  checkWord.mockImplementation(answers);
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask("qx");
  await screen.findByText("not valid");
  await ask(" qi,  qx ");
  await waitFor(() => expect(screen.getAllByText(/^(QI|QX)$/)).toHaveLength(3));
  expect(checkWord).toHaveBeenCalledWith({ gameId: "g1", word: "qi" });
  expect(checkWord).toHaveBeenCalledWith({ gameId: "g1", word: "qx" });
  // the first entry, and one ask per word of the second
  expect(checkWord).toHaveBeenCalledTimes(3);
  const words = screen.getAllByText(/^(QI|QX)$/).map((el) => el.textContent);
  expect(words).toEqual(["QI", "QX", "QX"]);
});

it("starts afresh on another game's page", async () => {
  checkWord.mockImplementation(answers);
  const { rerender } = render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask("qi");
  expect(await screen.findByText("valid")).toBeInTheDocument();
  rerender(<WordCheck gameID="g2" />);
  // none of the other game's answers
  fireEvent.click(await screen.findByText("Check"));
  await screen.findByPlaceholderText("Word");
  expect(screen.queryByText("QI")).toBeNull();
});

it("asks the words one at a time, however many come at once", async () => {
  // each answer waits to be let go
  const waiting: Array<() => void> = [];
  checkWord.mockImplementation(
    (req: { word: string }) =>
      new Promise((resolve) =>
        waiting.push(() =>
          resolve({ valid: true, word: req.word.toUpperCase() }),
        ),
      ),
  );
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  await ask("aa ab ad");
  for (const [i, w] of ["aa", "ab", "ad"].entries()) {
    await waitFor(() =>
      expect(checkWord).toHaveBeenCalledWith({ gameId: "g1", word: w }),
    );
    // the next is not asked before this one is answered
    expect(checkWord).toHaveBeenCalledTimes(i + 1);
    await act(async () => waiting.shift()!());
  }
  await waitFor(() =>
    expect(screen.getAllByText(/^(AA|AB|AD)$/)).toHaveLength(3),
  );
  const words = screen.getAllByText(/^(AA|AB|AD)$/).map((el) => el.textContent);
  expect(words).toEqual(["AA", "AB", "AD"]);
});

it("asks with the Check beside the box, as a phone may have no Enter", async () => {
  checkWord.mockImplementation(answers);
  render(<WordCheck gameID="g1" />);
  fireEvent.click(await screen.findByText("Check"));
  const input = await screen.findByPlaceholderText("Word");
  fireEvent.change(input, { target: { value: "qi" } });
  // the box's own button, the popover's
  const ask = document.querySelector(
    ".word-check .ant-input-search-button",
  ) as HTMLElement;
  fireEvent.click(ask);
  expect(await screen.findByText("valid")).toBeInTheDocument();
  expect(checkWord).toHaveBeenCalledWith({ gameId: "g1", word: "qi" });
});
