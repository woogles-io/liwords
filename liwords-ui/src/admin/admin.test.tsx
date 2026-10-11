import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { Admin } from "./admin";

vi.mock("../store/store", () => ({
  useLoginStateStoreContext: () => ({
    loginState: { permissions: ["can_see_private_user_data"] },
  }),
}));
vi.mock("../navigation/topbar", () => ({ TopBar: () => null }));
vi.mock("./user_details", () => ({ UserDetails: () => "user details page" }));
vi.mock("./tourney_editor", () => ({ TourneyEditor: () => null }));
vi.mock("./announcement_editor", () => ({ AnnouncementEditor: () => null }));
vi.mock("./puzzle_generator", () => ({ PuzzleGenerator: () => null }));
vi.mock("./perms_and_roles", () => ({ PermsAndRoles: () => null }));
vi.mock("./user_badges", () => ({ UserBadges: () => null }));
vi.mock("./verification_queue", () => ({ VerificationQueue: () => null }));
vi.mock("./manual_org_assignment", () => ({
  ManualOrgAssignment: () => null,
}));
vi.mock("./analysis_stats", () => ({ AnalysisStats: () => null }));

const renderAt = (url: string) =>
  render(
    <MemoryRouter initialEntries={[url]}>
      <Admin />
    </MemoryRouter>,
  );

describe("Admin", () => {
  it("opens the tab that ?tab= names", () => {
    renderAt("/admin?tab=user-details&username=cesar");
    expect(screen.getByText("user details page")).toBeInTheDocument();
  });

  it("opens no tab without ?tab=", () => {
    renderAt("/admin");
    expect(screen.queryByText("user details page")).toBeNull();
  });
});
