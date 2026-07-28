import type { Manual } from "./types";
import { streamingManual } from "./manuals/streaming";

/**
 * Every manual on /docs. Adding one means writing its chapters and appending it
 * here — the index page, the routing and the deep links all read from this.
 *
 * These live in the app rather than on the blog on purpose. A manual makes
 * falsifiable claims about how the code behaves, so it belongs in the same
 * repository and the same pull request as the behaviour it describes, where a
 * reviewer changing that behaviour can see the documentation that is about to
 * become wrong. Editorial content — strategy, announcements, how to play — has
 * no such coupling and is better off on the blog.
 */
export const MANUALS: Manual[] = [streamingManual];

export const manualById = (id: string | undefined): Manual | undefined =>
  MANUALS.find((m) => m.id === id);
