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

/**
 * One like-for-like comparison. Selectors should target `data-cmp` attributes
 * the gallery sets itself rather than either library's internal class names --
 * an earlier version matched antd's primary button against Mantine's default
 * one and reported four differences that were purely the wrong pairing.
 */
export type Measurable = {
  label: string;
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

type Result = { label: string; diffs: Diff[] | null; missing: string | null };

export function useStyleDiff(targets: Measurable[] | undefined) {
  const [antdRoot, setAntdRoot] = useState<HTMLElement | null>(null);
  const [mantineRoot, setMantineRoot] = useState<HTMLElement | null>(null);
  const [results, setResults] = useState<Result[] | null>(null);

  const run = useCallback(() => {
    if (!targets) return;
    setResults(
      targets.map((t): Result => {
        const a = measure(antdRoot, t.antd);
        const m = measure(mantineRoot, t.mantine);
        if (!a || !m) {
          return {
            label: t.label,
            diffs: null,
            missing: `no match for "${a ? t.mantine : t.antd}"`,
          };
        }
        return {
          label: t.label,
          missing: null,
          diffs: TRACKED.filter((prop) => a[prop] !== m[prop]).map((prop) => ({
            prop,
            antd: a[prop],
            mantine: m[prop],
          })),
        };
      }),
    );
  }, [antdRoot, mantineRoot, targets]);

  return {
    setAntdRoot,
    setMantineRoot,
    results,
    run,
    enabled: !!targets?.length,
  };
}

export const StyleDiffReport = ({
  results,
  run,
  enabled,
}: ReturnType<typeof useStyleDiff>) => {
  if (!enabled) return null;
  return (
    <Box mt={12}>
      <Button size="compact-xs" variant="default" onClick={run} mb={8}>
        Measure
      </Button>
      {results?.map((r) => (
        <Box key={r.label} mb={8}>
          <Group gap={6} align="center" mb={4}>
            <Text size="xs" fw={700}>
              {r.label}
            </Text>
            {r.diffs && r.diffs.length === 0 && (
              <Badge color="green" size="xs">
                identical on {TRACKED.length} properties
              </Badge>
            )}
            {r.diffs && r.diffs.length > 0 && (
              <Badge color="red" size="xs">
                {r.diffs.length} differ
              </Badge>
            )}
            {r.missing && (
              <Text size="xs" c="red">
                {r.missing}
              </Text>
            )}
          </Group>
          {r.diffs && r.diffs.length > 0 && (
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
                {r.diffs.map((d) => (
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
      ))}
    </Box>
  );
};
