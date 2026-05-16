// @ts-nocheck — @solidjs/start/config API changed in v2; config shape still valid at runtime
import { defineConfig } from "@solidjs/start/config"

export default defineConfig({
  middleware: "./src/middleware.ts",
  vite: {
    server: {
      allowedHosts: true,
    },
    build: {
      rollupOptions: {
        external: ["cloudflare:workers"],
      },
      minify: false,
    },
  },
  server: {
    compatibilityDate: "2024-09-19",
    preset: "cloudflare_module",
    cloudflare: {
      nodeCompat: true,
    },
  },
})
