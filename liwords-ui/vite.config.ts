import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import ViteSassPlugin from 'vite-plugin-sass';

export default defineConfig({
  // depending on your application, base can also be "/"
  base: '',
  plugins: [react(), ViteSassPlugin()],
  server: {
    // this ensures that the browser opens upon server start
    open: true,
    // this sets a default port to 3000
    port: 3000,
  },
});
