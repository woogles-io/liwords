import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { Store, useLoginStateStoreContext } from "../store/store";
import { LiwordsSocket } from "./socket";

const mocks = vi.hoisted(() => {
  // socket.ts reads this when it is imported
  window.RUNTIME_CONFIGURATION = { appVersion: "test" };
  return {
    // the url getter LiwordsSocket hands react-use-websocket
    getUrl: null as null | (() => Promise<string>),
    // each token fetch waits here until the test answers it with a cid
    answerToken: [] as Array<(cid: string) => void>,
  };
});

vi.mock("react-use-websocket", () => ({
  default: (getUrl: () => Promise<string>) => {
    mocks.getUrl = getUrl;
    return { sendMessage: () => {} };
  },
}));

// a token jwtDecode can read: {"unn":"u","uid":"u1","a":true,"cs":"0"}
const token = [
  btoa(JSON.stringify({ alg: "none" })),
  btoa(JSON.stringify({ unn: "u", uid: "u1", a: true, cs: "0" })),
  "sig",
].join(".");

vi.mock("../utils/hooks/connect", () => ({
  useClient: () => ({
    getSocketToken: () =>
      new Promise((resolve) =>
        mocks.answerToken.push((cid) =>
          resolve({ cid, frontEndVersion: "test", token }),
        ),
      ),
  }),
}));

const queryClient = new QueryClient();

const WithSocket = (props: { children: ReactNode }) => (
  <MemoryRouter>
    <QueryClientProvider client={queryClient}>
      <Store>
        <LiwordsSocket resetSocket={() => {}} setValues={() => {}} />
        {props.children}
      </Store>
    </QueryClientProvider>
  </MemoryRouter>
);

describe("LiwordsSocket", () => {
  it("lets only the latest token fetch set the connection id", async () => {
    const { result } = renderHook(
      () => useLoginStateStoreContext().loginState.connID,
      { wrapper: WithSocket },
    );
    const getUrl = mocks.getUrl!;
    // two fetches overlap, as when a dev build's StrictMode runs the effect
    // twice; the socket opens with the url from the later one
    const earlier = getUrl();
    const later = getUrl();
    expect(mocks.answerToken.length).toBe(2);
    // the later one is answered first, and the earlier one lands last
    let laterUrl = "";
    await act(async () => {
      mocks.answerToken[1]("cid-later");
      laterUrl = await later;
    });
    let earlierUrl = "";
    await act(async () => {
      mocks.answerToken[0]("cid-earlier");
      earlierUrl = await earlier;
    });
    expect(result.current).toBe("cid-later");
    expect(laterUrl).toContain("cid=cid-later");
    expect(earlierUrl).toBe("");
  });
});
