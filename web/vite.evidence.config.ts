import { fileURLToPath, URL } from "node:url";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  root: fileURLToPath(new URL("./evidence", import.meta.url)),
  plugins: [react(), tailwindcss()],
  define: {
    "import.meta.env.VITE_STATIC_DEMO": JSON.stringify("true"),
    "import.meta.env.VITE_UI_EVIDENCE": JSON.stringify("true"),
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: { "/api": "http://localhost:8080", "/health": "http://localhost:8080" },
  },
  build: {
    target: "es2022",
    rollupOptions: { input: {
      main: fileURLToPath(new URL("./evidence/index.html", import.meta.url)),
      privacyTable: fileURLToPath(new URL("./evidence/privacy-table.html", import.meta.url)),
    } },
    sourcemap: true,
    cssCodeSplit: true,
    outDir: fileURLToPath(new URL("./dist-evidence", import.meta.url)),
    emptyOutDir: true,
  },
});
