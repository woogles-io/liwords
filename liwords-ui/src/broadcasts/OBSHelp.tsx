import React, { useLayoutEffect, useRef, useState } from "react";
import { Button, Modal, Tabs } from "antd";
import { QuestionCircleOutlined, ExportOutlined } from "@ant-design/icons";
import { Link } from "react-router";
import { streamingManual } from "../docs/manuals/streaming";
import type { StreamingSection } from "../docs/manuals/streaming";
import "../docs/docs.scss";
import "./obs_help.scss";

/**
 * In-context help for the OBS slot features.
 *
 * The chapters themselves live in src/docs/manuals/streaming.tsx and are also
 * served at /docs/streaming — this is the same document in a modal, opened from
 * wherever a slot is configured and landing on the chapter that matches. One
 * source of chapters means the page and the modal cannot drift apart.
 */
export type OBSHelpSection = StreamingSection;

const SECTIONS = streamingManual.sections;

type Props = {
  /** Chapter to open on. Defaults to the overview. */
  section?: OBSHelpSection;
  /** Renders a bare "?" icon instead of a labelled button. */
  iconOnly?: boolean;
};

export const OBSHelpButton: React.FC<Props> = ({
  section = "overview",
  iconOnly = false,
}) => {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState<string>(section);
  const wrapRef = useRef<HTMLDivElement>(null);

  const show = () => {
    // Re-seed on every open so the button always lands on its own chapter,
    // even after the reader has wandered off into another one.
    setActive(section);
    setOpen(true);
  };

  // Chapters share one scroll container, so switching from the bottom of a long
  // chapter would otherwise drop you into the middle of the next one. Reset on
  // every chapter change and on reopen.
  useLayoutEffect(() => {
    // Reset both candidates: the tab content pane is the intended scroller, but
    // if the modal body ends up taking the overflow instead, that needs the
    // same treatment and resetting an unscrolled element is a no-op.
    const holder = wrapRef.current?.querySelector(".ant-tabs-content-holder");
    if (holder) holder.scrollTop = 0;
    const modalBody = wrapRef.current?.closest(".ant-modal-body");
    if (modalBody) modalBody.scrollTop = 0;
  }, [active, open]);

  return (
    <>
      <button
        type="button"
        className="obs-help-trigger"
        onClick={show}
        aria-label="OBS streaming help"
        title="OBS streaming help"
      >
        <QuestionCircleOutlined />
        {iconOnly ? null : <span>Help</span>}
      </button>
      <Modal
        open={open}
        title={streamingManual.title}
        width={900}
        zIndex={1200}
        onCancel={() => setOpen(false)}
        footer={
          <div className="obs-help-footer">
            {/* Escape hatch to the full page, so a reader can bookmark the
                chapter or send it to somebody. */}
            <Link
              to={`/docs/${streamingManual.id}/${active}`}
              onClick={() => setOpen(false)}
            >
              Open full manual <ExportOutlined />
            </Link>
            <Button onClick={() => setOpen(false)}>Close</Button>
          </div>
        }
      >
        <div ref={wrapRef}>
          <Tabs
            tabPosition="left"
            activeKey={active}
            onChange={setActive}
            className="obs-help-tabs"
            items={SECTIONS.map((s) => ({
              key: s.key,
              label: s.label,
              children: (
                <div className="doc-body">
                  <s.body />
                </div>
              ),
            }))}
          />
        </div>
      </Modal>
    </>
  );
};
