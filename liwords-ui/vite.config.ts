import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import ViteSassPlugin from 'vite-plugin-sass';
import wasm from 'vite-plugin-wasm';
import topLevelAwait from 'vite-plugin-top-level-await';

export default defineConfig({
  // depending on your application, base can also be "/"
  base: '',
  plugins: [react(), ViteSassPlugin(), wasm(), topLevelAwait()],
  server: {
    // this sets a default port to 3000
    port: 3000,
  },
});
