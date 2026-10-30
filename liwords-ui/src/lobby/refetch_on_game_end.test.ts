import { act, renderHook } from "@testing-library/react";
import { useRefetchOnGameEnd } from "./refetch_on_game_end";

type Props = { ids: Array<string> };

const setup = (fetched: Array<Array<string>>) => {
  const refetch = vi.fn(async () => fetched.shift() ?? []);
  const hook = renderHook(
    (props: Props) => useRefetchOnGameEnd(props.ids, refetch),
    {
      initialProps: { ids: ["a", "b"] },
    },
  );
  return { refetch, hook };
};

// Lets the timers that are due run, and the fetches they start settle.
const advance = async (ms: number) => {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(ms);
  });
};

describe("useRefetchOnGameEnd", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("fetches a moment after a game leaves the list", async () => {
    const { refetch, hook } = setup([["a"]]);
    hook.rerender({ ids: ["b"] });
    await advance(999);
    expect(refetch).not.toHaveBeenCalled();
    await advance(1);
    expect(refetch).toHaveBeenCalledTimes(1);
    // The game came back with it: no second try.
    await advance(5000);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("tries once more when the game is not saved yet", async () => {
    const { refetch, hook } = setup([[], ["a"], ["a"]]);
    hook.rerender({ ids: ["b"] });
    await advance(1000);
    expect(refetch).toHaveBeenCalledTimes(1);
    await advance(3000);
    expect(refetch).toHaveBeenCalledTimes(2);
    await advance(10000);
    expect(refetch).toHaveBeenCalledTimes(2);
  });

  it("does not fetch for a game that joins the list or a move", async () => {
    const { refetch, hook } = setup([]);
    hook.rerender({ ids: ["a", "b", "c"] });
    hook.rerender({ ids: ["a", "b", "c"] });
    await advance(10000);
    expect(refetch).not.toHaveBeenCalled();
  });

  it("keeps the fetch when the list changes again meanwhile", async () => {
    const { refetch, hook } = setup([["a"]]);
    hook.rerender({ ids: ["b"] });
    await advance(500);
    hook.rerender({ ids: ["b", "c"] });
    await advance(500);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("drops what is pending on leaving the page", async () => {
    const { refetch, hook } = setup([["a"]]);
    hook.rerender({ ids: ["b"] });
    hook.unmount();
    await advance(10000);
    expect(refetch).not.toHaveBeenCalled();
  });

  it("makes one fetch for endings close together", async () => {
    const { refetch, hook } = setup([["a", "b"]]);
    hook.rerender({ ids: ["b"] });
    await advance(500);
    // Another ending starts the wait again.
    hook.rerender({ ids: [] });
    await advance(999);
    expect(refetch).not.toHaveBeenCalled();
    await advance(1);
    expect(refetch).toHaveBeenCalledTimes(1);
    await advance(10000);
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("fetches a second after an ending during the second try's wait", async () => {
    const { refetch, hook } = setup([[], ["b"]]);
    hook.rerender({ ids: ["b"] });
    await advance(1000);
    expect(refetch).toHaveBeenCalledTimes(1);
    // "a" is missing, so a second try waits; "b" ends meanwhile.
    await advance(1500);
    hook.rerender({ ids: [] });
    await advance(1000);
    expect(refetch).toHaveBeenCalledTimes(2);
    // That fetch was "a"'s second look: no more, though "a" is still missing.
    await advance(10000);
    expect(refetch).toHaveBeenCalledTimes(2);
  });

  it("looks again for a game that left while a fetch was out", async () => {
    let answer: (ids: Array<string>) => void = () => {};
    const refetch = vi.fn(
      () =>
        new Promise<Array<string>>((resolve) => {
          answer = resolve;
        }),
    );
    const hook = renderHook(
      (props: Props) => useRefetchOnGameEnd(props.ids, refetch),
      { initialProps: { ids: ["a", "b"] } },
    );
    hook.rerender({ ids: ["b"] });
    await advance(1000);
    expect(refetch).toHaveBeenCalledTimes(1);
    // "b" ends while that fetch is out; its answer cannot have "b".
    hook.rerender({ ids: [] });
    await act(async () => {
      answer(["a"]);
    });
    await advance(1000);
    expect(refetch).toHaveBeenCalledTimes(2);
  });
});
