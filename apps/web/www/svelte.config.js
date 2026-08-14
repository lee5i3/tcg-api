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
      // Every route is fully prerendered (see src/routes/+layout.ts);
      // 404.html only handles unknown URLs (nginx error_page / S3 error doc).
      fallback: "404.html",
      precompress: false,
      strict: true,
    }),
  },
};

export default config;
