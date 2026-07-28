import React, { useState } from "react";
import { Button, Input, Popconfirm, Space, Tag, Typography } from "antd";
import { UserAddOutlined } from "@ant-design/icons";

type Props = {
  title: string;
  hint?: string;
  usernames: string[];
  onAdd: (username: string) => void;
  onRemove: (username: string) => void;
  adding?: boolean;
  /**
   * When set, removing asks for confirmation with this message (the username is
   * interpolated by the caller). Directors use it; annotators are cheap enough
   * to re-add that confirming would just be friction.
   */
  confirmRemove?: (username: string) => string;
  placeholder?: string;
  emptyText?: string;
};

/**
 * A list of usernames as removable chips, plus an input to add one.
 *
 * Annotators and directors were ~80 lines of near-identical JSX differing only
 * in which mutation they called and whether removal confirmed first.
 */
export const UserListSection: React.FC<Props> = ({
  title,
  hint,
  usernames,
  onAdd,
  onRemove,
  adding = false,
  confirmRemove,
  placeholder = "Username",
  emptyText = "Nobody yet.",
}) => {
  const [draft, setDraft] = useState("");

  const submit = () => {
    const name = draft.trim();
    if (!name) return;
    onAdd(name);
    setDraft("");
  };

  return (
    <div>
      <div className="panel-section-title">{title}</div>
      {hint && <Typography.Text className="panel-hint">{hint}</Typography.Text>}

      {usernames.length === 0 ? (
        <Typography.Text className="panel-empty">{emptyText}</Typography.Text>
      ) : (
        <div className="user-list-tags">
          {usernames.map((u) =>
            confirmRemove ? (
              <Popconfirm
                key={u}
                title={confirmRemove(u)}
                onConfirm={() => onRemove(u)}
              >
                {/* preventDefault so the chip's own close doesn't fire before
                    the confirmation is answered. */}
                <Tag closable onClose={(e) => e.preventDefault()}>
                  {u}
                </Tag>
              </Popconfirm>
            ) : (
              <Tag key={u} closable onClose={() => onRemove(u)}>
                {u}
              </Tag>
            ),
          )}
        </div>
      )}

      <Space.Compact style={{ width: "100%" }}>
        <Input
          size="small"
          placeholder={placeholder}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onPressEnter={submit}
        />
        <Button
          size="small"
          icon={<UserAddOutlined />}
          loading={adding}
          onClick={submit}
          disabled={!draft.trim()}
        >
          Add
        </Button>
      </Space.Compact>
    </div>
  );
};
