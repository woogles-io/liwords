import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UsernameWithContext } from "./usernameWithContext";

const login = vi.hoisted(() => ({
  permissions: [] as string[],
  loggedIn: true,
}));

vi.mock("../store/store", () => ({
  useContextMatchContext: () => ({ handleContextMatches: [] }),
  useLoginStateStoreContext: () => ({
    loginState: {
      loggedIn: login.loggedIn,
      userID: "me-id",
      permissions: login.permissions,
    },
  }),
}));
vi.mock("./display_flag", () => ({ DisplayUserFlag: () => null }));
vi.mock("./display_title", () => ({ DisplayUserTitle: () => null }));
vi.mock("../profile/badge", () => ({ DisplayUserBadges: () => null }));
vi.mock("./blocker", () => ({ TheBlocker: () => null }));
vi.mock("./follower", () => ({ TheFollower: () => null }));
vi.mock("./player_avatar", async () => {
  const { createContext } = await import("react");
  return {
    PettableContext: createContext({
      isPettable: false,
      isPetting: false,
      setPetting: () => {},
    }),
  };
});

const openMenu = async () => {
  render(
    <MemoryRouter>
      <UsernameWithContext
        username="cesar"
        userID="cesar-id"
        omitFriend
        omitBlock
      />
    </MemoryRouter>,
  );
  await userEvent.click(screen.getByText("cesar"));
  // The menu is open once its always-present entry shows.
  await screen.findByRole("link", { name: "View profile" });
};

describe("the user menu's User details entry", () => {
  beforeEach(() => {
    // The menu renders into #root.
    const root = document.createElement("div");
    root.id = "root";
    document.body.replaceChildren(root);
  });

  it("links those who can see private user data to the user lookup", async () => {
    login.permissions = ["can_see_private_user_data"];
    await openMenu();
    const link = screen.getByRole("link", { name: "User details" });
    expect(link).toHaveAttribute(
      "href",
      "/admin?tab=user-details&username=cesar",
    );
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("is there for admins too", async () => {
    login.permissions = ["admin_all_access"];
    await openMenu();
    expect(
      screen.getByRole("link", { name: "User details" }),
    ).toBeInTheDocument();
  });

  it("is not there for anyone else", async () => {
    login.permissions = ["can_moderate_users"];
    await openMenu();
    expect(screen.queryByRole("link", { name: "User details" })).toBeNull();
  });
});

describe("the user menu's Report this message entry", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.replaceChildren(root);
    login.permissions = [];
    login.loggedIn = true;
  });

  const openMenuOf = async (userID: string, reportMessage?: () => void) => {
    render(
      <MemoryRouter>
        <UsernameWithContext
          username="cesar"
          userID={userID}
          omitFriend
          omitBlock
          reportMessage={reportMessage}
        />
      </MemoryRouter>,
    );
    await userEvent.click(screen.getByText("cesar"));
    await screen.findByRole("link", { name: "View profile" });
  };

  it("reports another person's chat message", async () => {
    const report = vi.fn();
    await openMenuOf("cesar-id", report);
    await userEvent.click(screen.getByText("Report this message"));
    expect(report).toHaveBeenCalledTimes(1);
  });

  it("is not on a name that is not on a chat message", async () => {
    await openMenuOf("cesar-id");
    expect(screen.queryByText("Report this message")).toBeNull();
  });

  it("is on one's own message too", async () => {
    const report = vi.fn();
    await openMenuOf("me-id", report);
    await userEvent.click(screen.getByText("Report this message"));
    expect(report).toHaveBeenCalledTimes(1);
  });

  it("is not there for those not signed in", async () => {
    login.loggedIn = false;
    await openMenuOf("cesar-id", vi.fn());
    expect(screen.queryByText("Report this message")).toBeNull();
  });
});
