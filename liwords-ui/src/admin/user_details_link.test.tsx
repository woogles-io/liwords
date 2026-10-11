import { render, screen } from "@testing-library/react";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UserDetails } from "./user_details";

const refetch = vi.hoisted(() => vi.fn());

vi.mock("@connectrpc/connect-query", () => ({
  useQuery: () => ({ data: undefined, refetch }),
}));

// StrictMode runs effects twice on mount, as development builds do.
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

  it("looks the linked user up once, with the name filled in", () => {
    renderAt("/admin?tab=user-details&username=cesar");
    expect(refetch).toHaveBeenCalledTimes(1);
    expect(refetch).toHaveBeenCalledWith({ throwOnError: true });
    expect(screen.getByDisplayValue("cesar")).toBeInTheDocument();
  });

  it("looks nobody up without a username", () => {
    renderAt("/admin?tab=user-details");
    expect(refetch).not.toHaveBeenCalled();
  });
});
