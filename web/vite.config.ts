import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Прокси решает CORS в dev: SPA живёт на :5173, но ходит на свои бэкенды
// относительными путями. /api → link service, /ws → dashboard service.
const proxy = {
  "/api": {
    target: "http://localhost:8081",
    changeOrigin: true,
  },
  "/ws": {
    target: "ws://localhost:8082",
    ws: true,
  },
};

export default defineConfig({
  plugins: [react()],
  server: { proxy },
  preview: { proxy },
});
