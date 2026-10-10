import { create, toBinary } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "antd";
import React, { useEffect } from "react";
import { MemoryRouter, useLocation } from "react-router";
import { afterEach, expect, it, vi } from "vitest";
import {
  AnalysisCompleteEventSchema,
  MessageType,
} from "../gen/api/proto/ipc/ipc_pb";
import { encodeToSocketFmt } from "../utils/protobuf";
import { useOnSocketMsg } from "./socket_handlers";
import { Store } from "./store";

// Where the "Computer analysis ready" toast goes: the game with its computer
// analysis open (#2020), which table_analysis_param.test.tsx covers from the
// game page's side. The toast's look and click wiring are pinned in
// analysis_toast.test.tsx; this drives the real socket handler.

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

let onSocketMsg: ReturnType<typeof useOnSocketMsg> = () => {};
let path = "";
const Probe = () => {
  const handler = useOnSocketMsg();
  const location = useLocation();
  useEffect(() => {
    onSocketMsg = handler;
  }, [handler]);
  useEffect(() => {
    path = location.pathname + location.search;
  }, [location]);
  return null;
};

const showToast = async () => {
  render(
    <MemoryRouter initialEntries={["/"]}>
      <Store>
        <QueryClientProvider client={new QueryClient()}>
          <App>
            <Probe />
          </App>
        </QueryClientProvider>
      </Store>
    </MemoryRouter>,
  );
  const bytes = encodeToSocketFmt(
    MessageType.ANALYSIS_COMPLETE,
    toBinary(
      AnalysisCompleteEventSchema,
      create(AnalysisCompleteEventSchema, { gameId: "abc" }),
    ),
  );
  act(() => {
    onSocketMsg({ result: bytes.buffer } as FileReader);
  });
  return screen.findByText("Computer analysis ready");
};

it("opens the game with its computer analysis", async () => {
  const toast = await showToast();

  await userEvent.click(toast);

  await waitFor(() => expect(path).toBe("/game/abc?analysis=computer"));
});

it("opens the same in a new tab on ctrl-click", async () => {
  const open = vi.spyOn(window, "open").mockImplementation(() => null);
  const toast = await showToast();

  const user = userEvent.setup();
  await user.keyboard("{Control>}");
  await user.click(toast);

  expect(open).toHaveBeenCalledWith("/game/abc?analysis=computer");
  expect(path).toBe("/");
});
