import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [react(), wails("./bindings")],
  onLog(level, msg) {
    // Suppress known eval warnings from web-tree-sitter (Emscripten WASM glue code).
    if (level === 'warn' && msg.includes('web-tree-sitter') && msg.includes('eval')) {
      return false
    }
  },
});
