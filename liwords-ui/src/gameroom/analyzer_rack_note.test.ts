import { rackNote } from "./analyzer";

it("says no rack was entered when the rack is empty mid-game", () => {
  expect(rackNote(0, 80, false)).toBe("No rack entered for this turn.");
});

it("counts the known tiles of a partial rack", () => {
  expect(rackNote(3, 80, false)).toBe("Analyzing 3 known tiles.");
  expect(rackNote(1, 80, false)).toBe("Analyzing 1 known tile.");
});

it("says nothing about a full rack", () => {
  expect(rackNote(7, 80, false)).toBeNull();
});

it("says nothing once a short rack may just be the endgame", () => {
  expect(rackNote(3, 10, false)).toBeNull();
  expect(rackNote(0, 7, false)).toBeNull();
});

it("says nothing after the game is over", () => {
  expect(rackNote(0, 80, true)).toBeNull();
});

it("points to Space in the board editor", () => {
  expect(rackNote(0, 80, false, "keyboard")).toBe(
    "No rack entered for this turn. Press Space to enter the rack.",
  );
  expect(rackNote(3, 80, false, "keyboard")).toBe(
    "Analyzing 3 known tiles. Press Space to enter the rack.",
  );
});

it("points to the pencil on a touch screen", () => {
  expect(rackNote(0, 80, false, "touch")).toBe(
    "No rack entered for this turn. Tap the pencil to enter the rack.",
  );
});
