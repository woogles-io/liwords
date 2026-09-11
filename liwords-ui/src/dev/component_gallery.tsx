import React, { useState } from "react";
import {
  Alert as AntAlert,
  Button as AntButton,
  Card as AntCard,
  Dropdown as AntDropdown,
  Input as AntInput,
  InputNumber as AntInputNumber,
  Modal as AntModal,
  Select as AntSelect,
  Table as AntTable,
  Tabs as AntTabs,
  Tag as AntTag,
  Tooltip as AntTooltip,
} from "antd";
import {
  Alert,
  Badge,
  Box,
  Button,
  Card,
  Group,
  Menu,
  Modal,
  NumberInput,
  PasswordInput,
  Select,
  Stack,
  Table,
  Tabs,
  Text,
  Textarea,
  TextInput,
  Title,
  Tooltip,
} from "@mantine/core";

/**
 * Side-by-side reference for the antd -> Mantine migration.
 *
 * Phase 2 translates ~322 lines of antd reskin mixins from base.scss into
 * Mantine component theme overrides. Without something rendering both, that
 * work is written blind and its mistakes only surface later, tangled up with
 * whichever area is being migrated at the time.
 *
 * This page renders each component both ways so the theme can be tuned against
 * the thing it is supposed to match. It inherits the app's colour mode, so
 * toggling dark mode in Settings exercises the real path rather than a mock.
 *
 * Dev-only -- App.tsx does not register the route in production builds. Delete
 * this directory in Phase 9 along with antd.
 */

type SpecimenProps = {
  name: string;
  note?: string;
  antd: React.ReactNode;
  mantine: React.ReactNode;
};

