import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      // Keep the build output at <app>/dist so the existing tooling
      // (S3 sync, Dockerfile COPY) stays uniform across all sites.
      pages: "dist",
      assets: "dist",
      // Pure SPA: every route is served the index.html shell and rendered
      // client-side (see src/routes/+layout.ts).
      fallback: "index.html",
      precompress: false,
      strict: true,
    }),
  },
};

export default config;
