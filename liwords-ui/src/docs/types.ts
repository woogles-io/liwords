import type React from "react";

/** One chapter of a manual. Rendered identically on the page and in a modal. */
export type DocSection = {
  /** URL segment: /docs/<manual>/<key>. Stable — people bookmark these. */
  key: string;
  /** Nav label. Kept short; the chapter's own <H> carries the full title. */
  label: string;
  body: React.FC;
};

export type Manual = {
  /** URL segment: /docs/<id>. Stable. */
  id: string;
  title: string;
  /** One sentence for the index card. Say what the reader will be able to do. */
  blurb: string;
  /** Who it's for, so people can skip the ones that aren't about them. */
  audience: string;
  sections: DocSection[];
};
