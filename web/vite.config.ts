import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const staticDemo = loadEnv(mode, ".", "").VITE_STATIC_DEMO === "true";
  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: staticDemo ? [
        { find: /^react-dom\/test-utils$/, replacement: "preact/test-utils" },
        { find: /^react-dom\/client$/, replacement: "preact/compat/client" },
        { find: /^react-dom$/, replacement: "preact/compat" },
        { find: /^react\/jsx-runtime$/, replacement: "preact/jsx-runtime" },
        { find: /^react$/, replacement: "preact/compat" },
      ] : [],
    },
    server: {
      port: 5173,
      strictPort: true,
      proxy: { "/api": "http://localhost:8080", "/health": "http://localhost:8080" },
    },
    build: { target: "es2022", sourcemap: true, cssCodeSplit: true },
  };
});
