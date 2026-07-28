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

/**
 * A screenshot, for the few things the app cannot render itself — chiefly OBS's
 * own interface.
 *
 * Prefer embedding the real component over a screenshot wherever the subject is
 * Woogles UI: an embed cannot go stale, follows light and dark mode, and needs
 * no binary in the repo. Reach for this only when the subject is somebody
 * else's application.
 *
 * Renders a labelled placeholder when the file is missing, so a manual can be
 * written before its screenshots are taken and a deleted file is obvious rather
 * than silently blank.
 */
export const DocImage: React.FC<{
  src: string;
  alt: string;
  caption?: string;
}> = ({ src, alt, caption }) => {
  const [failed, setFailed] = React.useState(false);

  return (
    <figure className="doc-figure">
      {failed ? (
        <div className="doc-figure-missing">
          Screenshot not available: <code>{src}</code>
        </div>
      ) : (
        <img src={src} alt={alt} onError={() => setFailed(true)} />
      )}
      {caption && <figcaption>{caption}</figcaption>}
    </figure>
  );
};
