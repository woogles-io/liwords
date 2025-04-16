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
  html: {
    template: "./index.html",
  },
  source: {
    entry: {
      index: "./src/index.tsx",
    },
  },
};
