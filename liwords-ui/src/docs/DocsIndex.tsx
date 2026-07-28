import React from "react";
import { Card, Typography } from "antd";
import { Link } from "react-router";
import { TopBar } from "../navigation/topbar";
import { MANUALS } from "./registry";
import "./docs.scss";

/** /docs — the list of manuals. */
export const DocsIndex: React.FC = () => (
  <>
    <TopBar />
    <div className="docs-page">
      <Typography.Title level={2} className="doc-h">
        Documentation
      </Typography.Title>
      <Typography.Paragraph className="doc-p">
        Manuals for features on Woogles. For strategy, announcements and
        how-to-play guides, see the{" "}
        <a href="https://blog.woogles.io/guides">blog guides</a>.
      </Typography.Paragraph>

      <div className="docs-card-grid">
        {MANUALS.map((m) => (
          <Link key={m.id} to={`/docs/${m.id}`} className="docs-card-link">
            <Card size="small" hoverable className="docs-card">
              <Typography.Title level={5} className="doc-h">
                {m.title}
              </Typography.Title>
              <div className="docs-card-audience">{m.audience}</div>
              <Typography.Paragraph className="doc-p">
                {m.blurb}
              </Typography.Paragraph>
              <div className="docs-card-chapters">
                {m.sections.length} chapters
              </div>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  </>
);
