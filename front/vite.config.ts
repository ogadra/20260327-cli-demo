import { defineConfig } from "vite-plus";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
  lint: { options: { typeAware: true, typeCheck: true } },
  plugins: [react()],
  server: {
    host: true,
    proxy: {
      "/api": {
        target: "http://localhost:3000",
        changeOrigin: true,
      },
    },
  },
});
