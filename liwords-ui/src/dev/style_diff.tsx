import React, { useCallback, useState } from "react";
import { Badge, Box, Button, Group, Table, Text } from "@mantine/core";

/**
 * Computed-style differ for the component gallery.
 *
 * "Looks different" is not actionable, and eyeballing two panes does not tell
 * you *which* of forty properties is off. This measures both specimens in the
 * live DOM and lists only the properties that disagree, so matching a component
 * becomes a finite checklist rather than a hunt.
 *
 * Measured against the real rendered output, so it accounts for the whole
 * cascade -- Mantine's own CSS, our theme overrides, the antd reskin mixins and
 * the layer ordering all included.
 */

/** Properties worth matching. Ordered roughly by how visible a mismatch is. */
const TRACKED = [
  "height",
  "min-height",
  "padding-top",
  "padding-right",
  "padding-bottom",
  "padding-left",
  "margin-top",
  "margin-right",
  "margin-bottom",
  "margin-left",
  "background-color",
  "color",
  "border-top-width",
  "border-top-style",
  "border-top-color",
  "border-radius",
  "font-family",
  "font-size",
  "font-weight",
  "line-height",
  "letter-spacing",
  "text-transform",
  "box-shadow",
  "opacity",
] as const;

export type Measurable = {
  /** CSS selector for the element to measure inside each pane. */
  antd: string;
  mantine: string;
};

type Diff = { prop: string; antd: string; mantine: string };

function measure(root: HTMLElement | null, selector: string) {
  if (!root) return null;
  const el = root.querySelector(selector);
  if (!(el instanceof HTMLElement)) return null;
  const cs = getComputedStyle(el);
  const out: Record<string, string> = {};
  for (const p of TRACKED) out[p] = cs.getPropertyValue(p).trim();
  return out;
}

export function useStyleDiff(target: Measurable | undefined) {
  const [antdRoot, setAntdRoot] = useState<HTMLElement | null>(null);
  const [mantineRoot, setMantineRoot] = useState<HTMLElement | null>(null);
  const [diffs, setDiffs] = useState<Diff[] | null>(null);
  const [missing, setMissing] = useState<string | null>(null);

  const run = useCallback(() => {
    if (!target) return;
    const a = measure(antdRoot, target.antd);
    const m = measure(mantineRoot, target.mantine);
    if (!a || !m) {
      setMissing(
        !a
          ? `no match for "${target.antd}"`
          : `no match for "${target.mantine}"`,
      );
      setDiffs(null);
      return;
    }
    setMissing(null);
    setDiffs(
      TRACKED.filter((p) => a[p] !== m[p]).map((p) => ({
        prop: p,
        antd: a[p],
        mantine: m[p],
      })),
    );
  }, [antdRoot, mantineRoot, target]);

  return {
    setAntdRoot,
    setMantineRoot,
    diffs,
    missing,
    run,
    enabled: !!target,
  };
}

export const StyleDiffReport = ({
  diffs,
  missing,
  run,
  enabled,
}: ReturnType<typeof useStyleDiff>) => {
  if (!enabled) return null;
  return (
    <Box mt={12}>
      <Group gap={8} align="center" mb={diffs || missing ? 8 : 0}>
        <Button size="compact-xs" variant="default" onClick={run}>
          Measure
        </Button>
        {diffs && diffs.length === 0 && (
          <Badge color="green" size="sm">
            identical on {TRACKED.length} tracked properties
          </Badge>
        )}
        {diffs && diffs.length > 0 && (
          <Badge color="red" size="sm">
            {diffs.length} differ
          </Badge>
        )}
        {missing && (
          <Text size="xs" c="red">
            {missing}
          </Text>
        )}
      </Group>
      {diffs && diffs.length > 0 && (
        <Table
          withTableBorder
          fz={11}
          verticalSpacing={2}
          horizontalSpacing={6}
        >
          <Table.Thead>
            <Table.Tr>
              <Table.Th>property</Table.Th>
              <Table.Th>antd</Table.Th>
              <Table.Th>mantine</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {diffs.map((d) => (
              <Table.Tr key={d.prop}>
                <Table.Td>{d.prop}</Table.Td>
                <Table.Td>{d.antd}</Table.Td>
                <Table.Td>{d.mantine}</Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
    </Box>
  );
};
