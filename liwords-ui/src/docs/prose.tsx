import React from "react";
import { Typography } from "antd";

/**
 * Prose primitives shared by every manual, so chapters look the same whether
 * they are read on /docs or in a help modal, and so a new manual doesn't invent
 * its own typography.
 */

/** Inline code: field names, URLs, config values. */
export const C: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <code className="doc-code">{children}</code>
);

export const P: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <Typography.Paragraph className="doc-p">{children}</Typography.Paragraph>
);

/** Section heading within a chapter. */
export const H: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <Typography.Title level={5} className="doc-h">
    {children}
  </Typography.Title>
);
