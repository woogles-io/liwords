import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import wasm from 'vite-plugin-wasm';
import topLevelAwait from 'vite-plugin-top-level-await';
import browserslistToEsbuild from 'browserslist-to-esbuild';

export default defineConfig({
  plugins: [react(), wasm(), topLevelAwait()],
  server: {
    // this sets a default port to 3000
    port: 3000,
  },
  build: {
    target: browserslistToEsbuild(['>0.2%', 'not dead', 'not op_mini all']),
  },
});
