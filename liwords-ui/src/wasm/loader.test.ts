import { beforeEach, describe, expect, it, vi } from "vitest";

// A Loadable hands its bytes over once and then reports itself done, so the
// interesting case is not "does switching lexicons free memory" but "does the
// one you switched away from still work when you come back". CSW21, then RD29,
// then CSW21 again.

const precacheKwg = vi.fn();
const precacheKlv = vi.fn();
const evictKwg = vi.fn();
const evictKlv = vi.fn();

// The wasm, with or without the evict functions newer wolges-wasm builds have.
// Without them, a real module reads them as undefined; the mock says so, as a
// vitest mock throws on a name it was not given.
const mockWolges = (canEvict: boolean) =>
  vi.doMock("wolges-wasm", () => ({
    precache_kwg: (name: string, data: Uint8Array) => precacheKwg(name, data),
    precache_klv: (name: string, data: Uint8Array) => precacheKlv(name, data),
    evict_kwg: canEvict ? (name: string) => evictKwg(name) : undefined,
    evict_klv: canEvict ? (name: string) => evictKlv(name) : undefined,
  }));

// Each file is a distinct, non-empty body, so an empty one is unmistakable.
const bodyFor = (path: string) => new TextEncoder().encode(`bytes:${path}`);
let fetched: string[] = [];

beforeEach(() => {
  vi.clearAllMocks();
  // The loader keeps what it has cached in module scope, so each test needs its
  // own copy of the module or the second one starts with the first one's state.
  vi.resetModules();
  mockWolges(true);
  fetched = [];
  vi.stubGlobal("fetch", (path: string) => {
    fetched.push(path);
    return Promise.resolve(new Response(bodyFor(path), { status: 200 }));
  });
});

const loaded = (mock: typeof precacheKwg) =>
  mock.mock.calls.map((call) => ({
    name: call[0] as string,
    length: (call[1] as Uint8Array).length,
  }));

describe("getWolges", () => {
  it("reloads a lexicon it evicted, with its bytes intact", async () => {
    const { getWolges } = await import("./loader");

    const sizeOf = (path: string) => bodyFor(path).length;

    await getWolges("CSW21");
    expect(loaded(precacheKwg)).toEqual([
      { name: "CSW21", length: sizeOf("/wasm/2024/CSW21.kwg") },
    ]);
    expect(evictKwg).not.toHaveBeenCalled();

    await getWolges("RD29");
    expect(evictKwg).toHaveBeenCalledWith("CSW21");
    expect(evictKlv).toHaveBeenCalledWith("CSW21");
    expect(loaded(precacheKwg)).toContainEqual({
      name: "RD29",
      length: sizeOf("/wasm/2024/RD29.kwg"),
    });

    // Back again. This is the assertion the change exists for: the second
    // CSW21 load must carry real bytes. A spent Loadable would hand over
    // undefined here, which arrives as an empty array and panics the wasm.
    await getWolges("CSW21");
    const csw21Loads = loaded(precacheKwg).filter((c) => c.name === "CSW21");
    expect(csw21Loads).toHaveLength(2);
    expect(csw21Loads[1].length).toBeGreaterThan(0);
    expect(csw21Loads[1]).toEqual(csw21Loads[0]);

    // And it really did go back to the network for them rather than keeping
    // them alive somewhere.
    expect(fetched.filter((p) => p === "/wasm/2024/CSW21.kwg")).toHaveLength(2);
  });

  it("keeps what the request still needs", async () => {
    const { getWolges } = await import("./loader");

    // CSW21 and CSW21.WordSmog share a leave table, so switching between them
    // must not drop it.
    await getWolges("CSW21");
    await getWolges("CSW21.WordSmog");

    expect(evictKwg).toHaveBeenCalledWith("CSW21");
    expect(evictKlv).not.toHaveBeenCalled();
    expect(loaded(precacheKlv)).toHaveLength(1);
  });

  it("keeps everything with a wasm that cannot evict", async () => {
    mockWolges(false);
    const { getWolges } = await import("./loader");

    await getWolges("CSW21");
    await getWolges("RD29");
    await getWolges("CSW21");

    // CSW21 was never dropped, so it was loaded once and fetched once.
    expect(loaded(precacheKwg).filter((c) => c.name === "CSW21")).toHaveLength(
      1,
    );
    expect(fetched.filter((p) => p === "/wasm/2024/CSW21.kwg")).toHaveLength(1);
  });
});
