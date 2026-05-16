import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  resolve: {
    dedupe: ["react", "react-dom"],
  },
  server: {
    proxy: {
      "/admin": "http://localhost:8080",
      "/api": "http://localhost:8080",
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['axios', 'notistack', 'react', 'react-dom', 'react-router-dom', '@mui/material', '@emotion/react', '@emotion/styled'],
        }
      }
    }
  }
});