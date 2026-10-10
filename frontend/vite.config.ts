/// <reference types="vitest/config" />
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [svelte()],
  // tests mount components in jsdom, which needs the client build of svelte
  resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
  // tests read the Go sources next door to check that every code is translated
  server: process.env.VITEST ? { fs: { allow: ['..'] } } : undefined,
  build: {
    target: 'es2022',
    // fonts and images stay separate hashed files, the icon is the only small asset
    assetsInlineLimit: 0,
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
    restoreMocks: true,
  },
})
