import path from "path"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react-swc'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, "")
  const proxyTarget = env.VITE_DEV_PROXY_TARGET || 'http://localhost:3000'

  return {
    plugins: [react(), tailwindcss()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
          secure: true,
          rewrite: (path) => path.replace(/^\/api/, '/api'),
        },
      },
    },
    build: {
      // ECharts is isolated into its own vendor chunk; keep warnings focused on app/page chunk regressions.
      chunkSizeWarningLimit: 900,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (!id.includes('node_modules')) return

            if (id.includes('/node_modules/echarts/')) return 'vendor-echarts'
            if (id.includes('/node_modules/zrender/')) return 'vendor-zrender'
            if (id.includes('/node_modules/react-syntax-highlighter/')) return 'vendor-syntax-highlighter'
            if (id.includes('/node_modules/react-markdown/') || id.includes('/node_modules/remark-gfm/') || id.includes('/node_modules/markdown-table/') || id.includes('/node_modules/mdast-util-') || id.includes('/node_modules/micromark') || id.includes('/node_modules/unist-') || id.includes('/node_modules/vfile')) {
              return 'vendor-markdown'
            }
            if (id.includes('/node_modules/@tanstack/')) return 'vendor-tanstack'
            if (id.includes('/node_modules/@radix-ui/')) return 'vendor-radix'
            if (id.includes('/node_modules/motion/')) return 'vendor-motion'
            if (id.includes('/node_modules/lucide-react/')) return 'vendor-icons'
            if (id.includes('/node_modules/react/') || id.includes('/node_modules/react-dom/') || id.includes('/node_modules/react-router/')) return 'vendor-react'
          },
        },
      },
    },
  }
})
