import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UsernameWithContext } from "./usernameWithContext";

const login = vi.hoisted(() => ({ permissions: [] as string[] }));

vi.mock("../store/store", () => ({
  useContextMatchContext: () => ({ handleContextMatches: [] }),
  useLoginStateStoreContext: () => ({
    loginState: {
      loggedIn: true,
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
