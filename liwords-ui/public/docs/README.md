Screenshots for the /docs manuals.

Only for things the app cannot render itself — in practice, OBS's own
interface. Anything that is Woogles UI should be embedded as the live
component instead (see DocImage in src/docs/prose.tsx for why).

Referenced by src/docs/manuals/*. A missing file renders a visible
placeholder rather than a broken image, so it is safe to add these later.

Expected files:
  obs-add-browser-source.png         Sources panel, + menu open, Browser highlighted
  obs-browser-source-properties.png  Properties dialog: URL, width, height
  obs-scene-layout.png               A finished scene with overlays over a board camera
