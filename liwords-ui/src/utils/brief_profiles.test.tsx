import { render, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { BriefProfiles, useBriefProfile } from "./brief_profiles";

const mock = vi.hoisted(() => ({ getBriefProfiles: vi.fn() }));

vi.mock("./hooks/connect", () => ({
  useClient: () => ({ getBriefProfiles: mock.getBriefProfiles }),
}));

const Name = ({ id }: { id: string }) => {
  const profile = useBriefProfile(id);
  return <span data-testid={id}>{profile?.username ?? ""}</span>;
};

describe("BriefProfiles", () => {
  it("asks for at most 100 users per request", async () => {
    mock.getBriefProfiles.mockImplementation(
      async ({ userIds }: { userIds: Array<string> }) => ({
        response: Object.fromEntries(
          userIds.map((id) => [id, { username: `name-${id}` }]),
        ),
      }),
    );
    const ids = Array.from({ length: 250 }, (_, i) => `u${i}`);
    const { getByTestId } = render(
      <BriefProfiles>
        {ids.map((id) => (
          <Name key={id} id={id} />
        ))}
      </BriefProfiles>,
    );

    await waitFor(() =>
      expect(getByTestId("u249").textContent).toBe("name-u249"),
    );
    const sizes = mock.getBriefProfiles.mock.calls.map(
      ([req]) => req.userIds.length,
    );
    expect(sizes).toEqual([100, 100, 50]);
    expect(getByTestId("u0").textContent).toBe("name-u0");
    expect(getByTestId("u150").textContent).toBe("name-u150");
  });
});
