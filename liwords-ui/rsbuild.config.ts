import { pluginEslint } from "@rsbuild/plugin-eslint";
import { pluginReact } from "@rsbuild/plugin-react";
import { pluginSass } from "@rsbuild/plugin-sass";
import { pluginSvgr } from "@rsbuild/plugin-svgr";

export default {
  plugins: [
    pluginEslint({ eslintPluginOptions: { configType: "flat" } }),
    pluginReact(),
    pluginSass(),
    pluginSvgr(),
  ],
  dev: { client: { overlay: false } },
  // rsbuild 2's dev server binds localhost only by default; expose it on all
  // interfaces so the docker-compose proxy (frontend service) can reach it.
  server: { host: "0.0.0.0" },
  html: {
    template: "./index.html",
  },
  source: {
    entry: {
      index: "./src/index.tsx",
      //   embed: "./src/embed/embed-entry.tsx",
    },
  },
};
