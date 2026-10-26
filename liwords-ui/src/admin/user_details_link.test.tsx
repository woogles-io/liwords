import { render, screen, waitFor } from "@testing-library/react";
import { message } from "antd";
import React from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { UserDetails } from "./user_details";

const refetch = vi.hoisted(() => vi.fn());

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: () => ({ data: undefined, refetch }),
}));

// StrictMode mounts twice, as development builds do; the first unmount
// cancels the first lookup.
const renderAt = (url: string) =>
  render(
    <React.StrictMode>
      <MemoryRouter initialEntries={[url]}>
        <UserDetails />
      </MemoryRouter>
    </React.StrictMode>,
  );

describe("UserDetails opened from a user's menu", () => {
  beforeEach(() => {
    refetch.mockReset();
    refetch.mockResolvedValue({});
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("looks the linked user up on each mount, with the name filled in", async () => {
    const error = vi.spyOn(message, "error");
    refetch.mockRejectedValueOnce(new Error("CancelledError"));
    renderAt("/admin?tab=user-details&username=cesar");
    expect(refetch).toHaveBeenCalledTimes(2);
    expect(refetch).toHaveBeenCalledWith({ throwOnError: true });
    expect(screen.getByDisplayValue("cesar")).toBeInTheDocument();
    // The canceled lookup says nothing.
    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(error).not.toHaveBeenCalled();
  });

  it("says when the lookup fails", async () => {
    const error = vi.spyOn(message, "error");
    refetch.mockRejectedValue(new Error("user not found"));
    renderAt("/admin?tab=user-details&username=cesar");
    await waitFor(() => expect(error).toHaveBeenCalledTimes(1));
  });

  it("looks nobody up without a username", () => {
    renderAt("/admin?tab=user-details");
    expect(refetch).not.toHaveBeenCalled();
  });
});