// Layout uses Mantine style props rather than a stylesheet, so this module has
// no side-effect import and tree-shakes cleanly out of production builds.
// Nothing here styles the specimens themselves: the point is to see each
// library's real output, so cosmetics applied here would be lying to us.
const Pane = ({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) => (
  <Box
    style={{
      border: "1px dashed var(--woogles-color-gray-subtle)",
      borderRadius: 4,
      padding: 12,
      minWidth: 0,
    }}
  >
    <Text
      size="10px"
      tt="uppercase"
      c="dimmed"
      mb={8}
      style={{ letterSpacing: "0.12em" }}
    >
      {label}
    </Text>
    <Group gap={8} align="center" wrap="wrap">
      {children}
    </Group>
  </Box>
);

const Specimen = ({ name, note, antd, mantine }: SpecimenProps) => (
  <Box
    style={{
      display: "grid",
      gridTemplateColumns: "160px 1fr 1fr",
      gap: 16,
      alignItems: "start",
      padding: "20px 0",
      borderTop: "1px solid var(--woogles-color-gray-subtle)",
    }}
  >
    <Box pt={4}>
      <Text fw={700}>{name}</Text>
      {note && (
        <Text size="xs" c="dimmed" mt={2}>
          {note}
        </Text>
      )}
    </Box>
    <Pane label="antd">{antd}</Pane>
    <Pane label="mantine">{mantine}</Pane>
  </Box>
);

const rows = [
  { key: "1", player: "cesar", rating: 1842, result: "Win" },
  { key: "2", player: "jesse", rating: 1790, result: "Loss" },
];

export const ComponentGallery = React.memo(() => {
  const [antModalOpen, setAntModalOpen] = useState(false);
  const [mantineModalOpen, setMantineModalOpen] = useState(false);

  return (
    <Stack
      gap={0}
      p={24}
      maw={1400}
      mx="auto"
      c="var(--woogles-color-gray-extreme)"
    >
      <Title order={2}>Component gallery</Title>
      <Text c="dimmed" size="sm">
        antd on the left, Mantine on the right. Toggle dark mode in Settings to
        compare both schemes. Dev builds only.
      </Text>

      <Specimen
        name="Button"
        note="base.scss @mixin button, 104 lines"
        antd={
          <>
            <AntButton>Default</AntButton>
            <AntButton type="primary">Primary</AntButton>
            <AntButton disabled>Disabled</AntButton>
            <AntButton danger>Danger</AntButton>
          </>
        }
        mantine={
          <>
            <Button variant="default">Default</Button>
            <Button>Primary</Button>
            <Button disabled>Disabled</Button>
            <Button color="red">Danger</Button>
          </>
        }
      />

      <Specimen
        name="Card"
        note="189 .ant-card selectors in SCSS"
        antd={
          <AntCard title="Card title" extra={<a href="#gallery">More</a>}>
            Card body content.
          </AntCard>
        }
        mantine={
          <Card withBorder>
            <Group justify="space-between" mb="xs">
              <Text fw={700}>Card title</Text>
              <Text size="sm">More</Text>
            </Group>
            Card body content.
          </Card>
        }
      />

      <Specimen
        name="Text inputs"
        antd={
          <>
            <AntInput placeholder="Text" />
            <AntInputNumber placeholder="Number" />
            <AntInput.Password placeholder="Password" />
            <AntInput.TextArea placeholder="Textarea" rows={2} />
          </>
        }
        mantine={
          <>
            <TextInput placeholder="Text" />
            <NumberInput placeholder="Number" />
            <PasswordInput placeholder="Password" />
            <Textarea placeholder="Textarea" rows={2} />
          </>
        }
      />

      <Specimen
        name="Select"
        note="184 Select.Option children to convert"
        antd={
          <AntSelect
            defaultValue="csw24"
            options={[
              { value: "csw24", label: "CSW24" },
              { value: "nwl23", label: "NWL23" },
            ]}
          />
        }
        mantine={
          <Select
            defaultValue="csw24"
            data={[
              { value: "csw24", label: "CSW24" },
              { value: "nwl23", label: "NWL23" },
            ]}
          />
        }
      />

      <Specimen
        name="Tabs"
        note="base.scss @mixin tabs, 44 lines"
        antd={
          <AntTabs
            defaultActiveKey="a"
            items={[
              { key: "a", label: "Chat", children: "Chat pane" },
              { key: "b", label: "Players", children: "Players pane" },
            ]}
          />
        }
        mantine={
          <Tabs defaultValue="a">
            <Tabs.List>
              <Tabs.Tab value="a">Chat</Tabs.Tab>
              <Tabs.Tab value="b">Players</Tabs.Tab>
            </Tabs.List>
            <Tabs.Panel value="a" pt="xs">
              Chat pane
            </Tabs.Panel>
            <Tabs.Panel value="b" pt="xs">
              Players pane
            </Tabs.Panel>
          </Tabs>
        }
      />

      <Specimen
        name="Tag / Badge"
        note="custom 16-colour palette in App.scss"
        antd={
          <>
            <AntTag className="ant-tag-win">Win</AntTag>
            <AntTag className="ant-tag-loss">Loss</AntTag>
            <AntTag className="ant-tag-bye">Bye</AntTag>
            <AntTag className="ant-tag-repeat">Repeat</AntTag>
          </>
        }
        mantine={
          <>
            <Badge color="green">Win</Badge>
            <Badge color="red">Loss</Badge>
            <Badge color="gray">Bye</Badge>
            <Badge color="blue">Repeat</Badge>
          </>
        }
      />

      <Specimen
        name="Alert"
        antd={
          <>
            <AntAlert message="Info message" type="info" />
            <AntAlert message="Error message" type="error" />
          </>
        }
        mantine={
          <>
            <Alert>Info message</Alert>
            <Alert color="red">Error message</Alert>
          </>
        }
      />

      <Specimen
        name="Tooltip"
        note="33 files"
        antd={
          <AntTooltip title="Tooltip body" open>
            <AntButton>Hover</AntButton>
          </AntTooltip>
        }
        mantine={
          <Tooltip label="Tooltip body" opened>
            <Button variant="default">Hover</Button>
          </Tooltip>
        }
      />

      <Specimen
        name="Dropdown / Menu"
        antd={
          <AntDropdown
            menu={{
              items: [
                { key: "1", label: "Resign" },
                { key: "2", label: "Offer draw" },
              ],
            }}
          >
            <AntButton>Actions</AntButton>
          </AntDropdown>
        }
        mantine={
          <Menu>
            <Menu.Target>
              <Button variant="default">Actions</Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item>Resign</Menu.Item>
              <Menu.Item>Offer draw</Menu.Item>
            </Menu.Dropdown>
          </Menu>
        }
      />

      <Specimen
        name="Table"
        note="38 instances; Mantine's is presentational only"
        antd={
          <AntTable
            size="small"
            pagination={false}
            dataSource={rows}
            columns={[
              { title: "Player", dataIndex: "player", key: "player" },
              { title: "Rating", dataIndex: "rating", key: "rating" },
              { title: "Result", dataIndex: "result", key: "result" },
            ]}
          />
        }
        mantine={
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Player</Table.Th>
                <Table.Th>Rating</Table.Th>
                <Table.Th>Result</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {rows.map((r) => (
                <Table.Tr key={r.key}>
                  <Table.Td>{r.player}</Table.Td>
                  <Table.Td>{r.rating}</Table.Td>
                  <Table.Td>{r.result}</Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        }
      />

      <Specimen
        name="Modal"
        note="base.scss @mixin modal, 56 lines; also mobile full-screen"
        antd={
          <>
            <AntButton onClick={() => setAntModalOpen(true)}>
              Open antd modal
            </AntButton>
            <AntModal
              title="Modal title"
              open={antModalOpen}
              onCancel={() => setAntModalOpen(false)}
              onOk={() => setAntModalOpen(false)}
            >
              Modal body content.
            </AntModal>
          </>
        }
        mantine={
          <>
            <Button variant="default" onClick={() => setMantineModalOpen(true)}>
              Open Mantine modal
            </Button>
            <Modal
              title="Modal title"
              opened={mantineModalOpen}
              onClose={() => setMantineModalOpen(false)}
            >
              Modal body content.
            </Modal>
          </>
        }
      />
    </Stack>
  );
});

export default ComponentGallery;
