import { sveltekit } from "@sveltejs/kit/vite";
import { svelteTesting } from "@testing-library/svelte/vite";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [sveltekit(), svelteTesting()],
  // Expose PUBLIC_-prefixed variables (SvelteKit convention) through
  // import.meta.env alongside Vite's default VITE_ prefix.
  envPrefix: ["VITE_", "PUBLIC_"],
  test: {
    environment: "jsdom",
    include: ["src/**/*.{test,spec}.ts"],
  },
});
