import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { cloudflare } from '@cloudflare/vite-plugin';

// https://vitejs.dev/config/
export default defineConfig(({ command }) => {
  if (command === 'serve') {
    // Development server configuration
    return {
      plugins: [react()],
      server: {
        proxy: {
          '/ws': {
            target: 'http://localhost:8080',
            ws: true,
          },
        },
      },
    };
  } else {
    // Build configuration
    return {
      plugins: [react(), cloudflare()],
    };
  }
});