import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import { editorValidatorPlugin } from './editorValidatorPlugin'

export default defineConfig({
  plugins: [react(), editorValidatorPlugin()],
  base: './',
  server: {
    host: '127.0.0.1',
    port: 5173,
    strictPort: true,
    origin: 'http://127.0.0.1:5173',
    cors: { origin: /^http:\/\/(localhost|127\.0\.0\.1)(:\d+)?$/ },
  },
  build: { outDir: '../internal/webui/dist', emptyOutDir: true },
})
