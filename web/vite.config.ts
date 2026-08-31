import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(() => ({
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      strictPort: true,
      proxy: { "/api": "http://localhost:8080", "/health": "http://localhost:8080" },
    },
    build: { target: "es2022", sourcemap: true, cssCodeSplit: true },
}));
