import { pluginReact } from "@rsbuild/plugin-react";
import { pluginSass } from "@rsbuild/plugin-sass";

export default {
  plugins: [pluginReact(), pluginSass()],
  lazyCompilation: false,
  source: {
    entry: {
      embed: "./src/embed/embed-entry.tsx",
    },
  },
  output: {
    // Output to a temp directory first
    distPath: {
      root: "embed-build",
    },
    filename: {
      js: "embed-standalone.js",
    },
    // Create an IIFE (Immediately Invoked Function Expression)
    library: {
      type: "iife",
    },
    // Inline CSS into JS
    injectStyles: true,
    // Inline all assets
    dataUriLimit: 100000,
    // Minimize the output
    minify: false, // Keep unminified for debugging
    // Clean dist before build
    cleanDistPath: false,
    // Force single chunk
    chunkLoadingGlobal: false,
  },
  performance: {
    // Bundle analysis
    bundleAnalyze: false,
  },
  // Force everything into one bundle
  tools: {
    rspack: {
      optimization: {
        splitChunks: false,
        runtimeChunk: false,
      },
    },
  },
  html: {
    // Generate a simple HTML for testing
    template: "./src/embed/demo.html",
  },
  performance: {
    // Bundle analysis
    bundleAnalyze: false,
  },
  server: {
    // Disable dev server for this config
    port: 3001,
  },
};
