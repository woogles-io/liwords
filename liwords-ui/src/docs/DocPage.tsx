import React, { useLayoutEffect, useRef } from "react";
import { Typography, Grid, Select } from "antd";
import { Link, useParams, useNavigate, Navigate } from "react-router";
import { TopBar } from "../navigation/topbar";
import { manualById } from "./registry";
import "./docs.scss";

/**
 * /docs/:manualId/:sectionId — one manual, one chapter at a time.
 *
 * Chapters get their own URL rather than living behind in-page state so they
 * can be linked to: from the help modals, from a Discord reply, from a support
 * answer. That is most of the value of having the manual on a page at all.
 */
export const DocPage: React.FC = () => {
  const { manualId, sectionId } = useParams();
  const navigate = useNavigate();
  const screens = Grid.useBreakpoint();
  const bodyRef = useRef<HTMLDivElement>(null);

  const manual = manualById(manualId);

  // Long chapters, one scroll container: without this, following a link to a
  // chapter would drop you wherever the previous one was scrolled to.
  useLayoutEffect(() => {
    bodyRef.current?.scrollTo?.({ top: 0 });
    window.scrollTo({ top: 0 });
  }, [manualId, sectionId]);

  if (!manual) {
    return <Navigate to="/docs" replace />;
  }

  const active =
    manual.sections.find((s) => s.key === sectionId) ?? manual.sections[0];

  // Bare /docs/<manual> resolves to the first chapter, so the URL always names
  // what is on screen and is safe to copy out of the address bar.
  if (!sectionId) {
    return <Navigate to={`/docs/${manual.id}/${active.key}`} replace />;
  }

  const Body = active.body;

  return (
    <>
      <TopBar />
      <div className="docs-page">
        <div className="docs-breadcrumb">
          <Link to="/docs">Documentation</Link>
          <span> / </span>
          <span>{manual.title}</span>
        </div>
        <Typography.Title level={2} className="doc-h">
          {manual.title}
        </Typography.Title>

        <div className="docs-layout">
          {screens.md ? (
            <nav className="docs-nav" aria-label="Chapters">
              {manual.sections.map((s) => (
                <Link
                  key={s.key}
                  to={`/docs/${manual.id}/${s.key}`}
                  className={
                    s.key === active.key
                      ? "docs-nav-item active"
                      : "docs-nav-item"
                  }
                >
                  {s.label}
                </Link>
              ))}
            </nav>
          ) : (
            <Select
              value={active.key}
              onChange={(k) => navigate(`/docs/${manual.id}/${k}`)}
              className="docs-nav-select"
              options={manual.sections.map((s) => ({
                value: s.key,
                label: s.label,
              }))}
            />
          )}

          <article className="doc-body docs-article" ref={bodyRef}>
            <Body />
          </article>
        </div>
      </div>
    </>
  );
};
